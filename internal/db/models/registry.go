package models

import (
	"time"

	"gorm.io/gorm"
)

// Registry represents a container registry configuration
type Registry struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	URL       string         `gorm:"not null" json:"url"`
	Username  string         `json:"username"`
	Password  string         `json:"password,omitempty"` // Accept password input but should be cleared before response
	Insecure  bool           `json:"insecure"`
	RateLimit int            `json:"rate_limit"` // QPS limit
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (Registry) TableName() string {
	return "registries"
}

// BeforeSave hook
func (r *Registry) BeforeSave(tx *gorm.DB) error {
	// Could add password encryption here
	return nil
}
