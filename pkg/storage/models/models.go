package models

import (
	"time"

	"gorm.io/gorm"
)

// ResourceRecord is the main table storing all Argo Workflow resources as JSON blobs
// with queryable metadata columns.
type ResourceRecord struct {
	ID              uint           `gorm:"primaryKey;autoIncrement"`
	Kind            string         `gorm:"type:varchar(64);not null;index:idx_kind_ns_name,unique"`
	Namespace       string         `gorm:"type:varchar(256);not null;index:idx_kind_ns_name,unique"`
	Name            string         `gorm:"type:varchar(256);not null;index:idx_kind_ns_name,unique"`
	UID             string         `gorm:"type:varchar(128);not null;uniqueIndex"`
	ResourceVersion int64          `gorm:"not null;index"`
	Generation      int64          `gorm:"not null;default:1"`
	Data            string         `gorm:"type:text;not null"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
	Labels          []ResourceLabel `gorm:"foreignKey:ResourceID;constraint:OnDelete:CASCADE"`
}

func (ResourceRecord) TableName() string {
	return "resource_records"
}

// ResourceLabel stores labels for resources, enabling efficient label selector queries.
type ResourceLabel struct {
	ID         uint   `gorm:"primaryKey;autoIncrement"`
	ResourceID uint   `gorm:"not null;index:idx_resource_label,unique"`
	Key        string `gorm:"type:varchar(317);not null;index:idx_resource_label,unique;index:idx_key_value"`
	Value      string `gorm:"type:varchar(63);not null;index:idx_key_value"`
}

func (ResourceLabel) TableName() string {
	return "resource_labels"
}

// ResourceVersionCounter is a single-row table for a global monotonic resource version counter.
type ResourceVersionCounter struct {
	ID      uint  `gorm:"primaryKey"`
	Version int64 `gorm:"not null;default:0"`
}

func (ResourceVersionCounter) TableName() string {
	return "resource_version_counter"
}

// WatchEvent stores events for watch replay on reconnection.
type WatchEvent struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	Kind            string    `gorm:"type:varchar(64);not null;index:idx_watch_kind_ns"`
	Namespace       string    `gorm:"type:varchar(256);not null;index:idx_watch_kind_ns"`
	Name            string    `gorm:"type:varchar(256);not null"`
	UID             string    `gorm:"type:varchar(128);not null"`
	ResourceVersion int64     `gorm:"not null;index"`
	EventType       string    `gorm:"type:varchar(16);not null"` // ADDED, MODIFIED, DELETED
	Data            string    `gorm:"type:text;not null"`
	CreatedAt       time.Time `gorm:"not null;index"`
}

func (WatchEvent) TableName() string {
	return "watch_events"
}
