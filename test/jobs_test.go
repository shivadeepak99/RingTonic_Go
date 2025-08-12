package jobs_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ringtonic-backend/internal/jobs"
	"ringtonic-backend/internal/log"
	"ringtonic-backend/internal/store"
)

// MockStore implements store interface for testing
type MockStore struct {
	mock.Mock
}

func (m *MockStore) CreateJob(job *store.Job) error {
	args := m.Called(job)
	return args.Error(0)
}

func (m *MockStore) GetJob(id string) (*store.Job, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Job), args.Error(1)
}

func (m *MockStore) UpdateJobStatus(id, status string, errorMessage *string) error {
	args := m.Called(id, status, errorMessage)
	return args.Error(0)
}

func (m *MockStore) IncrementJobAttempts(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStore) CreateRingtone(ringtone *store.Ringtone) error {
	args := m.Called(ringtone)
	return args.Error(0)
}

func (m *MockStore) GetRingtoneByJobID(jobID string) (*store.Ringtone, error) {
	args := m.Called(jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Ringtone), args.Error(1)
}

func (m *MockStore) GetRingtoneByFileName(fileName string) (*store.Ringtone, error) {
	args := m.Called(fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Ringtone), args.Error(1)
}

func (m *MockStore) GetJobStats() (map[string]int, error) {
	args := m.Called()
	return args.Get(0).(map[string]int), args.Error(1)
}

func (m *MockStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStore) Migrate() error {
	args := m.Called()
	return args.Error(0)
}

// MockN8NClient implements n8n client interface for testing
type MockN8NClient struct {
	mock.Mock
}

func (m *MockN8NClient) TriggerWebhook(payload map[string]interface{}) error {
	args := m.Called(payload)
	return args.Error(0)
}

func (m *MockN8NClient) VerifyWebhookSignature(token string) bool {
	args := m.Called(token)
	return args.Bool(0)
}

func TestCreateJob(t *testing.T) {
	mockStore := &MockStore{}
	mockN8N := &MockN8NClient{}
	logger := log.New("error")

	manager := jobs.New(mockStore, mockN8N, logger)

	// Set up mock expectations
	mockStore.On("CreateJob", mock.AnythingOfType("*store.Job")).Return(nil)

	req := &jobs.CreateJobRequest{
		SourceURL: "https://www.youtube.com/watch?v=test",
		Options: &store.JobOptions{
			Format: "mp3",
		},
	}

	response, err := manager.CreateJob(req)

	require.NoError(t, err)
	assert.NotEmpty(t, response.JobID)
	assert.Equal(t, "queued", response.Status)
	assert.Contains(t, response.PollURL, response.JobID)

	mockStore.AssertExpectations(t)
}

func TestGetJobStatus(t *testing.T) {
	mockStore := &MockStore{}
	mockN8N := &MockN8NClient{}
	logger := log.New("error")

	manager := jobs.New(mockStore, mockN8N, logger)

	// Set up mock expectations
	job := &store.Job{
		ID:        "test-job",
		SourceURL: "https://test.com",
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockStore.On("GetJob", "test-job").Return(job, nil)

	response, err := manager.GetJobStatus("test-job")

	require.NoError(t, err)
	assert.Equal(t, "test-job", response.JobID)
	assert.Equal(t, "queued", response.Status)
	assert.Nil(t, response.DownloadURL)

	mockStore.AssertExpectations(t)
}

func TestGetJobStatusCompleted(t *testing.T) {
	mockStore := &MockStore{}
	mockN8N := &MockN8NClient{}
	logger := log.New("error")

	manager := jobs.New(mockStore, mockN8N, logger)

	// Set up mock expectations
	job := &store.Job{
		ID:        "test-job",
		SourceURL: "https://test.com",
		Status:    store.StatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ringtone := &store.Ringtone{
		JobID:    "test-job",
		FileName: "test-job.mp3",
		FilePath: "test-job.mp3",
		Format:   "mp3",
	}

	mockStore.On("GetJob", "test-job").Return(job, nil)
	mockStore.On("GetRingtoneByJobID", "test-job").Return(ringtone, nil)

	response, err := manager.GetJobStatus("test-job")

	require.NoError(t, err)
	assert.Equal(t, "completed", response.Status)
	require.NotNil(t, response.DownloadURL)
	assert.Equal(t, "/download/test-job.mp3", *response.DownloadURL)

	mockStore.AssertExpectations(t)
}

func TestHandleCallbackCompleted(t *testing.T) {
	mockStore := &MockStore{}
	mockN8N := &MockN8NClient{}
	logger := log.New("error")

	manager := New(mockStore, mockN8N, logger)

	// Set up mock expectations
	job := &store.Job{
		ID:     "test-job",
		Status: store.StatusProcessing,
	}
	mockStore.On("GetJob", "test-job").Return(job, nil)
	mockStore.On("CreateRingtone", mock.AnythingOfType("*store.Ringtone")).Return(nil)
	mockStore.On("UpdateJobStatus", "test-job", store.StatusCompleted, (*string)(nil)).Return(nil)

	filePath := "test-job.mp3"
	req := &CallbackRequest{
		JobID:    "test-job",
		Status:   store.StatusCompleted,
		FilePath: &filePath,
		Metadata: map[string]interface{}{
			"duration": 30.0,
		},
	}

	err := manager.HandleCallback(req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestHandleCallbackFailed(t *testing.T) {
	mockStore := &MockStore{}
	mockN8N := &MockN8NClient{}
	logger := log.New("error")

	manager := New(mockStore, mockN8N, logger)

	// Set up mock expectations
	job := &store.Job{
		ID:     "test-job",
		Status: store.StatusProcessing,
	}
	errorMsg := "Processing failed"
	mockStore.On("GetJob", "test-job").Return(job, nil)
	mockStore.On("UpdateJobStatus", "test-job", store.StatusFailed, &errorMsg).Return(nil)

	req := &CallbackRequest{
		JobID:  "test-job",
		Status: store.StatusFailed,
		Metadata: map[string]interface{}{
			"error": errorMsg,
		},
	}

	err := manager.HandleCallback(req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}
