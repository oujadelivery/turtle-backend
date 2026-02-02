package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"

	"turtle/config"
	"turtle/graph"
	"turtle/graph/generated"
	"turtle/internal/application/services"
	"turtle/internal/application/usecases"
	"turtle/internal/domain"
	"turtle/internal/infrastructure/cache"
	"turtle/internal/infrastructure/database"
	"turtle/internal/infrastructure/dataloader"
	infraPostgres "turtle/internal/infrastructure/persistence/postgres"
	"turtle/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Print configuration in development
	if cfg.IsDevelopment() {
		cfg.PrintConfig()
	}

	// Connect to PostgreSQL
	err = database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := database.MigrateDatabase(database.GetDB(), cfg.IsDevelopment()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Connect to Redis
	err = cache.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	// ============================================================================
	// INITIALIZE REPOSITORIES (Data Layer)
	// ============================================================================
	userRepo := infraPostgres.NewUserRepository(database.GetDB())
	addressRepo := infraPostgres.NewAddressRepository(database.GetDB())
	otpRepo := infraPostgres.NewOTPRepository(database.GetDB())
	refreshTokenRepo := infraPostgres.NewRefreshTokenRepository(database.GetDB())

	// ============================================================================
	// INITIALIZE SERVICES (Business Logic Layer with DataLoader)
	// ============================================================================
	// Services wrap repositories and provide DataLoader optimization
	userService := services.NewUserService(userRepo)
	addressService := services.NewAddressService(addressRepo)
	
	// Authentication service (already exists, uses repositories directly)
	authService := usecases.NewAuthenticationService(
		userRepo,
		otpRepo,
		refreshTokenRepo,
		&CacheAdapter{},
	)

	// ============================================================================
	// INITIALIZE RESOLVER (Presentation Layer)
	// ============================================================================
	// Resolver now uses services instead of repositories
	resolver := graph.NewResolver(
		userService,
		addressService,
		authService,
		&CacheAdapter{},
	)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Enable introspection for development
	if cfg.IsDevelopment() {
		srv.Use(extension.Introspection{})
	}

	// Add WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
	})

	// Add transports
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})

	// Setup HTTP routes
	http.Handle("/graphql", createGraphQLHandler(srv, cfg, userRepo, addressRepo))
	http.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))
	http.HandleFunc("/health", healthCheckHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      nil, // Use DefaultServeMux
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s", cfg.Server.Port)
		log.Printf("🎮 GraphQL Playground: http://localhost:%s", cfg.Server.Port)
		log.Printf("💚 Health check: http://localhost:%s/health", cfg.Server.Port)
		
		if cfg.IsDevelopment() {
			log.Printf("📊 DataLoader enabled - Statistics will be logged per request")
			log.Printf("🔧 Service Layer active - Optimized batching and caching")
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}

// createGraphQLHandler creates GraphQL handler with all middleware
func createGraphQLHandler(srv *handler.Server, cfg *config.Config, userRepo domain.UserRepository, addressRepo domain.AddressRepository) http.Handler {
	var h http.Handler = srv

	// Apply middleware in reverse order (last applied = first executed)

	// Recovery (catch panics)
	h = middleware.RecoveryMiddleware()(h)

	// Logging
	h = middleware.LoggingMiddleware()(h)

	// Request ID
	h = middleware.RequestIDMiddleware()(h)

	// DataLoader (CRITICAL: Must be early in chain for per-request lifecycle)
	h = dataloader.DataLoaderMiddleware(userRepo, addressRepo)(h)

	if cfg.IsProduction() {
		// Rate limiting
		cacheService := &CacheAdapter{}
		h = middleware.RateLimitMiddleware(cacheService, 60, time.Minute)(h)

		// Operation-specific rate limiting
		h = middleware.OperationRateLimiter(cacheService, middleware.DefaultOperationLimits())(h)
	} else {
		// ⚠️ In development, log that rate limiting is disabled
		log.Println("⚠️  Rate limiting DISABLED in development mode")
	}

	// Authentication (optional - sets context if token present)
	h = middleware.AuthMiddleware()(h)

	// CORS
	origins := middleware.DefaultAllowedOrigins()
	if cfg.IsProduction() {
		origins = middleware.ProductionAllowedOrigins()
	}
	h = middleware.CORSMiddleware(origins)(h)

	// Timeout
	h = middleware.TimeoutMiddleware(30 * time.Second)(h)

	return h
}

// healthCheckHandler returns server health status
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check database health
	dbErr := database.HealthCheck(r.Context())

	// Check Redis health
	cacheErr := cache.HealthCheck(r.Context())

	healthy := dbErr == nil && cacheErr == nil

	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	
	if !healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := fmt.Sprintf(`{
		"status": "%s",
		"database": %t,
		"redis": %t,
		"timestamp": "%s"
	}`, status, dbErr == nil, cacheErr == nil, time.Now().Format(time.RFC3339))

	w.Write([]byte(response))
}

// CacheAdapter adapts our cache package to the CacheService interface
type CacheAdapter struct{}

func (a *CacheAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return cache.Set(ctx, key, value, expiration)
}

func (a *CacheAdapter) Get(ctx context.Context, key string) (string, error) {
	return cache.Get(ctx, key)
}

func (a *CacheAdapter) Delete(ctx context.Context, keys ...string) error {
	return cache.Delete(ctx, keys...)
}

func (a *CacheAdapter) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	return cache.CheckRateLimit(ctx, key, limit, window)
}

func (c *CacheAdapter) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return cache.AcquireLock(ctx, key, ttl)
}

func (c *CacheAdapter) ReleaseLock(ctx context.Context, key string) error {
	return cache.ReleaseLock(ctx, key)
}

func (c *CacheAdapter) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	return cache.BlacklistToken(ctx, token, expiration)
}

func (c *CacheAdapter) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return cache.IsTokenBlacklisted(ctx, token)
}
