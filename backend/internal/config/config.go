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
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
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
	// If DATABASE_URL is set, parse it (for Neon, Render, and other managed services)
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		log.Println("Using DATABASE_URL for database connection (managed service)")
		return DatabaseConfig{
			Host:    databaseURL, // Store the full URL to signal we're using DATABASE_URL
			SSLMode: "url",       // Marker to use full URL mode
		}
	}

	// In production, DATABASE_URL is required
	if os.Getenv("ENV") == "production" {
		log.Fatal("DATABASE_URL environment variable must be set in production. Please set DATABASE_URL to your PostgreSQL connection string (e.g., from Neon or Render).")
	}

	// Fall back to individual environment variables (development/local setup)
	log.Println("DATABASE_URL not found, using individual DB_* environment variables (development mode)")
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
	// ✅ Highest priority: DATABASE_URL
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// Fallback for local/dev
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
