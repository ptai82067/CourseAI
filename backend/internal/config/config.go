package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type JWTConfig struct {
	Secret      string
	ExpireHours int
}

type ServerConfig struct {
	Port        string
	FrontendURL string
}

func Load() (*Config, error) {
	// Detect if running on managed platform (Render, Railway, etc)
	// Render doesn't set explicit RENDER env var, so check if .env fails to load
	dotenvLoaded := true
	if os.Getenv("RENDER") != "true" && os.Getenv("ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			dotenvLoaded = false
			log.Println("⚠️  .env file not found (expected in development)")
		}
	} else {
		log.Println("✓ Skipping .env file (production environment detected)")
		dotenvLoaded = true
	}

	// Force DATABASE_URL on non-development environments
	// If .env didn't load AND DATABASE_URL is empty, we're likely on Render without DATABASE_URL set
	if !dotenvLoaded && os.Getenv("DATABASE_URL") == "" {
		log.Fatal("FATAL: DATABASE_URL is required. " +
			"On Render/Railway, set DATABASE_URL in Environment Variables. " +
			"Example: postgresql://user:password@host:port/db?sslmode=require")
	}

	expireHours, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))

	// Try to use DATABASE_URL if available (Neon, Render, etc.)
	// Otherwise fall back to individual DB_* environment variables
	dbConfig := parseDatabase()

	return &Config{
		Database: dbConfig,
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "default-secret-change-me"),
			ExpireHours: expireHours,
		},
		Server: ServerConfig{
			Port:        getEnv("PORT", "8080"),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
		},
	}, nil
}

func parseDatabase() DatabaseConfig {
	databaseURL := os.Getenv("DATABASE_URL")
	
	log.Printf("Using DATABASE_URL: %v\n", databaseURL != "")

	// If DATABASE_URL is set, use it directly
	if databaseURL != "" {
		log.Println("✓ DATABASE_URL found - connecting to managed database")
		return DatabaseConfig{
			Host:    databaseURL,
			SSLMode: "url", // Marker to use full URL in DSN()
		}
	}

	// Fall back to individual DB vars (local/dev only)
	log.Println("⚠️  Using individual DB_* variables (development mode)")
	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Name:     getEnv("DB_NAME", "courseai_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func (c *DatabaseConfig) DSN() string {
	// If we're using DATABASE_URL, return it directly
	if c.SSLMode == "url" {
		return c.Host
	}

	// Otherwise, construct DSN from individual components
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
