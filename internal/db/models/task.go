package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// StringArray is a custom type for storing string arrays in database
type StringArray []string

// Scan implements sql.Scanner
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, a)
}

// Value implements driver.Valuer
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// SyncTask represents a synchronization task
type SyncTask struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Name            string         `gorm:"uniqueIndex;not null" json:"name"`
	Description     string         `json:"description"`
	SourceRegistry  uint           `gorm:"not null" json:"source_registry"`
	SourceProject   string         `gorm:"not null" json:"source_project"`          // 新增：源项目名
	SourceRepo      string         `json:"source_repo"`                             // 改为可选：空=同步整个项目
	TargetRegistry  uint           `gorm:"not null" json:"target_registry"`
	TargetProject   string         `gorm:"not null" json:"target_project"`          // 新增：目标项目名
	TargetRepo      string         `json:"target_repo"`                             // 改为可选：空=使用源仓库名
	TagInclude      StringArray    `gorm:"type:json" json:"tag_include"`
	TagExclude      StringArray    `gorm:"type:json" json:"tag_exclude"`
	TagLatest       int            `json:"tag_latest"`
	Architectures   StringArray    `gorm:"type:json" json:"architectures"`
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	CronExpression  string         `json:"cron_expression"`

	// Notification settings
	SendNotification       bool   `gorm:"default:false" json:"send_notification"`
	NotificationCondition  string `gorm:"default:'all'" json:"notification_condition"` // "all" or "failed"
	NotificationChannelIDs string `gorm:"type:json" json:"notification_channel_ids"`   // JSON array of channel IDs

	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	SourceRegistryObj Registry `gorm:"foreignKey:SourceRegistry" json:"source_registry_obj,omitempty"`
	TargetRegistryObj Registry `gorm:"foreignKey:TargetRegistry" json:"target_registry_obj,omitempty"`
}

// GetSourceRepoPath 返回完整的源仓库路径
func (t *SyncTask) GetSourceRepoPath() string {
	if t.SourceRepo == "" {
		return t.SourceProject // 整个项目
	}
	return t.SourceProject + "/" + t.SourceRepo
}

// GetTargetRepoPath 返回完整的目标仓库路径
func (t *SyncTask) GetTargetRepoPath(sourceRepo string) string {
	targetRepo := sourceRepo
	if t.TargetRepo != "" {
		targetRepo = t.TargetRepo
	}
	return t.TargetProject + "/" + targetRepo
}

// TableName specifies the table name
func (SyncTask) TableName() string {
	return "sync_tasks"
}
