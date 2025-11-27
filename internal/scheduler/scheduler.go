package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"registry-sync/internal/db/models"
	"registry-sync/internal/db/store"
	"registry-sync/internal/websocket"
	"registry-sync/pkg/config"
	"registry-sync/pkg/filter"
	"registry-sync/pkg/notification"
	"registry-sync/pkg/registry"
)

// Scheduler manages task scheduling and execution
type Scheduler struct {
	store   *store.Store
	cron    *cron.Cron
	hub     *websocket.Hub
	running map[uint]context.CancelFunc // task_id -> cancel function
}

// NewScheduler creates a new scheduler
func NewScheduler(store *store.Store, hub *websocket.Hub) *Scheduler {
	return &Scheduler{
		store:   store,
		cron:    cron.New(),
		hub:     hub,
		running: make(map[uint]context.CancelFunc),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	log.Println("Starting scheduler...")

	// Load all enabled tasks with cron expressions
	tasks, err := s.store.ListEnabledTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Schedule each task
	for _, task := range tasks {
		if task.CronExpression != "" {
			if err := s.ScheduleTask(&task); err != nil {
				log.Printf("Failed to schedule task %s: %v", task.Name, err)
			}
		}
	}

	s.cron.Start()
	log.Println("Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping scheduler...")
	s.cron.Stop()

	// Cancel all running tasks
	for taskID, cancel := range s.running {
		log.Printf("Cancelling task %d", taskID)
		cancel()
	}

	log.Println("Scheduler stopped")
}

// ScheduleTask schedules a task
func (s *Scheduler) ScheduleTask(task *models.SyncTask) error {
	if task.CronExpression == "" {
		return nil
	}

	_, err := s.cron.AddFunc(task.CronExpression, func() {
		log.Printf("Cron triggered for task: %s", task.Name)
		if err := s.ExecuteTask(context.Background(), task.ID); err != nil {
			log.Printf("Failed to execute task %s: %v", task.Name, err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	log.Printf("Scheduled task %s with cron: %s", task.Name, task.CronExpression)
	return nil
}

// ExecuteTask executes a task immediately
func (s *Scheduler) ExecuteTask(parentCtx context.Context, taskID uint) error {
	// Check if task is already running
	if _, exists := s.running[taskID]; exists {
		return fmt.Errorf("task %d is already running", taskID)
	}

	// Load task
	task, err := s.store.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to load task: %w", err)
	}

	// Create execution record
	execution := &models.Execution{
		TaskID:    task.ID,
		Status:    models.StatusRunning,
		StartTime: time.Now(),
	}

	if err := s.store.CreateExecution(execution); err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	log.Printf("Started execution %d for task %s", execution.ID, task.Name)

	// Create cancellable context
	ctx, cancel := context.WithCancel(parentCtx)
	s.running[taskID] = cancel

	// Run task in background
	go func() {
		defer func() {
			delete(s.running, taskID)
		}()

		startTime := execution.StartTime
		if err := s.runTask(ctx, task, execution); err != nil {
			log.Printf("Task %s failed: %v", task.Name, err)

			// Update execution status
			endTime := time.Now()
			execution.Status = models.StatusFailed
			execution.EndTime = &endTime
			execution.ErrorMessage = err.Error()
			s.store.UpdateExecution(execution)

			// Broadcast failure
			s.hub.BroadcastLog(execution.ID, "error", fmt.Sprintf("Task failed: %v", err))

			// Send notification if configured
			s.sendNotification(task, string(execution.Status), endTime.Sub(startTime), execution)
		} else {
			log.Printf("Task %s completed successfully", task.Name)

			// Update execution status
			endTime := time.Now()
			execution.Status = models.StatusSuccess
			execution.EndTime = &endTime
			s.store.UpdateExecution(execution)

			// Broadcast success
			s.hub.BroadcastLog(execution.ID, "info", "Task completed successfully")

			// Send notification if configured
			s.sendNotification(task, string(execution.Status), endTime.Sub(startTime), execution)
		}

		// Broadcast final progress
		s.hub.BroadcastProgress(execution.ID, map[string]interface{}{
			"status":       execution.Status,
			"total_blobs":  execution.TotalBlobs,
			"synced_blobs": execution.SyncedBlobs,
			"progress":     execution.Progress(),
		})
	}()

	return nil
}

// runTask runs the actual sync task
func (s *Scheduler) runTask(ctx context.Context, task *models.SyncTask, execution *models.Execution) error {
	// Load source and target registries
	sourceReg, err := s.store.GetRegistry(task.SourceRegistry)
	if err != nil {
		return fmt.Errorf("failed to load source registry: %w", err)
	}

	targetReg, err := s.store.GetRegistry(task.TargetRegistry)
	if err != nil {
		return fmt.Errorf("failed to load target registry: %w", err)
	}

	log.Printf("Starting sync: %s/%s -> %s/%s", sourceReg.Name, task.GetSourceRepoPath(), targetReg.Name, task.TargetProject)

	// Create execution log
	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     fmt.Sprintf("开始同步: %s/%s -> %s/%s", sourceReg.Name, task.GetSourceRepoPath(), targetReg.Name, task.TargetProject),
		Timestamp:   time.Now(),
	})

	// Create registry clients
	sourceClient := registry.NewClient(
		config.NormalizeRegistryURL(sourceReg.URL),
		sourceReg.Username,
		sourceReg.Password,
		sourceReg.Insecure,
		sourceReg.RateLimit,
	)

	targetClient := registry.NewClient(
		config.NormalizeRegistryURL(targetReg.URL),
		targetReg.Username,
		targetReg.Password,
		targetReg.Insecure,
		targetReg.RateLimit,
	)

	// Test connectivity
	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     "测试 Registry 连接...",
		Timestamp:   time.Now(),
	})

	if err := sourceClient.PingCheck(ctx); err != nil {
		errMsg := fmt.Sprintf("源 Registry 连接失败: %v", err)
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelError,
			Message:     errMsg,
			Timestamp:   time.Now(),
		})
		return fmt.Errorf(errMsg)
	}

	if err := targetClient.PingCheck(ctx); err != nil {
		errMsg := fmt.Sprintf("目标 Registry 连接失败: %v", err)
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelError,
			Message:     errMsg,
			Timestamp:   time.Now(),
		})
		return fmt.Errorf(errMsg)
	}

	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     "Registry 连接成功",
		Timestamp:   time.Now(),
	})

	// 检查并创建目标项目
	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     fmt.Sprintf("检查目标项目 %s 是否存在...", task.TargetProject),
		Timestamp:   time.Now(),
	})

	exists, err := targetClient.ProjectExists(ctx, task.TargetProject)
	if err != nil {
		// 项目检查失败，记录警告但继续（可能不是 Harbor）
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("无法检查项目存在性（可能不是 Harbor）: %v", err),
			Timestamp:   time.Now(),
		})
	} else if !exists {
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("目标项目 %s 不存在，正在创建...", task.TargetProject),
			Timestamp:   time.Now(),
		})

		if err := targetClient.CreateProject(ctx, task.TargetProject, true); err != nil {
			errMsg := fmt.Sprintf("创建目标项目失败: %v", err)
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelError,
				Message:     errMsg,
				Timestamp:   time.Now(),
			})
			return fmt.Errorf(errMsg)
		}

		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("成功创建目标项目 %s", task.TargetProject),
			Timestamp:   time.Now(),
		})
	} else {
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("目标项目 %s 已存在", task.TargetProject),
			Timestamp:   time.Now(),
		})
	}

	// 确定要同步的仓库列表
	var repositories []string
	if task.SourceRepo == "" {
		// 同步整个项目
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("获取项目 %s 的仓库列表...", task.SourceProject),
			Timestamp:   time.Now(),
		})

		repos, err := sourceClient.ListRepositories(ctx, task.SourceProject)
		if err != nil {
			errMsg := fmt.Sprintf("获取仓库列表失败: %v", err)
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelError,
				Message:     errMsg,
				Timestamp:   time.Now(),
			})
			return fmt.Errorf(errMsg)
		}
		repositories = repos

		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("找到 %d 个仓库: %v", len(repositories), repositories),
			Timestamp:   time.Now(),
		})
	} else {
		// 只同步单个仓库
		repositories = []string{task.SourceRepo}
		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("同步单个仓库: %s", task.SourceRepo),
			Timestamp:   time.Now(),
		})
	}

	// 第一步：预先计算所有仓库的 blob 总数（用于准确的进度显示）
	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     "正在分析所有仓库，计算需要同步的总数据量...",
		Timestamp:   time.Now(),
	})

	type repoTagInfo struct {
		repoName   string
		tag        string
		manifest   *registry.Manifest
		sourceRepo string
		targetRepo string
	}
	var allRepoTags []repoTagInfo
	totalBlobsCount := 0

	for _, repoName := range repositories {
		sourceRepoPath := task.SourceProject + "/" + repoName

		// List tags
		tags, err := sourceClient.ListTags(ctx, sourceRepoPath)
		if err != nil {
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelError,
				Message:     fmt.Sprintf("获取仓库 %s 的 tag 列表失败: %v", sourceRepoPath, err),
				Timestamp:   time.Now(),
			})
			continue
		}

		// Apply tag filters
		tagFilter, err := filter.NewFilter(task.TagInclude, task.TagExclude, task.TagLatest)
		if err != nil {
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelError,
				Message:     fmt.Sprintf("创建 tag 过滤器失败: %v", err),
				Timestamp:   time.Now(),
			})
			continue
		}

		tagInfos := make([]filter.TagInfo, len(tags))
		for i, tag := range tags {
			tagInfos[i] = filter.TagInfo{
				Name:    tag,
				Updated: time.Now(),
			}
		}

		filteredTags := tagFilter.FilterTags(tagInfos)

		// Get manifests and count blobs
		for _, tag := range filteredTags {
			manifest, err := sourceClient.GetManifest(ctx, sourceRepoPath, tag)
			if err != nil {
				s.store.CreateExecutionLog(&models.ExecutionLog{
					ExecutionID: execution.ID,
					Level:       models.LogLevelError,
					Message:     fmt.Sprintf("获取 manifest 失败 (%s:%s): %v", sourceRepoPath, tag, err),
					Timestamp:   time.Now(),
				})
				continue
			}

			blobs := manifest.GetAllBlobs()
			totalBlobsCount += len(blobs)

			allRepoTags = append(allRepoTags, repoTagInfo{
				repoName:   repoName,
				tag:        tag,
				manifest:   manifest,
				sourceRepo: sourceRepoPath,
				targetRepo: task.GetTargetRepoPath(repoName),
			})
		}
	}

	// 设置总 blob 数
	execution.TotalBlobs = totalBlobsCount
	s.store.UpdateExecution(execution)

	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     fmt.Sprintf("分析完成：共 %d 个仓库，%d 个 tag，%d 个 blob 需要同步", len(repositories), len(allRepoTags), totalBlobsCount),
		Timestamp:   time.Now(),
	})

	// 第二步：遍历所有仓库进行同步
	currentRepo := ""
	for tagIndex, repoTag := range allRepoTags {
		// 如果是新仓库，输出仓库信息
		if repoTag.repoName != currentRepo {
			currentRepo = repoTag.repoName
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelInfo,
				Message:     fmt.Sprintf("开始同步仓库: %s -> %s", repoTag.sourceRepo, repoTag.targetRepo),
				Timestamp:   time.Now(),
			})
		}

		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("[%d/%d] 同步 tag: %s:%s", tagIndex+1, len(allRepoTags), repoTag.repoName, repoTag.tag),
			Timestamp:   time.Now(),
		})

		// Get all blobs from the pre-fetched manifest
		blobs := repoTag.manifest.GetAllBlobs()

		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("tag %s 有 %d 个 blob", repoTag.tag, len(blobs)),
			Timestamp:   time.Now(),
		})

		// Sync blobs
		for _, blob := range blobs {
			// Check if exists
			exists, _, err := targetClient.BlobExists(ctx, repoTag.targetRepo, blob.Digest)
			if err != nil {
				s.store.CreateExecutionLog(&models.ExecutionLog{
					ExecutionID: execution.ID,
					Level:       models.LogLevelError,
					Message:     fmt.Sprintf("检查 blob 失败: %v", err),
					Timestamp:   time.Now(),
				})
				execution.FailedBlobs++
				continue
			}

			if exists {
				execution.SkippedBlobs++
				execution.SyncedBlobs++
			} else {
				// Copy blob
				err = registry.CopyBlob(ctx, sourceClient, targetClient, repoTag.sourceRepo, repoTag.targetRepo, blob.Digest, blob.Size)
				if err != nil {
					s.store.CreateExecutionLog(&models.ExecutionLog{
						ExecutionID: execution.ID,
						Level:       models.LogLevelError,
						Message:     fmt.Sprintf("复制 blob 失败 (%s): %v", blob.Digest[:12], err),
						Timestamp:   time.Now(),
					})
					execution.FailedBlobs++
				} else {
					execution.SyncedBlobs++
					execution.SyncedSize += blob.Size
				}
			}
			s.store.UpdateExecution(execution)

			// Broadcast progress
			s.hub.BroadcastProgress(execution.ID, map[string]interface{}{
				"total_blobs":  execution.TotalBlobs,
				"synced_blobs": execution.SyncedBlobs,
				"progress":     execution.Progress(),
			})
		}

		// Upload manifest
		_, err = targetClient.PutManifest(ctx, repoTag.targetRepo, repoTag.tag, repoTag.manifest)
		if err != nil {
			s.store.CreateExecutionLog(&models.ExecutionLog{
				ExecutionID: execution.ID,
				Level:       models.LogLevelError,
				Message:     fmt.Sprintf("上传 manifest 失败 (%s): %v", repoTag.tag, err),
				Timestamp:   time.Now(),
			})
			continue
		}

		s.store.CreateExecutionLog(&models.ExecutionLog{
			ExecutionID: execution.ID,
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("tag %s 同步完成", repoTag.tag),
			Timestamp:   time.Now(),
		})
	}

	s.store.CreateExecutionLog(&models.ExecutionLog{
		ExecutionID: execution.ID,
		Level:       models.LogLevelInfo,
		Message:     fmt.Sprintf("全部完成！共同步 %d 个 blob，跳过 %d 个，失败 %d 个", execution.SyncedBlobs, execution.SkippedBlobs, execution.FailedBlobs),
		Timestamp:   time.Now(),
	})

	return nil
}

