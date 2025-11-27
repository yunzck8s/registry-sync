package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"registry-sync/internal/db/models"
)

// Notifier handles sending notifications to different channels
type Notifier struct {
	channel *models.NotificationChannel
}

// NewNotifier creates a new notifier for a given channel
func NewNotifier(channel *models.NotificationChannel) *Notifier {
	return &Notifier{channel: channel}
}

// SendTestMessage sends a test notification
func (n *Notifier) SendTestMessage() error {
	title := "Registry Sync - 测试通知"
	content := "### 连接测试\n\n> 这是一条测试消息，用于验证通知渠道配置是否正确。\n> \n> 如果您收到此消息，说明配置已成功。\n\n<font color=\"comment\">测试时间：" + time.Now().Format("2006-01-02 15:04:05") + "</font>"

	switch n.channel.Type {
	case "wechat":
		return n.sendWeChatMessage(title, content)
	case "dingtalk":
		return n.sendDingTalkMessage(title, content)
	default:
		return fmt.Errorf("unsupported channel type: %s", n.channel.Type)
	}
}

// SendTaskNotification sends a task execution notification
func (n *Notifier) SendTaskNotification(taskName string, status string, duration time.Duration, stats map[string]interface{}) error {
	var title string
	switch status {
	case "success":
		title = "Registry Sync - 任务执行成功"
	case "failed":
		title = "Registry Sync - 任务执行失败"
	default:
		title = "Registry Sync - 任务通知"
	}

	content := n.formatTaskNotification(taskName, status, duration, stats)

	switch n.channel.Type {
	case "wechat":
		return n.sendWeChatMessage(title, content)
	case "dingtalk":
		return n.sendDingTalkMessage(title, content)
	default:
		return fmt.Errorf("unsupported channel type: %s", n.channel.Type)
	}
}

// formatTaskNotification formats the notification content
func (n *Notifier) formatTaskNotification(taskName string, status string, duration time.Duration, stats map[string]interface{}) string {
	statusColor := ""
	switch status {
	case "success":
		statusColor = "<font color=\"info\">成功</font>"
	case "failed":
		statusColor = "<font color=\"warning\">失败</font>"
	default:
		statusColor = status
	}

	content := fmt.Sprintf("### 镜像同步任务通知\n\n")
	content += fmt.Sprintf("> **任务名称**：%s\n", taskName)
	content += fmt.Sprintf("> **执行状态**：%s\n", statusColor)
	content += fmt.Sprintf("> **执行耗时**：%s\n", formatDuration(duration))

	// Add statistics
	if totalBlobs, ok := stats["total_blobs"].(int); ok {
		syncedBlobs := 0
		if val, ok := stats["synced_blobs"].(int); ok {
			syncedBlobs = val
		}
		skippedBlobs := 0
		if val, ok := stats["skipped_blobs"].(int); ok {
			skippedBlobs = val
		}
		failedBlobs := 0
		if val, ok := stats["failed_blobs"].(int); ok {
			failedBlobs = val
		}

		content += fmt.Sprintf("> \n")
		content += fmt.Sprintf("> **数据统计**\n")
		content += fmt.Sprintf("> - 总计：%d 个 blob\n", totalBlobs)
		content += fmt.Sprintf("> - 成功：%d 个\n", syncedBlobs)
		if skippedBlobs > 0 {
			content += fmt.Sprintf("> - 跳过：%d 个\n", skippedBlobs)
		}
		if failedBlobs > 0 {
			content += fmt.Sprintf("> - 失败：<font color=\"warning\">%d 个</font>\n", failedBlobs)
		}
	}

	// Add error message if failed
	if status == "failed" {
		if errMsg, ok := stats["error"].(string); ok && errMsg != "" {
			content += fmt.Sprintf("> \n> **错误信息**：\n> ```\n> %s\n> ```\n", errMsg)
		}
	}

	content += fmt.Sprintf("\n<font color=\"comment\">%s</font>", time.Now().Format("2006-01-02 15:04:05"))

	return content
}

// sendWeChatMessage sends a message via WeChat Work webhook
func (n *Notifier) sendWeChatMessage(title, content string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("# %s\n\n%s", title, content),
		},
	}

	return n.sendHTTPRequest(n.channel.WebhookURL, payload)
}

// sendDingTalkMessage sends a message via DingTalk webhook
func (n *Notifier) sendDingTalkMessage(title, content string) error {
	// DingTalk uses a different format
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  fmt.Sprintf("# %s\n\n%s", title, content),
		},
	}

	return n.sendHTTPRequest(n.channel.WebhookURL, payload)
}

// sendHTTPRequest sends an HTTP POST request with JSON payload
func (n *Notifier) sendHTTPRequest(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	return nil
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1f小时", d.Hours())
}
