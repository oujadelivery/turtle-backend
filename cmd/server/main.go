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
	"turtle/graph/generated" // ✅ Add this import for generated schema
	"turtle/internal/application/usecases"
	"turtle/internal/infrastructure/cache"
	"turtle/internal/infrastructure/database"
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

	// Connect to Redis
	err = cache.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer cache.Close()

	// Initialize repositories
	userRepo := infraPostgres.NewUserRepository(database.GetDB())
	addressRepo := infraPostgres.NewAddressRepository(database.GetDB())
	otpRepo := infraPostgres.NewOTPRepository(database.GetDB())
	refreshTokenRepo := infraPostgres.NewRefreshTokenRepository(database.GetDB())

	// Initialize services
	authService := usecases.NewAuthenticationService(
		userRepo,
		otpRepo,
		refreshTokenRepo,
		&CacheAdapter{},
	)

	// Initialize resolver
	resolver := graph.NewResolver(
		userRepo,
		addressRepo,
		otpRepo,
		refreshTokenRepo,
		authService,
		&CacheAdapter{},
	)

	// Create GraphQL server with correct import
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

	// Add POST transport
	srv.AddTransport(transport.POST{})

	// Add GET transport (for playground)
	srv.AddTransport(transport.GET{})

	// Setup HTTP routes
	mux := http.NewServeMux()

	// GraphQL endpoint
	mux.Handle("/graphql", createGraphQLHandler(srv, cfg))

	// GraphQL Playground (only in development)
	if cfg.IsDevelopment() {
		mux.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))
		log.Println("🎮 GraphQL Playground: http://localhost:" + cfg.Server.Port)
	}

	// Health check endpoint
	mux.HandleFunc("/health", healthCheckHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s/graphql\n", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
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
func createGraphQLHandler(srv *handler.Server, cfg *config.Config) http.Handler {
	var h http.Handler = srv

	// Apply middleware in reverse order (last applied = first executed)

	// Recovery (catch panics)
	h = middleware.RecoveryMiddleware()(h)

	// Logging
	h = middleware.LoggingMiddleware()(h)

	// Request ID
	h = middleware.RequestIDMiddleware()(h)

	// Rate limiting
	cacheService := &CacheAdapter{}
	h = middleware.RateLimitMiddleware(cacheService, 60, time.Minute)(h)

	// Operation-specific rate limiting
	h = middleware.OperationRateLimiter(cacheService, middleware.DefaultOperationLimits())(h)

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
	code := http.StatusOK
	if !healthy {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := fmt.Sprintf(`{
		"status": "%s",
		"timestamp": "%s",
		"services": {
			"database": %t,
			"cache": %t
		}
	}`, status, time.Now().Format(time.RFC3339), dbErr == nil, cacheErr == nil)
	
	w.Write([]byte(response))
}

// CacheAdapter adapts cache.Client to graph.CacheService interface
type CacheAdapter struct{}

func (c *CacheAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return cache.Set(ctx, key, value, expiration)
}

func (c *CacheAdapter) Get(ctx context.Context, key string) (string, error) {
	return cache.Get(ctx, key)
}

func (c *CacheAdapter) Delete(ctx context.Context, keys ...string) error {
	return cache.Delete(ctx, keys...)
}

func (c *CacheAdapter) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	return cache.CheckRateLimit(ctx, key, limit, window)
}

func (r *CacheAdapter) AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return cache.AcquireLock(ctx, key, ttl)
}

func (r *CacheAdapter) ReleaseLock(ctx context.Context, key string) error {
	return cache.ReleaseLock(ctx, key)
}

func (r *CacheAdapter) BlacklistToken(ctx context.Context, token string, expiration time.Duration) error {
	return cache.BlacklistToken(ctx, token, expiration)
}

func (r *CacheAdapter) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	return cache.IsTokenBlacklisted(ctx, token)
}
