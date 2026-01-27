package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Server   ServerConfig
}

// AppConfig contains application-level configuration
type AppConfig struct {
	Name        string
	Environment string // dev, staging, production
	Version     string
	Debug       bool
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RedisConfig contains Redis connection settings
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// JWTConfig contains JWT settings
type JWTConfig struct {
	Secret               string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

var globalConfig *Config

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "turtle-delivery"),
			Environment: getEnv("APP_ENV", "dev"),
			Version:     getEnv("APP_VERSION", "1.0.0"),
			Debug:       getEnvAsBool("APP_DEBUG", true),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvAsInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Database:        getEnv("DB_NAME", "turtle_delivery"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:               getEnv("JWT_SECRET", "change-this-secret-in-production"),
			AccessTokenDuration:  getEnvAsDuration("JWT_ACCESS_DURATION", 15*time.Minute),
			RefreshTokenDuration: getEnvAsDuration("JWT_REFRESH_DURATION", 60*24*time.Hour),
		},
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     getEnvAsDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
	}

	// Validate critical configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	globalConfig = cfg
	return cfg, nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		panic("configuration not loaded - call config.Load() first")
	}
	return globalConfig
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Environment == "production" {
		if c.JWT.Secret == "change-this-secret-in-production" {
			return fmt.Errorf("JWT secret must be changed in production")
		}
		if c.Database.SSLMode == "disable" {
			return fmt.Errorf("database SSL must be enabled in production")
		}
	}

	if c.Database.MaxOpenConns < c.Database.MaxIdleConns {
		return fmt.Errorf("max open connections must be >= max idle connections")
	}

	return nil
}

// IsDevelopment checks if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "dev" || c.App.Environment == "development"
}

// IsProduction checks if running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "prod" || c.App.Environment == "production"
}

// GetDatabaseDSN returns PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// Helper functions to read environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// PrintConfig prints configuration (for debugging)
func (c *Config) PrintConfig() {
	fmt.Println("=== Application Configuration ===")
	fmt.Printf("Name: %s\n", c.App.Name)
	fmt.Printf("Environment: %s\n", c.App.Environment)
	fmt.Printf("Version: %s\n", c.App.Version)
	fmt.Printf("Debug: %v\n", c.App.Debug)
	fmt.Println()

	fmt.Println("=== Database Configuration ===")
	fmt.Printf("Host: %s\n", c.Database.Host)
	fmt.Printf("Port: %d\n", c.Database.Port)
	fmt.Printf("Database: %s\n", c.Database.Database)
	fmt.Printf("SSL Mode: %s\n", c.Database.SSLMode)
	fmt.Printf("Max Open Connections: %d\n", c.Database.MaxOpenConns)
	fmt.Printf("Max Idle Connections: %d\n", c.Database.MaxIdleConns)
	fmt.Println()

	fmt.Println("=== Redis Configuration ===")
	fmt.Printf("Host: %s\n", c.Redis.Host)
	fmt.Printf("Port: %d\n", c.Redis.Port)
	fmt.Printf("DB: %d\n", c.Redis.DB)
	fmt.Println()

	fmt.Println("=== Server Configuration ===")
	fmt.Printf("Port: %s\n", c.Server.Port)
	fmt.Printf("Read Timeout: %v\n", c.Server.ReadTimeout)
	fmt.Printf("Write Timeout: %v\n", c.Server.WriteTimeout)
	fmt.Println("================================")
}
