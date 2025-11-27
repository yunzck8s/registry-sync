package store

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"registry-sync/internal/db/models"
)

// Store represents the database store
type Store struct {
	db *gorm.DB
}

// NewStore creates a new store
func NewStore(dbPath string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto migrate schemas
	if err := db.AutoMigrate(
		&models.Registry{},
		&models.SyncTask{},
		&models.Execution{},
		&models.ExecutionLog{},
		&models.NotificationChannel{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Store{db: db}, nil
}

// DB returns the underlying database instance
func (s *Store) DB() *gorm.DB {
	return s.db
}

// Close closes the database connection
func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Registry operations
func (s *Store) CreateRegistry(reg *models.Registry) error {
	return s.db.Create(reg).Error
}

func (s *Store) GetRegistry(id uint) (*models.Registry, error) {
	var reg models.Registry
	if err := s.db.First(&reg, id).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (s *Store) GetRegistryByName(name string) (*models.Registry, error) {
	var reg models.Registry
	if err := s.db.Where("name = ?", name).First(&reg).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (s *Store) ListRegistries() ([]models.Registry, error) {
	var regs []models.Registry
	if err := s.db.Find(&regs).Error; err != nil {
		return nil, err
	}
	return regs, nil
}

func (s *Store) UpdateRegistry(reg *models.Registry) error {
	return s.db.Save(reg).Error
}

func (s *Store) DeleteRegistry(id uint) error {
	return s.db.Delete(&models.Registry{}, id).Error
}

// Task operations
func (s *Store) CreateTask(task *models.SyncTask) error {
	return s.db.Create(task).Error
}

func (s *Store) GetTask(id uint) (*models.SyncTask, error) {
	var task models.SyncTask
	if err := s.db.Preload("SourceRegistryObj").Preload("TargetRegistryObj").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) GetTaskByName(name string) (*models.SyncTask, error) {
	var task models.SyncTask
	if err := s.db.Preload("SourceRegistryObj").Preload("TargetRegistryObj").Where("name = ?", name).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) ListTasks() ([]models.SyncTask, error) {
	var tasks []models.SyncTask
	if err := s.db.Preload("SourceRegistryObj").Preload("TargetRegistryObj").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Store) ListEnabledTasks() ([]models.SyncTask, error) {
	var tasks []models.SyncTask
	if err := s.db.Preload("SourceRegistryObj").Preload("TargetRegistryObj").Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *Store) UpdateTask(task *models.SyncTask) error {
	return s.db.Save(task).Error
}

func (s *Store) DeleteTask(id uint) error {
	return s.db.Delete(&models.SyncTask{}, id).Error
}

// Execution operations
func (s *Store) CreateExecution(exec *models.Execution) error {
	return s.db.Create(exec).Error
}

func (s *Store) GetExecution(id uint) (*models.Execution, error) {
	var exec models.Execution
	if err := s.db.Preload("Task").Preload("Logs").First(&exec, id).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

func (s *Store) ListExecutions(limit int) ([]models.Execution, error) {
	var execs []models.Execution
	query := s.db.Preload("Task").Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&execs).Error; err != nil {
		return nil, err
	}
	return execs, nil
}

func (s *Store) ListExecutionsByTask(taskID uint, limit int) ([]models.Execution, error) {
	var execs []models.Execution
	query := s.db.Where("task_id = ?", taskID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&execs).Error; err != nil {
		return nil, err
	}
	return execs, nil
}

func (s *Store) UpdateExecution(exec *models.Execution) error {
	return s.db.Save(exec).Error
}

func (s *Store) DeleteExecution(id uint) error {
	// Delete logs first
	s.db.Where("execution_id = ?", id).Delete(&models.ExecutionLog{})
	return s.db.Delete(&models.Execution{}, id).Error
}

// ExecutionLog operations
func (s *Store) CreateExecutionLog(log *models.ExecutionLog) error {
	return s.db.Create(log).Error
}

func (s *Store) ListExecutionLogs(executionID uint, limit int) ([]models.ExecutionLog, error) {
	var logs []models.ExecutionLog
	query := s.db.Where("execution_id = ?", executionID).Order("timestamp ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetRunningExecution returns the current running execution for a task
func (s *Store) GetRunningExecution(taskID uint) (*models.Execution, error) {
	var exec models.Execution
	if err := s.db.Where("task_id = ? AND status = ?", taskID, models.StatusRunning).First(&exec).Error; err != nil {
		return nil, err
	}
	return &exec, nil
}

// Statistics
func (s *Store) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalTasks int64
	s.db.Model(&models.SyncTask{}).Count(&totalTasks)
	stats["total_tasks"] = totalTasks

	var enabledTasks int64
	s.db.Model(&models.SyncTask{}).Where("enabled = ?", true).Count(&enabledTasks)
	stats["enabled_tasks"] = enabledTasks

	var totalExecutions int64
	s.db.Model(&models.Execution{}).Count(&totalExecutions)
	stats["total_executions"] = totalExecutions

	var runningExecutions int64
	s.db.Model(&models.Execution{}).Where("status = ?", models.StatusRunning).Count(&runningExecutions)
	stats["running_executions"] = runningExecutions

	var successExecutions int64
	s.db.Model(&models.Execution{}).Where("status = ?", models.StatusSuccess).Count(&successExecutions)
	stats["success_executions"] = successExecutions

	var failedExecutions int64
	s.db.Model(&models.Execution{}).Where("status = ?", models.StatusFailed).Count(&failedExecutions)
	stats["failed_executions"] = failedExecutions

	var totalRegistries int64
	s.db.Model(&models.Registry{}).Count(&totalRegistries)
	stats["total_registries"] = totalRegistries

	return stats, nil
}

// NotificationChannel operations
func (s *Store) CreateNotificationChannel(channel *models.NotificationChannel) error {
	return s.db.Create(channel).Error
}

func (s *Store) GetNotificationChannel(id uint) (*models.NotificationChannel, error) {
	var channel models.NotificationChannel
	if err := s.db.First(&channel, id).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *Store) ListNotificationChannels() ([]models.NotificationChannel, error) {
	var channels []models.NotificationChannel
	if err := s.db.Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (s *Store) UpdateNotificationChannel(channel *models.NotificationChannel) error {
	return s.db.Save(channel).Error
}

func (s *Store) DeleteNotificationChannel(id uint) error {
	return s.db.Delete(&models.NotificationChannel{}, id).Error
}
