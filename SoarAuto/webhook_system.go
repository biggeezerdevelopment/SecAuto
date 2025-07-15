package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	URL        string            `json:"url"`
	Events     []string          `json:"events"` // "job_started", "job_completed", "job_failed", "job_cancelled"
	Headers    map[string]string `json:"headers,omitempty"`
	Timeout    int               `json:"timeout_seconds,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
	RetryDelay int               `json:"retry_delay_seconds,omitempty"`
	Enabled    bool              `json:"enabled"`
}

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	Event     string                 `json:"event"`
	JobID     string                 `json:"job_id"`
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Playbook  []interface{}          `json:"playbook,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Results   []interface{}          `json:"results,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  float64                `json:"duration_seconds,omitempty"`
}

// WebhookManager manages webhook notifications
type WebhookManager struct {
	webhooks []WebhookConfig
	client   *http.Client
	mutex    sync.RWMutex
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager() *WebhookManager {
	return &WebhookManager{
		webhooks: make([]WebhookConfig, 0),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// AddWebhook adds a webhook configuration
func (wm *WebhookManager) AddWebhook(config WebhookConfig) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	wm.webhooks = append(wm.webhooks, config)
}

// SendWebhook sends a webhook notification
func (wm *WebhookManager) SendWebhook(event WebhookEvent) {
	wm.mutex.RLock()
	webhooks := make([]WebhookConfig, len(wm.webhooks))
	copy(webhooks, wm.webhooks)
	wm.mutex.RUnlock()

	for _, webhook := range webhooks {
		if !webhook.Enabled {
			continue
		}

		// Check if this webhook is interested in this event
		interested := false
		for _, eventType := range webhook.Events {
			if eventType == event.Event {
				interested = true
				break
			}
		}

		if !interested {
			continue
		}

		// Send webhook asynchronously
		go wm.sendWebhookWithRetry(webhook, event)
	}
}

// sendWebhookWithRetry sends a webhook with retry logic
func (wm *WebhookManager) sendWebhookWithRetry(config WebhookConfig, event WebhookEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		logger.Error("Failed to marshal webhook payload", map[string]interface{}{
			"component":   "webhook",
			"webhook_url": config.URL,
			"error":       err.Error(),
		})
		return
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewBuffer(payload))
	if err != nil {
		logger.Error("Failed to create webhook request", map[string]interface{}{
			"component":   "webhook",
			"webhook_url": config.URL,
			"error":       err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SecAuto-Webhook/1.0")

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	retryCount := config.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}

	retryDelay := time.Duration(config.RetryDelay) * time.Second
	if retryDelay == 0 {
		retryDelay = 5 * time.Second
	}

	for attempt := 0; attempt <= retryCount; attempt++ {
		resp, err := wm.client.Do(req)
		if err != nil {
			logger.Error("Webhook attempt failed", map[string]interface{}{
				"component":   "webhook",
				"webhook_url": config.URL,
				"attempt":     attempt + 1,
				"retry_count": retryCount,
				"error":       err.Error(),
			})
			if attempt < retryCount {
				time.Sleep(retryDelay)
				continue
			}
			return
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Info("Webhook sent successfully", map[string]interface{}{
				"component":   "webhook",
				"webhook_url": config.URL,
				"event":       event.Event,
				"job_id":      event.JobID,
				"status_code": resp.StatusCode,
			})
			return
		}

		logger.Warning("Webhook attempt failed", map[string]interface{}{
			"component":   "webhook",
			"webhook_url": config.URL,
			"attempt":     attempt + 1,
			"retry_count": retryCount,
			"status_code": resp.StatusCode,
		})
		if attempt < retryCount {
			time.Sleep(retryDelay)
		}
	}

	logger.Error("Webhook failed after all attempts", map[string]interface{}{
		"component":   "webhook",
		"webhook_url": config.URL,
		"retry_count": retryCount,
	})
}
