package n8n

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ringtonic-backend/internal/log"
)

// Client handles communication with n8n
type Client struct {
	webhookURL string
	secret     string
	httpClient *http.Client
	logger     *log.Logger
}

// New creates a new n8n client
func New(webhookURL, secret string, logger *log.Logger) *Client {
	return &Client{
		webhookURL: webhookURL,
		secret:     secret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TriggerWebhook sends a webhook request to n8n
func (c *Client) TriggerWebhook(payload map[string]interface{}) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", c.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Token", c.secret)

	// Add request ID for tracing
	if jobID, ok := payload["job_id"].(string); ok {
		req.Header.Set("X-Request-ID", jobID)
	}

	c.logger.Info("Sending webhook to n8n", "url", c.webhookURL, "payload_size", len(jsonData))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	c.logger.Info("Webhook sent successfully", "status_code", resp.StatusCode)
	return nil
}

// VerifyWebhookSignature verifies the webhook signature from n8n callback
func (c *Client) VerifyWebhookSignature(token string) bool {
	// For simple token-based authentication
	return hmac.Equal([]byte(token), []byte(c.secret))
}

// GenerateSignature generates HMAC signature for payload (if needed for future)
func (c *Client) GenerateSignature(payload []byte) string {
	h := hmac.New(sha256.New, []byte(c.secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyPayloadSignature verifies HMAC signature of payload (if needed for future)
func (c *Client) VerifyPayloadSignature(payload []byte, signature string) bool {
	expectedSignature := c.GenerateSignature(payload)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
