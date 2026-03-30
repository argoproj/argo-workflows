package models

import (
	"gorm.io/gorm"
)

// AutoMigrate creates or updates all tables using GORM auto-migration.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&ResourceRecord{},
		&ResourceLabel{},
		&ResourceVersionCounter{},
		&WatchEvent{},
	); err != nil {
		return err
	}

	// Ensure the single-row resource version counter exists.
	var count int64
	if err := db.Model(&ResourceVersionCounter{}).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Create(&ResourceVersionCounter{ID: 1, Version: 0}).Error
	}
	return nil
}
