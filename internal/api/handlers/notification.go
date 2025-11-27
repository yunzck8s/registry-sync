package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"registry-sync/internal/db/models"
	"registry-sync/internal/db/store"
	"registry-sync/pkg/notification"
)

// NotificationHandler handles notification-related requests
type NotificationHandler struct {
	store *store.Store
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(store *store.Store) *NotificationHandler {
	return &NotificationHandler{store: store}
}

// CreateNotificationChannel creates a new notification channel
// POST /api/v1/notifications
func (h *NotificationHandler) CreateNotificationChannel(c *gin.Context) {
	var req models.NotificationChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate channel type
	if req.Type != "wechat" && req.Type != "dingtalk" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type, must be 'wechat' or 'dingtalk'"})
		return
	}

	if err := h.store.CreateNotificationChannel(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

// GetNotificationChannel gets a notification channel by ID
// GET /api/v1/notifications/:id
func (h *NotificationHandler) GetNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := h.store.GetNotificationChannel(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// ListNotificationChannels lists all notification channels
// GET /api/v1/notifications
func (h *NotificationHandler) ListNotificationChannels(c *gin.Context) {
	channels, err := h.store.ListNotificationChannels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, channels)
}

// UpdateNotificationChannel updates a notification channel
// PUT /api/v1/notifications/:id
func (h *NotificationHandler) UpdateNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	var req models.NotificationChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uint(id)

	// Validate channel type
	if req.Type != "wechat" && req.Type != "dingtalk" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type, must be 'wechat' or 'dingtalk'"})
		return
	}

	if err := h.store.UpdateNotificationChannel(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, req)
}

// DeleteNotificationChannel deletes a notification channel
// DELETE /api/v1/notifications/:id
func (h *NotificationHandler) DeleteNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	if err := h.store.DeleteNotificationChannel(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification channel deleted"})
}

// TestNotificationChannel sends a test notification
// POST /api/v1/notifications/:id/test
func (h *NotificationHandler) TestNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := h.store.GetNotificationChannel(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	// Send test notification
	notifier := notification.NewNotifier(channel)
	err = notifier.SendTestMessage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "test notification sent successfully"})
}
