package models

import (
	"time"
)

// ExecutionStatus represents execution status
type ExecutionStatus string

const (
	StatusPending  ExecutionStatus = "pending"
	StatusRunning  ExecutionStatus = "running"
	StatusSuccess  ExecutionStatus = "success"
	StatusFailed   ExecutionStatus = "failed"
	StatusCanceled ExecutionStatus = "canceled"
)

// Execution represents a task execution record
type Execution struct {
	ID           uint            `gorm:"primaryKey" json:"id"`
	TaskID       uint            `gorm:"not null;index" json:"task_id"`
	Status       ExecutionStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	StartTime    time.Time       `json:"start_time"`
	EndTime      *time.Time      `json:"end_time"`
	TotalBlobs   int             `json:"total_blobs"`
	SyncedBlobs  int             `json:"synced_blobs"`
	SkippedBlobs int             `json:"skipped_blobs"`
	FailedBlobs  int             `json:"failed_blobs"`
	TotalSize    int64           `json:"total_size"`
	SyncedSize   int64           `json:"synced_size"`
	ErrorMessage string          `gorm:"type:text" json:"error_message"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`

	// Relations
	Task SyncTask        `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Logs []ExecutionLog `gorm:"foreignKey:ExecutionID" json:"logs,omitempty"`
}

// TableName specifies the table name
func (Execution) TableName() string {
	return "executions"
}

// LogLevel represents log level
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelDebug LogLevel = "debug"
)

// ExecutionLog represents execution logs
type ExecutionLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ExecutionID uint      `gorm:"not null;index" json:"execution_id"`
	Level       LogLevel  `gorm:"type:varchar(10)" json:"level"`
	Message     string    `gorm:"type:text" json:"message"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
}

// TableName specifies the table name
func (ExecutionLog) TableName() string {
	return "execution_logs"
}

// Duration returns the execution duration
func (e *Execution) Duration() time.Duration {
	if e.EndTime == nil {
		return time.Since(e.StartTime)
	}
	return e.EndTime.Sub(e.StartTime)
}

// IsRunning checks if execution is running
func (e *Execution) IsRunning() bool {
	return e.Status == StatusRunning
}

// IsComplete checks if execution is complete
func (e *Execution) IsComplete() bool {
	return e.Status == StatusSuccess || e.Status == StatusFailed || e.Status == StatusCanceled
}

// Progress returns the progress percentage
func (e *Execution) Progress() float64 {
	if e.TotalBlobs == 0 {
		return 0
	}
	return float64(e.SyncedBlobs) / float64(e.TotalBlobs) * 100
}
