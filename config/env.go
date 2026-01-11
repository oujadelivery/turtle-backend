package config

import (
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	return godotenv.Load(".env." + env)
}
