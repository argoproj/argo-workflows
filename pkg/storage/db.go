package storage

import (
	"fmt"

	"gorm.io/driver/postgres"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/argoproj/argo-workflows/v4/pkg/storage/models"
)

// NewDB initializes a GORM database connection and runs migrations.
// driver must be "sqlite" or "postgres". dsn is the data source name
// (e.g. ":memory:" or "argo.db" for sqlite, or a postgres connection string).
func NewDB(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "sqlite":
		dialector = sqlite.Open(dsn)
	case "postgres":
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (must be sqlite or postgres)", driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if driver == "sqlite" {
		// Enable WAL mode and foreign keys for SQLite.
		if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
			return nil, fmt.Errorf("failed to set WAL mode: %w", err)
		}
		if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
			return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}
