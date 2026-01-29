package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"

	"turtle/config"
	"turtle/db"
	"turtle/graph"
	"turtle/graph/generated"
	"turtle/infra"
	"turtle/middlewares"
)

func main() {
    // Load environment variables
    env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	if err := config.Load(); err != nil {
		infra.Log.Warn("No env file found, using OS env")
	}

    // Initialize database
    db.Connect()
    log.Println("✅ Database connected")

    // Initialize Redis
    infra.InitRedis()

    // Create GraphQL server
    srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
        Resolvers: &graph.Resolver{},
    }))

    // Setup CORS
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
        AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders:   []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
    })

    // Create router with middleware chain
    mux := http.NewServeMux()

    // GraphQL endpoint with auth middleware
    graphqlHandler := middlewares.AuthMiddleware(srv)
    mux.Handle("/query", graphqlHandler)

    // Playground (no auth required)
    mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))

    // Health check endpoint (no auth required)
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // Wrap with CORS
    handler := c.Handler(mux)

    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("🚀 Server ready at http://localhost:%s/", port)
    log.Printf("🎮 Playground at http://localhost:%s/", port)
    log.Fatal(http.ListenAndServe(":"+port, handler))
}