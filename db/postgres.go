package db

import (
	"log"
	"os"
	"turtle/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
    dsn := os.Getenv("DB_URL")
    
    config := &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    }
    
    database, err := gorm.Open(postgres.Open(dsn), config)
    if err != nil {
        panic(err)
    }
    
    if database == nil {
        panic("Database is nil")
    }
    
    // Auto-migrate all models
    err = database.AutoMigrate(
        &models.User{},
        &models.RefreshToken{},
        &models.OtpSession{},
        &models.Address{},
        &models.Order{},
        &models.Rating{},
        &models.Transaction{},
        &models.LocationTracking{},
        &models.CurrentLocation{},
        &models.Notification{},
        &models.Coupon{},
        &models.SupportTicket{},
        &models.CaptainEarnings{},
    )
    
    if err != nil {
        log.Fatal("Migration failed:", err)
    }
    
    DB = database
    log.Println("✅ PostgreSQL connected & migrated")
}