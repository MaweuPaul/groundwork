package main

import (
	"log"

	"groundwork/internal/config"
	"groundwork/internal/database"
	"groundwork/internal/migrate"
)

const migrationsDir = "migrations"

func main() {
	cfg := config.Load()

	log.Printf("Connecting to database...")
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	log.Println("Database connection successful!")

	applied, err := migrate.Run(db, migrationsDir, log.Printf)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(applied) == 0 {
		log.Println("Database is up to date.")
		return
	}
	log.Printf("Applied %d migration(s).", len(applied))
}
