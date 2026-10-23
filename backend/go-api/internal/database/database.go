package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect initializes the GORM DB connection
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	return db, nil
}

// Close closes the underlying *sql.DB connection
func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get sql.DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Failed to close sql.DB: %v", err)
	}
}
