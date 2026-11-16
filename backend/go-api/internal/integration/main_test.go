// Package integration runs the repositories and the database's own rules (triggers, queue
// functions, hash chain) against a real PostGIS database.
//
// It needs TEST_DATABASE_URL pointing at a database whose name ends in _test, with the
// postgis and pgcrypto extensions created. Without it every test here is skipped. Each
// test runs in a transaction that is rolled back, so the database stays empty.
package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"groundwork/internal/database"
	"groundwork/internal/migrate"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	url := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if url == "" {
		fmt.Println("integration: TEST_DATABASE_URL is not set; skipping")
		os.Exit(0)
	}

	var err error
	db, err = database.Connect(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration: connect:", err)
		os.Exit(1)
	}

	// The tests insert and roll back freely; never let that near a real database.
	var name string
	if err := db.Raw("SELECT current_database()").Scan(&name).Error; err != nil {
		fmt.Fprintln(os.Stderr, "integration: current_database:", err)
		os.Exit(1)
	}
	if !strings.HasSuffix(name, "_test") {
		fmt.Fprintf(os.Stderr, "integration: refusing to run against %q; the database name must end in _test\n", name)
		os.Exit(1)
	}

	// Quiet: GORM would log each long migration file as a slow query.
	if _, err := migrate.Run(db.Session(&gorm.Session{Logger: logger.Discard}), "../../migrations", nil); err != nil {
		fmt.Fprintln(os.Stderr, "integration: migrate:", err)
		os.Exit(1)
	}

	code := m.Run()
	database.Close(db)
	os.Exit(code)
}

// begin starts a transaction that is rolled back when the test ends.
func begin(t *testing.T) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	return tx
}
