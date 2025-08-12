package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ringtonic-backend/internal/api"
	"ringtonic-backend/internal/files"
	"ringtonic-backend/internal/jobs"
	"ringtonic-backend/internal/log"
	"ringtonic-backend/internal/n8n"
	"ringtonic-backend/internal/store"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	// Create temporary database
	dbPath := "./test_api.db"
	database, err := store.New(dbPath)
	require.NoError(t, err)

	err = database.Migrate()
	require.NoError(t, err)

	// Create temp storage directory
	storageDir := "./test_storage"
	err = os.MkdirAll(storageDir, 0755)
	require.NoError(t, err)

	// Initialize components
	logger := log.New("error") // Reduce noise in tests
	fileManager := files.New(storageDir, logger)
	n8nClient := n8n.New("http://test:5678/webhook", "test-secret", logger)
	jobManager := jobs.New(database, n8nClient, logger)

	// Create server
	server := api.New(&api.Config{
		Database:      database,
		FileManager:   fileManager,
		JobManager:    jobManager,
		Logger:        logger,
		WebhookSecret: "test-secret",
	})

	// Cleanup function
	cleanup := func() {
		database.Close()
		os.Remove(dbPath)
		os.RemoveAll(storageDir)
	}

	return server, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "1.0.0", response.Version)
}

func TestCreateRingtone(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	requestBody := jobs.CreateJobRequest{
		SourceURL: "https://www.youtube.com/watch?v=test",
		Options: &store.JobOptions{
			StartSeconds:    intPtr(10),
			DurationSeconds: intPtr(20),
			Format:          "mp3",
		},
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/create-ringtone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response jobs.CreateJobResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.JobID)
	assert.Equal(t, "queued", response.Status)
	assert.Contains(t, response.PollURL, response.JobID)
}

func TestCreateRingtoneInvalidURL(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	requestBody := jobs.CreateJobRequest{
		SourceURL: "invalid-url",
	}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/create-ringtone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response api.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_URL", response.Code)
}

func TestJobStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test job directly in the database
	job := &store.Job{
		ID:        "test-job-123",
		SourceURL: "https://www.youtube.com/watch?v=test",
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := server.config.Database.CreateJob(job)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/api/v1/job-status/test-job-123", nil)
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response jobs.JobStatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-job-123", response.JobID)
	assert.Equal(t, "queued", response.Status)
}

func TestJobStatusNotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/job-status/non-existent", nil)
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestN8NCallback(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test job first
	job := &store.Job{
		ID:        "test-job-123",
		SourceURL: "https://www.youtube.com/watch?v=test",
		Status:    store.StatusProcessing,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := server.config.Database.CreateJob(job)
	require.NoError(t, err)

	// Test callback
	callbackBody := jobs.CallbackRequest{
		JobID:    "test-job-123",
		Status:   "completed",
		FilePath: stringPtr("test-job-123.mp3"),
		Metadata: map[string]interface{}{
			"duration": 25.0,
		},
	}

	body, err := json.Marshal(callbackBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/n8n-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Token", "test-secret")
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify job was updated
	updatedJob, err := server.config.Database.GetJob("test-job-123")
	require.NoError(t, err)
	assert.Equal(t, store.StatusCompleted, updatedJob.Status)

	// Verify ringtone was created
	ringtone, err := server.config.Database.GetRingtoneByJobID("test-job-123")
	require.NoError(t, err)
	require.NotNil(t, ringtone)
	assert.Equal(t, "test-job-123.mp3", ringtone.FileName)
}

func TestN8NCallbackUnauthorized(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	callbackBody := jobs.CallbackRequest{
		JobID:  "test-job-123",
		Status: "completed",
	}

	body, err := json.Marshal(callbackBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/n8n-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Token", "wrong-secret")
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMetrics(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response api.MetricsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response.JobStats)
	assert.NotEmpty(t, response.Uptime)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}
