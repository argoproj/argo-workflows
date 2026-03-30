package storage

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/argoproj/argo-workflows/v4/pkg/storage/models"
)

// NextResourceVersion atomically increments the global resource version counter
// and returns the new value. Must be called within a transaction.
func NextResourceVersion(tx *gorm.DB) (int64, error) {
	var counter models.ResourceVersionCounter

	// Use FOR UPDATE on Postgres, exclusive transaction on SQLite.
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&counter, 1)
	if result.Error != nil {
		// Fallback for SQLite which doesn't support SELECT FOR UPDATE.
		result = tx.First(&counter, 1)
		if result.Error != nil {
			return 0, fmt.Errorf("failed to read resource version counter: %w", result.Error)
		}
	}

	counter.Version++
	if err := tx.Save(&counter).Error; err != nil {
		return 0, fmt.Errorf("failed to increment resource version counter: %w", err)
	}

	return counter.Version, nil
}

// CurrentResourceVersion returns the current (non-incrementing) resource version.
func CurrentResourceVersion(db *gorm.DB) (int64, error) {
	var counter models.ResourceVersionCounter
	if err := db.First(&counter, 1).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read resource version counter: %w", err)
	}
	return counter.Version, nil
}
