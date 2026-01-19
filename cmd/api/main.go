package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/vektah/gqlparser/v2/ast"

	"turtle/config"
	"turtle/db"
	"turtle/graph"
	"turtle/graph/generated"
	"turtle/infra"
	"turtle/middlewares"
)

func main() {
	// Load environment
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	log.Printf("🚀 Starting Turtle Backend [%s]", env)

	// Load config
	if err := config.Load(); err != nil {
		log.Printf("⚠️  No .env file found, using OS environment variables")
	}

	// Initialize database
	db.ConnectWithPooling()
	log.Println("✅ Database connected & migrated")

	// Initialize Redis
	infra.InitRedis()
	log.Println("✅ Redis connected")

	// Initialize services
	resolver := graph.NewResolver()

	// Create GraphQL server
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Add transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	// Add complexity limiting
	srv.Use(extension.FixedComplexityLimit(300))

	// Setup router
	router := chi.NewRouter()

	// Add middlewares
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.Compress(5))

	// CORS - IMPORTANT: Allow introspection for codegen
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:8080",
			"http://localhost:8081",
			"https://turtle.app",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	router.Use(corsHandler.Handler)

	// Health check endpoint (no auth)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"turtle-backend","version":"1.0.0"}`))
	})

	// Readiness check
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.HealthCheck(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database check failed"}`))
			return
		}

		if err := infra.Redis.Ping(r.Context()).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"redis ping failed"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","database":"ok","redis":"ok"}`))
	})

	// Liveness check
	router.Get("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("# HELP turtle_up Service is up\n# TYPE turtle_up gauge\nturtle_up 1\n"))
	})

	// GraphQL Playground (only in dev/staging)
	if env == "dev" || env == "staging" {
		router.Get("/", playground.Handler("GraphQL Playground", "/query"))
		log.Println("🎮 GraphQL Playground enabled at http://localhost:8080/")
	} else {
		router.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Turtle API v1.0 - GraphQL endpoint available at /query"))
		})
	}

	// Public introspection endpoint (NO AUTH)
	router.Handle("/graphql-introspect", srv)

	// GraphQL query endpoint with auth middleware
	router.Handle("/query", middlewares.AuthMiddleware(srv))

	// IMPORTANT: GraphQL schema introspection endpoint for codegen
	// This is what React Native codegen will use to fetch the schema
	router.Handle("/graphql", middlewares.AuthMiddleware(srv))

	// Alternative: Serve schema SDL (Schema Definition Language) file
	router.Get("/schema.graphql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// You can generate this with: go run github.com/99designs/gqlgen generate
		// For now, let introspection handle it
		w.Write([]byte("# Use /graphql endpoint for introspection\n"))
	})

	// WebSocket endpoint for subscriptions
	router.Handle("/subscriptions", srv)

	// File upload endpoint
	router.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement file upload
		w.Write([]byte("Upload endpoint"))
	})

	// Webhook endpoints
	router.Post("/webhooks/razorpay", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Post("/webhooks/fcm", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Server ready at http://localhost:%s/", port)
		if env == "dev" {
			log.Printf("🎮 Playground at http://localhost:%s/", port)
			log.Printf("📝 GraphQL endpoint: http://localhost:%s/query", port)
			log.Printf("📊 Schema endpoint: http://localhost:%s/graphql", port)
		}
		log.Printf("📊 Health check at http://localhost:%s/health", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("❌ Server failed:", err)
		}
	}()

	// Wait for interrupt signal
	<-stop

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("❌ Server forced to shutdown:", err)
	}

	// Close database connections
	sqlDB, _ := db.DB.DB()
	sqlDB.Close()

	// Close Redis connection
	infra.Redis.Close()

	log.Println("✅ Server exited gracefully")
}