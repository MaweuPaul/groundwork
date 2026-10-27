package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"

	"groundwork/internal/config"
	"groundwork/internal/database"

	"gorm.io/gorm"
)

const migrationsDir = "migrations"

func main() {
	// Load environment & configuration
	cfg := config.Load()

	log.Printf("Connecting to database...")
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	log.Println("Database connection successful!")

	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("Failed to prepare schema_migrations: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("Failed to list migrations: %v", err)
	}
	sort.Strings(files) // 001_..., 002_..., run in order

	applied := 0
	for _, path := range files {
		name := filepath.Base(path)

		var count int64
		if err := db.Table("schema_migrations").Where("name = ?", name).Count(&count).Error; err != nil {
			log.Fatalf("Failed to check migration %s: %v", name, err)
		}
		if count > 0 {
			continue
		}

		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read migration file (%s): %v", path, err)
		}

		log.Printf("Executing migration %s...", name)

		// Run the file and record it together, so a failed migration
		// is never marked as applied.
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sqlBytes)).Error; err != nil {
				return err
			}
			return tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name).Error
		})
		if err != nil {
			log.Fatalf("Migration %s failed: %v", name, err)
		}
		applied++
	}

	if applied == 0 {
		log.Println("Database is up to date.")
		return
	}
	log.Printf("Applied %d migration(s).", applied)
}

// ensureMigrationsTable creates the table that records applied migrations.
//
// Databases set up before this table existed already ran 001 by hand, so
// when the parcels table is present it is recorded as applied instead of
// being run again.
func ensureMigrationsTable(db *gorm.DB) error {
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`).Error; err != nil {
		return err
	}

	if db.Migrator().HasTable("parcels") {
		return db.Exec(`
			INSERT INTO schema_migrations (name)
			VALUES ('001_initial_schema.sql')
			ON CONFLICT (name) DO NOTHING`).Error
	}

	return nil
}
