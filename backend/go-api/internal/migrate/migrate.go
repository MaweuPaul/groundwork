// Package migrate applies the SQL files in migrations/ in name order, once each.
package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// Run applies every migration in dir that isn't recorded in schema_migrations yet, and
// returns the names it applied. logf, if set, is told about each one before it runs.
func Run(db *gorm.DB, dir string, logf func(format string, args ...any)) ([]string, error) {
	if err := ensureMigrationsTable(db); err != nil {
		return nil, fmt.Errorf("prepare schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no migrations found in %s", dir)
	}
	sort.Strings(files)

	var applied []string
	for _, path := range files {
		name := filepath.Base(path)

		var count int64
		if err := db.Table("schema_migrations").Where("name = ?", name).Count(&count).Error; err != nil {
			return applied, fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return applied, fmt.Errorf("read migration %s: %w", path, err)
		}
		if logf != nil {
			logf("Executing migration %s...", name)
		}

		// Run the file and record it together, so a failed migration is never marked as applied.
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sqlBytes)).Error; err != nil {
				return err
			}
			return tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name).Error
		})
		if err != nil {
			return applied, fmt.Errorf("migration %s failed: %w", name, err)
		}
		applied = append(applied, name)
	}
	return applied, nil
}

// ensureMigrationsTable creates the table that records applied migrations.
//
// Databases set up before this table existed already ran 001 by hand, so when the
// parcels table is present it is recorded as applied instead of being run again.
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
