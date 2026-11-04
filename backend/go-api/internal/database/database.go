package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gormWriter sends GORM's output to slog. GORM is set to log only warnings and errors.
type gormWriter struct{}

func (gormWriter) Printf(format string, args ...any) {
	slog.Warn("gorm", "detail", fmt.Sprintf(format, args...))
}

// Connect initializes the GORM DB connection
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(gormWriter{}, logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true, // "no such user" is a normal answer, not an error
			ParameterizedQueries:      true, // log "WHERE email = $1", never the email itself
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	return db, nil
}

// Close closes the underlying *sql.DB connection
func Close(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("get sql.DB", "error", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		slog.Error("close sql.DB", "error", err)
	}
}
