package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"turtle/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectWithPooling initializes database with connection pooling
func ConnectWithPooling() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL environment variable not set")
	}

	// Configure GORM logger
	logLevel := logger.Info
	if os.Getenv("APP_ENV") == "production" {
		// logLevel := logger.Warn
	}

	customLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	config := &gorm.Config{
		Logger:                                   customLogger,
		PrepareStmt:                              true,
		SkipDefaultTransaction:                   true,
		DisableForeignKeyConstraintWhenMigrating: false,
	}

	database, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if database == nil {
		log.Fatal("Database is nil")
	}

	// Configure connection pool
	sqlDB, err := database.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	maxOpenConns := 25
	maxIdleConns := 5
	connMaxLifetime := 5 * time.Minute
	connMaxIdleTime := 10 * time.Minute

	if os.Getenv("APP_ENV") == "production" {
		maxOpenConns = 100
		maxIdleConns = 10
		connMaxLifetime = 30 * time.Minute
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatal("Failed to ping database:", err)
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
		&models.ChatRoom{},
		&models.ChatMessage{},
	)

	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Create indexes
	createIndexes(database)

	DB = database
	log.Println("✅ PostgreSQL connected with connection pool")
	log.Printf("📊 Pool config: max_open=%d, max_idle=%d, max_lifetime=%v",
		maxOpenConns, maxIdleConns, connMaxLifetime)
}

// createIndexes creates database indexes
func createIndexes(db *gorm.DB) {
	log.Println("📊 Creating database indexes...")

	// Orders table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_customer_status 
		ON orders(customer_id, status) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_captain_status 
		ON orders(captain_id, status) WHERE deleted_at IS NULL AND captain_id IS NOT NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_status_created 
		ON orders(status, created_at DESC) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_placed_at 
		ON orders(placed_at DESC) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_pending_pickup 
		ON orders(status, pickup_lat, pickup_lng) 
		WHERE deleted_at IS NULL AND status = 'PENDING'`)

	// Users table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_role_status 
		ON users(role, status) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_available_captain 
		ON users(role, is_available, kyc_status) 
		WHERE deleted_at IS NULL AND role = 'CAPTAIN'`)

	// Simple location index (without PostGIS)
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_captain_location 
		ON users(current_lat, current_lng) 
		WHERE deleted_at IS NULL AND role = 'CAPTAIN' AND is_available = true`)

	// Addresses table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_addresses_user_active 
		ON addresses(user_id, is_active) WHERE deleted_at IS NULL AND is_active = true`)

	// Transactions table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_user_created 
		ON transactions(user_id, created_at DESC) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_order 
		ON transactions(order_id) WHERE deleted_at IS NULL AND order_id IS NOT NULL`)

	// Ratings table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ratings_rated_user 
		ON ratings(rated_user_id, created_at DESC) WHERE deleted_at IS NULL`)

	// ChatMessages table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_messages_room_created 
		ON chat_messages(room_id, created_at ASC) WHERE deleted_at IS NULL`)

	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chat_messages_unread 
		ON chat_messages(receiver_id, is_read) 
		WHERE deleted_at IS NULL AND is_read = false`)

	// Notifications table indexes
	executeIndex(db, `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_user_unread 
		ON notifications(user_id, is_read, created_at DESC) WHERE deleted_at IS NULL`)

	// Try to set up PostGIS if available (optional)
	setupPostGIS(db)

	log.Println("✅ Database indexes created/verified")
}

// executeIndex executes an index creation SQL with error handling
func executeIndex(db *gorm.DB, sql string) {
	result := db.Exec(sql)
	if result.Error != nil {
		// Log warning but don't fail - indexes might already exist
		log.Printf("⚠️  Index creation warning: %v", result.Error)
	}
}

