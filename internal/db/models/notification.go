package models

import (
	"time"

	"gorm.io/gorm"
)

// NotificationChannel represents a notification channel (WeChat, DingTalk, etc.)
type NotificationChannel struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"uniqueIndex;not null" json:"name"`
	Type       string         `gorm:"not null" json:"type"` // "wechat", "dingtalk"
	WebhookURL string         `gorm:"not null" json:"webhook_url"`
	Secret     string         `json:"secret,omitempty"` // For DingTalk signature (future)
	Enabled    bool           `gorm:"default:true" json:"enabled"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// NotificationCondition represents when to send notifications
type NotificationCondition string

const (
	NotificationAll    NotificationCondition = "all"    // Send on success and failure
	NotificationFailed NotificationCondition = "failed" // Send only on failure
)
