package db

import (
	"log"
	"os"
	"turtle/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DB_URL")

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}

	database.AutoMigrate(&models.User{})
	DB = database
	log.Println("PostgreSQL connected & migrated")
}
