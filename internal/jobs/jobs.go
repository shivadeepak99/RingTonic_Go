package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ringtonic-backend/internal/log"
	"ringtonic-backend/internal/store"
)

// StoreInterface defines the interface for database operations
type StoreInterface interface {
	CreateJob(job *store.Job) error
	GetJob(id string) (*store.Job, error)
	UpdateJobStatus(id, status string, errorMessage *string) error
	IncrementJobAttempts(id string) error
	CreateRingtone(ringtone *store.Ringtone) error
	GetRingtoneByJobID(jobID string) (*store.Ringtone, error)
}

// N8NClientInterface defines the interface for n8n operations
type N8NClientInterface interface {
	TriggerWebhook(payload map[string]interface{}) error
}

// Manager handles job lifecycle management
type Manager struct {
	store     StoreInterface
	n8nClient N8NClientInterface
	logger    *log.Logger
}

// CreateJobRequest represents the request to create a new job
type CreateJobRequest struct {
	SourceURL string            `json:"source_url"`
	UserID    *string           `json:"user_id,omitempty"`
	Options   *store.JobOptions `json:"options,omitempty"`
}

// CreateJobResponse represents the response for job creation
type CreateJobResponse struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	PollURL string `json:"poll_url"`
}

// JobStatusResponse represents the response for job status queries
type JobStatusResponse struct {
	JobID       string    `json:"job_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DownloadURL *string   `json:"download_url,omitempty"`
	Error       *string   `json:"error,omitempty"`
}

// CallbackRequest represents the n8n callback payload
type CallbackRequest struct {
	JobID    string                 `json:"job_id"`
	Status   string                 `json:"status"`
	FilePath *string                `json:"file_path,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// New creates a new job manager
func New(store StoreInterface, n8nClient N8NClientInterface, logger *log.Logger) *Manager {
	return &Manager{
		store:     store,
		n8nClient: n8nClient,
		logger:    logger,
	}
}

// CreateJob creates a new ringtone generation job
func (m *Manager) CreateJob(req *CreateJobRequest) (*CreateJobResponse, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Set default options if not provided
	if req.Options == nil {
		req.Options = &store.JobOptions{
			Format: "mp3",
		}
	}
	if req.Options.Format == "" {
		req.Options.Format = "mp3"
	}

	// Create n8n payload
	n8nPayload := map[string]interface{}{
		"job_id":       jobID,
		"source_url":   req.SourceURL,
		"options":      req.Options,
		"callback_url": fmt.Sprintf("http://backend:8080/api/v1/n8n-callback"),
	}

	payloadJSON, err := json.Marshal(n8nPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal n8n payload: %w", err)
	}

	payloadStr := string(payloadJSON)

	// Create job in database
	job := &store.Job{
		ID:         jobID,
		SourceURL:  req.SourceURL,
		UserID:     req.UserID,
		Status:     store.StatusQueued,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Attempts:   0,
		N8NPayload: &payloadStr,
	}

	if err := m.store.CreateJob(job); err != nil {
		return nil, fmt.Errorf("failed to create job in database: %w", err)
	}

	m.logger.Info("Job created", "job_id", jobID, "source_url", req.SourceURL)

	// Trigger n8n workflow asynchronously
	go m.triggerN8NWorkflow(jobID, n8nPayload)

	return &CreateJobResponse{
		JobID:   jobID,
		Status:  store.StatusQueued,
		PollURL: fmt.Sprintf("/api/v1/job-status/%s", jobID),
	}, nil
}

// triggerN8NWorkflow triggers the n8n workflow with retry logic
func (m *Manager) triggerN8NWorkflow(jobID string, payload map[string]interface{}) {
	logger := m.logger.WithJobID(jobID)

	const maxAttempts = 3
	const baseDelay = time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logger.Info("Triggering n8n workflow", "attempt", attempt)

		// Increment attempts in database
		if err := m.store.IncrementJobAttempts(jobID); err != nil {
			logger.Error("Failed to increment job attempts", "error", err)
		}

		// Trigger webhook
		if err := m.n8nClient.TriggerWebhook(payload); err != nil {
			logger.Error("Failed to trigger n8n webhook", "error", err, "attempt", attempt)

			if attempt == maxAttempts {
				// Mark job as failed after max attempts
				errorMsg := fmt.Sprintf("Failed to trigger n8n after %d attempts: %v", maxAttempts, err)
				if updateErr := m.store.UpdateJobStatus(jobID, store.StatusFailed, &errorMsg); updateErr != nil {
					logger.Error("Failed to update job status to failed", "error", updateErr)
				}
				return
			}

			// Exponential backoff
			delay := baseDelay * time.Duration(1<<(attempt-1))
			logger.Info("Retrying after delay", "delay", delay)
			time.Sleep(delay)
			continue
		}

		// Success - update job status to processing
		if err := m.store.UpdateJobStatus(jobID, store.StatusProcessing, nil); err != nil {
			logger.Error("Failed to update job status to processing", "error", err)
		} else {
			logger.Info("n8n workflow triggered successfully")
		}
		return
	}
}

