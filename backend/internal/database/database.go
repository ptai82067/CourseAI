package database

import (
	"courseai/backend/internal/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dsn string) error {
	var err error

	// Log connection attempt (mask password in logs if using DSN with individual components)
	if len(dsn) > 100 || dsn[:10] == "postgres:/" {
		log.Println("Connecting to PostgreSQL database (DATABASE_URL)...")
	} else {
		log.Printf("Connecting to PostgreSQL database: host=%s\n", extractHostFromDSN(dsn))
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w\nEnsure DATABASE_URL is set correctly or local PostgreSQL is running", err)
	}

	log.Println("✓ Database connected successfully")

	// Attempt to create pgcrypto extension (provides gen_random_uuid()) if available.
	// Do not fail connect if this is not permitted; migrations will handle errors in dev.
	if err := DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		log.Println("⚠ Warning: could not create pgcrypto extension:", err)
	} else {
		log.Println("✓ pgcrypto extension ensured")
	}
	return nil
}

func extractHostFromDSN(dsn string) string {
	// Simple extraction for logging purposes
	if len(dsn) > 50 {
		return dsn[:50] + "..."
	}
	return dsn
}

func AutoMigrate() error {
	log.Println("Running auto migrations...")

	// Run AutoMigrate per-model so failures are isolated and easier to debug.
	// Handle `users` table explicitly to avoid migration SQL that may rely
	// on database-side functions or extensions in environments where
	// privileges/extensions differ. We create the table idempotently with
	// a safe SQL CREATE TABLE IF NOT EXISTS and then allow GORM to migrate
	// the remaining models.

	// Ensure users table exists with a schema that is safe and idempotent.
	if !DB.Migrator().HasTable(&models.User{}) {
		createUsers := `CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			password_hash TEXT NOT NULL,
			role VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE,
			updated_at TIMESTAMP WITH TIME ZONE
		);`
		if err := DB.Exec(createUsers).Error; err != nil {
			return fmt.Errorf("migration failed for model *models.User (create table): %w", err)
		}
		// Create unique indexes idempotently
		if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);").Error; err != nil {
			return fmt.Errorf("failed to create unique index users.username: %w", err)
		}
		if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);").Error; err != nil {
			return fmt.Errorf("failed to create unique index users.email: %w", err)
		}
		log.Println("Created users table (idempotent)")
	}

	// Migrate remaining models (exclude User which is handled above)
	modelsToMigrate := []interface{}{
		&models.Program{},
		&models.Subcourse{},
		&models.Lesson{},
		&models.LessonObjective{},
		&models.LessonModel{},
		&models.LessonPreparation{},
		&models.LessonBuild{},
		&models.LessonContentBlock{},
		&models.LessonAttachment{},
		&models.LessonChallenge{},
		&models.LessonQuiz{},
		&models.LessonQuizOption{},
		&models.Media{},
		&models.TeacherAssignment{},
		&models.TeacherAssignmentLog{},
	}

	for _, m := range modelsToMigrate {
		// If the table already exists, skip AutoMigrate to avoid GORM's
		// introspection queries which can fail in some environments.
		if DB.Migrator().HasTable(m) {
			log.Printf("Skipping AutoMigrate for existing table %T", m)
			continue
		}
		if err := DB.AutoMigrate(m); err != nil {
			// Include the model type in the error message for easier root cause analysis
			return fmt.Errorf("migration failed for model %T: %w", m, err)
		}
	}

	// After migrating tables, ensure new Lesson columns exist using explicit SQL
	alterStmts := []string{
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS duration_minutes integer DEFAULT 0;",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS difficulty varchar(50);",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS estimated_time varchar(50);",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS cover_media_id uuid;",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS author_id uuid;",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS is_featured boolean DEFAULT false;",
		"ALTER TABLE lessons ADD COLUMN IF NOT EXISTS published_at timestamp with time zone;",
	}
	for _, stmt := range alterStmts {
		if err := DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute migration statement '%s': %w", stmt, err)
		}
	}

	log.Println("Migrations completed successfully")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