// setupPostGIS sets up PostGIS extension if available (optional for development)
func setupPostGIS(db *gorm.DB) {
	log.Println("🗺️  Attempting to set up PostGIS (optional)...")

	// Try to enable PostGIS extension
	result := db.Exec(`CREATE EXTENSION IF NOT EXISTS postgis`)
	if result.Error != nil {
		log.Println("⚠️  PostGIS not available - using basic location indexing")
		log.Println("   For production, install PostGIS: brew install postgis (Mac) or apt-get install postgis (Linux)")
		return
	}

	log.Println("✅ PostGIS extension enabled")

	// Add geography column
	result = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS location geography(POINT, 4326)`)
	if result.Error != nil {
		log.Printf("⚠️  Could not add location column: %v", result.Error)
		return
	}

	// Create trigger function - execute separately
	result = db.Exec(`
		CREATE OR REPLACE FUNCTION update_user_location()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NEW.current_lat IS NOT NULL AND NEW.current_lng IS NOT NULL THEN
				NEW.location = ST_SetSRID(ST_MakePoint(NEW.current_lng, NEW.current_lat), 4326)::geography;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql
	`)
	if result.Error != nil {
		log.Printf("⚠️  Could not create trigger function: %v", result.Error)
		return
	}

	// Drop trigger if exists
	db.Exec(`DROP TRIGGER IF EXISTS trg_update_user_location ON users`)

	// Create trigger - separate statement
	result = db.Exec(`
		CREATE TRIGGER trg_update_user_location
		BEFORE INSERT OR UPDATE ON users
		FOR EACH ROW
		EXECUTE FUNCTION update_user_location()
	`)
	if result.Error != nil {
		log.Printf("⚠️  Could not create trigger: %v", result.Error)
		return
	}

	// Create GIST index for geospatial queries
	result = db.Exec(`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_location_gist 
		ON users USING GIST(location) 
		WHERE deleted_at IS NULL AND role = 'CAPTAIN' AND is_available = true`)
	if result.Error != nil {
		log.Printf("⚠️  Could not create spatial index: %v", result.Error)
		return
	}

	log.Println("✅ PostGIS setup complete - spatial queries enabled")
}

// FindNearbyCaptains using Haversine formula (works without PostGIS)
func FindNearbyCaptains(lat, lng, radiusKm float64, limit int) ([]models.User, error) {
	var captains []models.User

	// Check if PostGIS is available
	var hasPostGIS bool
	DB.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&hasPostGIS)

	if hasPostGIS {
		// Use PostGIS for better performance
		query := `
			SELECT * FROM users
			WHERE deleted_at IS NULL
			  AND role = 'CAPTAIN'
			  AND is_available = true
			  AND status = 'ACTIVE'
			  AND kyc_status = 'VERIFIED'
			  AND location IS NOT NULL
			  AND ST_DWithin(
				location,
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
				$3 * 1000
			  )
			ORDER BY location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
			LIMIT $4
		`
		err := DB.Raw(query, lng, lat, radiusKm, limit).Scan(&captains).Error
		if err != nil {
			return nil, fmt.Errorf("failed to find nearby captains: %w", err)
		}
	} else {
		// Fallback: Use Haversine formula (slower but works without PostGIS)
		query := `
			SELECT * FROM users
			WHERE deleted_at IS NULL
			  AND role = 'CAPTAIN'
			  AND is_available = true
			  AND status = 'ACTIVE'
			  AND kyc_status = 'VERIFIED'
			  AND current_lat IS NOT NULL
			  AND current_lng IS NOT NULL
			  AND (
				6371 * acos(
					cos(radians($1)) * cos(radians(current_lat)) *
					cos(radians(current_lng) - radians($2)) +
					sin(radians($1)) * sin(radians(current_lat))
				)
			  ) <= $3
			ORDER BY (
				6371 * acos(
					cos(radians($1)) * cos(radians(current_lat)) *
					cos(radians(current_lng) - radians($2)) +
					sin(radians($1)) * sin(radians(current_lat))
				)
			) ASC
			LIMIT $4
		`
		err := DB.Raw(query, lat, lng, radiusKm, limit).Scan(&captains).Error
		if err != nil {
			return nil, fmt.Errorf("failed to find nearby captains: %w", err)
		}
	}

	return captains, nil
}

// FindNearbyPendingOrders using Haversine formula
func FindNearbyPendingOrders(lat, lng, radiusKm float64, limit int) ([]models.Order, error) {
	var orders []models.Order

	query := `
		SELECT * FROM orders
		WHERE deleted_at IS NULL
		  AND status = 'PENDING'
		  AND captain_id IS NULL
		  AND (
			6371 * acos(
				cos(radians($1)) * cos(radians(pickup_lat)) *
				cos(radians(pickup_lng) - radians($2)) +
				sin(radians($1)) * sin(radians(pickup_lat))
			)
		  ) <= $3
		ORDER BY created_at DESC
		LIMIT $4
	`

	err := DB.Raw(query, lat, lng, radiusKm, limit).Scan(&orders).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby orders: %w", err)
	}

	return orders, nil
}

// CleanupOldRecords removes old data periodically
func CleanupOldRecords() error {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete old OTP sessions (older than 1 hour)
	tx.Where("created_at < ?", time.Now().Add(-1*time.Hour)).Delete(&models.OtpSession{})

	// Delete old expired refresh tokens
	tx.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{})

	// Delete old location tracking (older than 30 days)
	tx.Where("recorded_at < ?", time.Now().AddDate(0, 0, -30)).Delete(&models.LocationTracking{})

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	log.Println("✅ Old records cleaned up")
	return nil
}

// HealthCheck checks database connectivity
func HealthCheck(ctx context.Context) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	stats := sqlDB.Stats()
	if stats.OpenConnections >= stats.MaxOpenConnections {
		log.Printf("⚠️  Connection pool at maximum capacity: %d/%d",
			stats.OpenConnections, stats.MaxOpenConnections)
	}

	return nil
}