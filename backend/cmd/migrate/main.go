package main

import (
	"log"
	"os"
	"path/filepath"

	"groundwork/internal/config"
	"groundwork/internal/database"
)

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

	// Locate the migration script
	migrationPath := filepath.Join("migrations", "001_initial_schema.sql")
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Failed to read migration file (%s): %v", migrationPath, err)
	}

	log.Printf("Executing migration %s...", migrationPath)
	if err := db.Exec(string(sqlBytes)).Error; err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration executed successfully! All PostGIS tables, indexes, and triggers have been created.")
}
