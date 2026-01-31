package database

import (
	"fmt"
	"log"

	infraPostgres "turtle/internal/infrastructure/persistence/postgres"

	"gorm.io/gorm"
)

// RunAutoMigrations runs GORM auto-migrations for all models
func RunAutoMigrations(db *gorm.DB) error {
	log.Println("🔄 Running GORM auto-migrations...")

	models := []interface{}{
		&infraPostgres.UserModel{},
		&infraPostgres.CaptainProfileModel{},
		&infraPostgres.AdminProfileModel{},
		&infraPostgres.AddressModel{},
		&infraPostgres.OTPSessionModel{},
		&infraPostgres.RefreshTokenModel{},
	}

	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	log.Println("✅ Auto-migrations completed successfully")
	return nil
}

// CreateTriggers creates database triggers for updated_at
func CreateTriggers(db *gorm.DB) error {
	log.Println("🔄 Creating database triggers...")

	// Create update timestamp function
	functionSQL := `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;
	`

	if err := db.Exec(functionSQL).Error; err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create triggers for each table
	tables := []string{"users", "captain_profiles", "admin_profiles", "addresses"}
	successCount := 0
	for _, table := range tables {
		// Drop trigger first (separate statement)
		dropSQL := fmt.Sprintf("DROP TRIGGER IF EXISTS update_%s_updated_at ON %s", table, table)
		db.Exec(dropSQL) // Ignore errors on drop
		
		// Create trigger (separate statement)
		createSQL := fmt.Sprintf(`CREATE TRIGGER update_%s_updated_at
		BEFORE UPDATE ON %s
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column()`, table, table)

		if err := db.Exec(createSQL).Error; err != nil {
			log.Printf("⚠️  Warning: Failed to create trigger for %s: %v", table, err)
			// Don't fail on trigger errors
		} else {
			log.Printf("✅ Trigger created for %s", table)
			successCount++
		}
	}

	if successCount > 0 {
		log.Printf("✅ Triggers created successfully (%d/%d)", successCount, len(tables))
	}
	return nil
}

// CreateIndexes creates additional indexes not handled by GORM
func CreateIndexes(db *gorm.DB, hasPostGIS bool) error {
	log.Println("🔄 Creating additional indexes...")

	// Basic indexes (always create)
	basicIndexes := []string{
		// Composite index for address suggestions
		"CREATE INDEX IF NOT EXISTS idx_addresses_user_suggestions ON addresses(user_id, is_default, last_used_at DESC NULLS LAST) WHERE deleted_at IS NULL;",

		// Index for OTP lookups - FIXED: Removed NOW() from predicate
		"CREATE INDEX IF NOT EXISTS idx_otp_target_purpose ON otp_sessions(target, purpose) WHERE used = FALSE;",

		// Index for refresh token lookups
		"CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON refresh_tokens(token) WHERE is_revoked = FALSE;",
		"CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_device ON refresh_tokens(user_id, device) WHERE is_revoked = FALSE;",
	}

	successCount := 0
	for _, indexSQL := range basicIndexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("⚠️  Warning: Failed to create index: %v", err)
		} else {
			successCount++
		}
	}

	// PostGIS-dependent indexes (only if PostGIS is available)
	if hasPostGIS {
		postgisIndexes := []string{
			// Geospatial index for captain location
			"CREATE INDEX IF NOT EXISTS idx_captain_location ON captain_profiles USING GIST(current_location);",

			// Index for captain availability queries
			"CREATE INDEX IF NOT EXISTS idx_captain_availability ON captain_profiles(is_available, kyc_status) WHERE is_available = TRUE AND deleted_at IS NULL;",
		}

		for _, indexSQL := range postgisIndexes {
			if err := db.Exec(indexSQL).Error; err != nil {
				log.Printf("⚠️  Warning: Failed to create PostGIS index: %v", err)
			} else {
				successCount++
			}
		}
	} else {
		log.Println("ℹ️  Skipping PostGIS-dependent indexes (PostGIS not available)")
	}

	// Full-text search index (if pg_trgm is available)
	ftsSQL := "CREATE INDEX IF NOT EXISTS idx_users_search ON users USING GIN(to_tsvector('english', COALESCE(first_name, '') || ' ' || COALESCE(last_name, '') || ' ' || COALESCE(email, '')));"
	if err := db.Exec(ftsSQL).Error; err != nil {
		log.Printf("⚠️  Warning: Failed to create full-text search index: %v", err)
	} else {
		successCount++
	}

	log.Printf("✅ Indexes created successfully (%d total)", successCount)
	return nil
}