// GetJobStatus retrieves the current status of a job
func (m *Manager) GetJobStatus(jobID string) (*JobStatusResponse, error) {
	job, err := m.store.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job == nil {
		return nil, fmt.Errorf("job not found")
	}

	response := &JobStatusResponse{
		JobID:     job.ID,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
		Error:     job.ErrorMessage,
	}

	// If job is completed, get download URL
	if job.Status == store.StatusCompleted {
		ringtone, err := m.store.GetRingtoneByJobID(jobID)
		if err != nil {
			m.logger.Error("Failed to get ringtone for completed job", "job_id", jobID, "error", err)
		} else if ringtone != nil {
			downloadURL := fmt.Sprintf("/download/%s", ringtone.FileName)
			response.DownloadURL = &downloadURL
		}
	}

	return response, nil
}

// HandleCallback processes n8n callback
func (m *Manager) HandleCallback(req *CallbackRequest) error {
	logger := m.logger.WithJobID(req.JobID)
	logger.Info("Processing n8n callback", "status", req.Status)

	// Get job from database
	job, err := m.store.GetJob(req.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job == nil {
		return fmt.Errorf("job not found")
	}

	switch req.Status {
	case store.StatusCompleted:
		return m.handleCompletedCallback(req, logger)
	case store.StatusFailed:
		return m.handleFailedCallback(req, logger)
	default:
		return fmt.Errorf("unknown callback status: %s", req.Status)
	}
}

// handleCompletedCallback handles successful job completion
func (m *Manager) handleCompletedCallback(req *CallbackRequest, logger *log.Logger) error {
	if req.FilePath == nil {
		return fmt.Errorf("file_path is required for completed status")
	}

	// Extract metadata
	var duration *int
	if req.Metadata != nil {
		if d, ok := req.Metadata["duration"].(float64); ok {
			durationInt := int(d)
			duration = &durationInt
		}
	}

	// Create ringtone record
	ringtone := &store.Ringtone{
		JobID:           req.JobID,
		FileName:        *req.FilePath,
		FilePath:        *req.FilePath,
		Format:          "mp3", // TODO: Extract from metadata or filename
		DurationSeconds: duration,
		CreatedAt:       time.Now(),
	}

	if err := m.store.CreateRingtone(ringtone); err != nil {
		return fmt.Errorf("failed to create ringtone record: %w", err)
	}

	// Update job status
	if err := m.store.UpdateJobStatus(req.JobID, store.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	logger.Info("Job completed successfully", "file_path", *req.FilePath)
	return nil
}

// handleFailedCallback handles job failure
func (m *Manager) handleFailedCallback(req *CallbackRequest, logger *log.Logger) error {
	errorMessage := "Job failed in n8n workflow"
	if req.Metadata != nil {
		if msg, ok := req.Metadata["error"].(string); ok {
			errorMessage = msg
		}
	}

	// Update job status
	if err := m.store.UpdateJobStatus(req.JobID, store.StatusFailed, &errorMessage); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	logger.Error("Job failed", "error", errorMessage)
	return nil
}
