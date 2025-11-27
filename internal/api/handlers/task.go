package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"registry-sync/internal/db/models"
	"registry-sync/internal/db/store"
)

// TaskHandler handles task-related requests
type TaskHandler struct {
	store *store.Store
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(store *store.Store) *TaskHandler {
	return &TaskHandler{store: store}
}

// CreateTask creates a new sync task
// POST /api/v1/tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req models.SyncTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate source and target registries exist
	if _, err := h.store.GetRegistry(req.SourceRegistry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source registry not found"})
		return
	}

	if _, err := h.store.GetRegistry(req.TargetRegistry); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target registry not found"})
		return
	}

	if err := h.store.CreateTask(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

// GetTask gets a task by ID
// GET /api/v1/tasks/:id
func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.store.GetTask(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks lists all tasks
// GET /api/v1/tasks
func (h *TaskHandler) ListTasks(c *gin.Context) {
	tasks, err := h.store.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// UpdateTask updates a task
// PUT /api/v1/tasks/:id
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req models.SyncTask
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uint(id)
	if err := h.store.UpdateTask(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// DeleteTask deletes a task
// DELETE /api/v1/tasks/:id
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	if err := h.store.DeleteTask(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

// RunTask runs a task immediately
// POST /api/v1/tasks/:id/run
func (h *TaskHandler) RunTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.store.GetTask(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// Check if task is already running
	if running, _ := h.store.GetRunningExecution(task.ID); running != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "task is already running"})
		return
	}

	// Note: Actual execution is handled by inline handler in main.go
	// This is just a placeholder for RESTful API consistency
	c.JSON(http.StatusOK, gin.H{
		"message": "task execution started",
		"task_id": task.ID,
	})
}

// StopTask stops a running task
// POST /api/v1/tasks/:id/stop
func (h *TaskHandler) StopTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.store.GetTask(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// Note: Actual cancellation is handled by inline handler in main.go
	// This is just a placeholder for RESTful API consistency
	c.JSON(http.StatusOK, gin.H{
		"message": "task stopped",
		"task_id": task.ID,
	})
}
