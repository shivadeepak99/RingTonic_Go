package store_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ringtonic-backend/internal/store"
)

func TestStore_CreateAndGetJob(t *testing.T) {
	// Create temporary database
	dbPath := "./test_ringtonic.db"
	defer os.Remove(dbPath)

	database, err := store.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run migrations
	err = database.Migrate()
	require.NoError(t, err)

	// Create test job
	userID := "test-user"
	job := &store.Job{
		ID:        "test-job-id",
		SourceURL: "https://youtube.com/watch?v=test",
		UserID:    &userID,
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Attempts:  0,
	}

	// Test create
	err = database.CreateJob(job)
	assert.NoError(t, err)

	// Test get
	retrieved, err := database.GetJob("test-job-id")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, job.ID, retrieved.ID)
	assert.Equal(t, job.SourceURL, retrieved.SourceURL)
	assert.Equal(t, *job.UserID, *retrieved.UserID)
	assert.Equal(t, job.Status, retrieved.Status)
}

func TestStore_UpdateJobStatus(t *testing.T) {
	// Create temporary database
	dbPath := "./test_ringtonic.db"
	defer os.Remove(dbPath)

	database, err := store.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run migrations
	err = database.Migrate()
	require.NoError(t, err)

	// Create test job
	job := &store.Job{
		ID:        "test-job-id",
		SourceURL: "https://youtube.com/watch?v=test",
		Status:    store.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Attempts:  0,
	}

	err = database.CreateJob(job)
	require.NoError(t, err)

	// Update status
	errorMsg := "Test error"
	err = database.UpdateJobStatus("test-job-id", store.StatusFailed, &errorMsg)
	assert.NoError(t, err)

	// Verify update
	retrieved, err := database.GetJob("test-job-id")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, store.StatusFailed, retrieved.Status)
	assert.Equal(t, errorMsg, *retrieved.ErrorMessage)
}

func TestStore_CreateAndGetRingtone(t *testing.T) {
	// Create temporary database
	dbPath := "./test_ringtonic.db"
	defer os.Remove(dbPath)

	database, err := store.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run migrations
	err = database.Migrate()
	require.NoError(t, err)

	// Create test job first
	job := &store.Job{
		ID:        "test-job-id",
		SourceURL: "https://youtube.com/watch?v=test",
		Status:    store.StatusCompleted,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Attempts:  1,
	}

	err = database.CreateJob(job)
	require.NoError(t, err)

	// Create test ringtone
	duration := 30
	ringtone := &store.Ringtone{
		JobID:           "test-job-id",
		FileName:        "test.mp3",
		FilePath:        "/storage/test.mp3",
		Format:          "mp3",
		DurationSeconds: &duration,
		CreatedAt:       time.Now(),
	}

	// Test create
	err = database.CreateRingtone(ringtone)
	assert.NoError(t, err)
	assert.NotZero(t, ringtone.ID)

	// Test get by job ID
	retrieved, err := database.GetRingtoneByJobID("test-job-id")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, ringtone.JobID, retrieved.JobID)
	assert.Equal(t, ringtone.FileName, retrieved.FileName)
	assert.Equal(t, ringtone.Format, retrieved.Format)
	assert.Equal(t, *ringtone.DurationSeconds, *retrieved.DurationSeconds)

	// Test get by filename
	retrievedByName, err := database.GetRingtoneByFileName("test.mp3")
	require.NoError(t, err)
	require.NotNil(t, retrievedByName)

	assert.Equal(t, retrieved.ID, retrievedByName.ID)
}

func TestStore_GetJobStats(t *testing.T) {
	// Create temporary database
	dbPath := "./test_ringtonic.db"
	defer os.Remove(dbPath)

	database, err := store.New(dbPath)
	require.NoError(t, err)
	defer database.Close()

	// Run migrations
	err = database.Migrate()
	require.NoError(t, err)

	// Create test jobs with different statuses
	jobs := []*store.Job{
		{ID: "job1", SourceURL: "url1", Status: store.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job2", SourceURL: "url2", Status: store.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job3", SourceURL: "url3", Status: store.StatusProcessing, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job4", SourceURL: "url4", Status: store.StatusCompleted, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "job5", SourceURL: "url5", Status: store.StatusFailed, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, job := range jobs {
		err = database.CreateJob(job)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := database.GetJobStats()
	require.NoError(t, err)

	assert.Equal(t, 2, stats[store.StatusQueued])
	assert.Equal(t, 1, stats[store.StatusProcessing])
	assert.Equal(t, 1, stats[store.StatusCompleted])
	assert.Equal(t, 1, stats[store.StatusFailed])
}