// CreateExtensions creates required PostgreSQL extensions
func CreateExtensions(db *gorm.DB, isDevelopment bool) error {
	log.Println("🔄 Creating PostgreSQL extensions...")

	extensions := []struct {
		SQL      string
		Required bool
		Name     string
	}{
		{
			SQL:      "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";",
			Required: true,
			Name:     "uuid-ossp",
		},
		{
			SQL:      "CREATE EXTENSION IF NOT EXISTS \"postgis\";",
			Required: !isDevelopment, // Not required in dev, but REQUIRED in production
			Name:     "postgis",
		},
		{
			SQL:      "CREATE EXTENSION IF NOT EXISTS \"pg_trgm\";",
			Required: false,
			Name:     "pg_trgm",
		},
	}

	hasPostGIS := false
	for _, ext := range extensions {
		if err := db.Exec(ext.SQL).Error; err != nil {
			if ext.Required {
				return fmt.Errorf("failed to create required extension %s: %w", ext.Name, err)
			} else {
				log.Printf("⚠️  Warning: Failed to create extension %s: %v", ext.Name, err)
			}
		} else {
			log.Printf("✅ Extension %s created", ext.Name)
			if ext.Name == "postgis" {
				hasPostGIS = true
			}
		}
	}

	if !hasPostGIS && !isDevelopment {
		return fmt.Errorf("PostGIS is required for production")
	}

	return nil
}

// MigrateDatabase runs all migration steps
func MigrateDatabase(db *gorm.DB, isDevelopment bool) error {
	log.Println("🚀 Starting database migration...")

	// Step 1: Create extensions
	if err := CreateExtensions(db, isDevelopment); err != nil {
		return err
	}

	// Check if PostGIS is available
	var hasPostGIS bool
	err := db.Raw("SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'postgis')").Scan(&hasPostGIS).Error
	if err != nil {
		log.Println("⚠️  Warning: Could not check PostGIS availability")
		hasPostGIS = false
	}

	// Step 2: Run auto-migrations
	if err := RunAutoMigrations(db); err != nil {
		return err
	}

	// Step 3: Create triggers
	if err := CreateTriggers(db); err != nil {
		// Don't fail on trigger errors in development
		if !isDevelopment {
			return err
		}
		log.Println("⚠️  Warning: Failed to create triggers (continuing anyway)")
	}

	// Step 4: Create additional indexes
	if err := CreateIndexes(db, hasPostGIS); err != nil {
		// Don't fail on index errors
		log.Println("⚠️  Warning: Some indexes may not have been created")
	}

	log.Println("🎉 Database migration completed successfully!")

	// Print warnings for development
	if isDevelopment && !hasPostGIS {
		log.Println("")
		log.Println("⚠️  DEVELOPMENT WARNING:")
		log.Println("   PostGIS is not installed. Geospatial features are disabled.")
		log.Println("   → Captain location tracking will not work")
		log.Println("   → Nearby captain search will not work")
		log.Println("")
		log.Println("   To enable geospatial features, install PostGIS:")
		log.Println("   → brew install postgis")
		log.Println("   → brew services restart postgresql@14")
		log.Println("")
		log.Println("   Or use Docker with PostGIS:")
		log.Println("   → docker run -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgis/postgis:15-3.3")
		log.Println("")
	}

	return nil
}