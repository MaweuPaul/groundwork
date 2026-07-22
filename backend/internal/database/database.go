package database

import (
	"fmt"
	"log"

	"groundwork/internal/models"

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

// AutoMigrate runs GORM auto migrations
func AutoMigrate(db *gorm.DB) error {
	log.Println("Running AutoMigrate...")
	// Note: AutoMigrate will not create PostGIS extension. Run migrations/001_initial_schema.sql first.
	return db.AutoMigrate(
		&models.Parcel{},
		&models.Methodology{},
		&models.Snapshot{},
		&models.Measurement{},
		&models.ImagerySource{},
		&models.Alert{},
	)
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