// CancelTask cancels a running task
func (s *Scheduler) CancelTask(taskID uint) error {
	cancel, exists := s.running[taskID]
	if !exists {
		return fmt.Errorf("task %d is not running", taskID)
	}

	cancel()
	delete(s.running, taskID)

	log.Printf("Cancelled task %d", taskID)
	return nil
}

// sendNotification sends notification if configured for the task
func (s *Scheduler) sendNotification(task *models.SyncTask, status string, duration time.Duration, execution *models.Execution) {
	// Check if notification is enabled
	if !task.SendNotification {
		return
	}

	// Check notification condition
	if task.NotificationCondition == "failed" && status != string(models.StatusFailed) {
		log.Printf("Skipping notification for task %s: status is %s but condition is 'failed'", task.Name, status)
		return
	}

	// Parse channel IDs
	var channelIDs []uint
	if task.NotificationChannelIDs != "" {
		if err := json.Unmarshal([]byte(task.NotificationChannelIDs), &channelIDs); err != nil {
			log.Printf("Failed to parse notification channel IDs: %v", err)
			return
		}
	}

	if len(channelIDs) == 0 {
		log.Printf("No notification channels configured for task %s", task.Name)
		return
	}

	// Prepare notification stats
	stats := map[string]interface{}{
		"total_blobs":   execution.TotalBlobs,
		"synced_blobs":  execution.SyncedBlobs,
		"skipped_blobs": execution.SkippedBlobs,
		"failed_blobs":  execution.FailedBlobs,
	}

	if status == string(models.StatusFailed) {
		stats["error"] = execution.ErrorMessage
	}

	// Send to each channel
	for _, channelID := range channelIDs {
		channel, err := s.store.GetNotificationChannel(channelID)
		if err != nil {
			log.Printf("Failed to get notification channel %d: %v", channelID, err)
			continue
		}

		if !channel.Enabled {
			log.Printf("Notification channel %s is disabled, skipping", channel.Name)
			continue
		}

		notifier := notification.NewNotifier(channel)
		if err := notifier.SendTaskNotification(task.Name, status, duration, stats); err != nil {
			log.Printf("Failed to send notification to %s: %v", channel.Name, err)
		} else {
			log.Printf("Notification sent to %s for task %s", channel.Name, task.Name)
		}
	}
}
