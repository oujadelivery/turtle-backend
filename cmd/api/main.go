package main

import (
	"log"
	"os"
	"turtle/config"

	"turtle/db"
	"turtle/graph"
	"turtle/graph/generated"
	"turtle/infra"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	if err := config.Load(); err != nil {
		infra.Log.Warn("No env file found, using OS env")
	}

	r := gin.Default()
	infra.InitLogger()
	defer infra.Log.Sync()
	r.Use(gin.Recovery())
	// r.SetTrustedProxies([]string{"127.0.0.1"})

	db.Connect()
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{Resolvers: &graph.Resolver{}},
		),
	)

	r.POST("/query", gin.WrapH(srv))
	r.GET("/", gin.WrapH(playground.Handler("GraphQL", "/query")))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Fatal(r.Run(":8080"))
}
