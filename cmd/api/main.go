package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
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
    db.Connect()
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

    // Add GraphQL extensions
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
    router.Use(middleware.Logger)
    router.Use(middleware.Recoverer)
    router.Use(middleware.Timeout(60 * time.Second))
    router.Use(middleware.Compress(5))

    // CORS
    corsHandler := cors.New(cors.Options{
        AllowedOrigins: []string{
            "http://localhost:3000",
            "http://localhost:8080",
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

    // Metrics endpoint (Prometheus)
    router.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
        // TODO: Add prometheus metrics handler
        w.Write([]byte("Metrics endpoint"))
    })

    // GraphQL Playground (disable in production)
    if env == "dev" || env == "staging" {
        router.Handle("/", playground.Handler("GraphQL Playground", "/query"))
        log.Println("🎮 GraphQL Playground enabled at /")
    } else {
        router.Get("/", func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Turtle API v1.0"))
        })
    }

    // GraphQL endpoint with auth middleware
    router.Route("/query", func(r chi.Router) {
        r.Use(middlewares.AuthMiddleware)
        r.Handle("/", srv)
    })

    // WebSocket endpoint for subscriptions
    router.Handle("/subscriptions", srv)

    // API versioning
    router.Route("/v1", func(r chi.Router) {
        // REST endpoints if needed
        r.Get("/orders/{id}", func(w http.ResponseWriter, r *http.Request) {
            // REST endpoint for specific needs
        })
    })

    // File upload endpoint
    router.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
        // TODO: Implement file upload (images, documents)
        w.Write([]byte("Upload endpoint"))
    })

    // Webhook endpoints (for payment gateways, etc.)
    router.Post("/webhooks/razorpay", func(w http.ResponseWriter, r *http.Request) {
        // TODO: Handle Razorpay webhooks
        w.WriteHeader(http.StatusOK)
    })

    router.Post("/webhooks/fcm", func(w http.ResponseWriter, r *http.Request) {
        // TODO: Handle FCM webhooks
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

    log.Printf("🚀 Server ready at http://localhost:%s/", port)
    if env == "dev" {
        log.Printf("🎮 Playground at http://localhost:%s/", port)
    }
    log.Printf("📊 Health check at http://localhost:%s/health", port)

    if err := server.ListenAndServe(); err != nil {
        log.Fatal("❌ Server failed:", err)
    }
}