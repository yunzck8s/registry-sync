package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"registry-sync/internal/db/store"
)

// ExecutionHandler handles execution-related requests
type ExecutionHandler struct {
	store *store.Store
}

// NewExecutionHandler creates a new execution handler
func NewExecutionHandler(store *store.Store) *ExecutionHandler {
	return &ExecutionHandler{store: store}
}

// GetExecution gets an execution by ID
// GET /api/v1/executions/:id
func (h *ExecutionHandler) GetExecution(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid execution ID"})
		return
	}

	exec, err := h.store.GetExecution(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "execution not found"})
		return
	}

	c.JSON(http.StatusOK, exec)
}

// ListExecutions lists all executions
// GET /api/v1/executions
func (h *ExecutionHandler) ListExecutions(c *gin.Context) {
	// Parse optional limit parameter
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	// Parse optional task_id filter
	taskIDStr := c.Query("task_id")
	var execs interface{}
	var err error

	if taskIDStr != "" {
		taskID, _ := strconv.ParseUint(taskIDStr, 10, 32)
		execs, err = h.store.ListExecutionsByTask(uint(taskID), limit)
	} else {
		execs, err = h.store.ListExecutions(limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, execs)
}

// GetExecutionLogs gets execution logs
// GET /api/v1/executions/:id/logs
func (h *ExecutionHandler) GetExecutionLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid execution ID"})
		return
	}

	// Parse optional limit parameter
	limitStr := c.DefaultQuery("limit", "1000")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.store.ListExecutionLogs(uint(id), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetStats gets system statistics
// GET /api/v1/stats
func (h *ExecutionHandler) GetStats(c *gin.Context) {
	stats, err := h.store.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
