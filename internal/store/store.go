package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store handles database operations
type Store struct {
	db *sql.DB
}

// Job represents a ringtone generation job
type Job struct {
	ID           string    `json:"id"`
	SourceURL    string    `json:"source_url"`
	UserID       *string   `json:"user_id,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Attempts     int       `json:"attempts"`
	N8NPayload   *string   `json:"n8n_payload,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}

// Ringtone represents a processed ringtone file
type Ringtone struct {
	ID              int       `json:"id"`
	JobID           string    `json:"job_id"`
	FileName        string    `json:"file_name"`
	FilePath        string    `json:"file_path"`
	Format          string    `json:"format"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// JobOptions represents processing options for a job
type JobOptions struct {
	StartSeconds    *int   `json:"start_seconds,omitempty"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
	FadeIn          bool   `json:"fade_in"`
	FadeOut         bool   `json:"fade_out"`
	Format          string `json:"format"`
}

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// New creates a new database connection
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate runs database migrations
func (s *Store) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			source_url TEXT NOT NULL,
			user_id TEXT,
			status TEXT NOT NULL DEFAULT 'queued',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			attempts INTEGER NOT NULL DEFAULT 0,
			n8n_payload TEXT,
			error_message TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS ringtones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			file_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			format TEXT NOT NULL,
			duration_seconds INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES jobs (id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_ringtones_job_id ON ringtones (job_id)`,
	}

	for _, migration := range migrations {
		if _, err := s.db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}

// CreateJob creates a new job
func (s *Store) CreateJob(job *Job) error {
	query := `
		INSERT INTO jobs (id, source_url, user_id, status, created_at, updated_at, attempts, n8n_payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		job.ID,
		job.SourceURL,
		job.UserID,
		job.Status,
		job.CreatedAt,
		job.UpdatedAt,
		job.Attempts,
		job.N8NPayload,
	)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (s *Store) GetJob(id string) (*Job, error) {
	query := `
		SELECT id, source_url, user_id, status, created_at, updated_at, attempts, n8n_payload, error_message
		FROM jobs
		WHERE id = ?
	`

	job := &Job{}
	err := s.db.QueryRow(query, id).Scan(
		&job.ID,
		&job.SourceURL,
		&job.UserID,
		&job.Status,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.Attempts,
		&job.N8NPayload,
		&job.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// UpdateJobStatus updates a job's status
func (s *Store) UpdateJobStatus(id, status string, errorMessage *string) error {
	query := `
		UPDATE jobs
		SET status = ?, updated_at = CURRENT_TIMESTAMP, error_message = ?
		WHERE id = ?
	`

	_, err := s.db.Exec(query, status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// IncrementJobAttempts increments the attempts counter for a job
func (s *Store) IncrementJobAttempts(id string) error {
	query := `
		UPDATE jobs
		SET attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to increment job attempts: %w", err)
	}

	return nil
}

// CreateRingtone creates a new ringtone record
func (s *Store) CreateRingtone(ringtone *Ringtone) error {
	query := `
		INSERT INTO ringtones (job_id, file_name, file_path, format, duration_seconds, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		ringtone.JobID,
		ringtone.FileName,
		ringtone.FilePath,
		ringtone.Format,
		ringtone.DurationSeconds,
		ringtone.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create ringtone: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get ringtone ID: %w", err)
	}

	ringtone.ID = int(id)
	return nil
}

// GetRingtoneByJobID retrieves a ringtone by job ID
func (s *Store) GetRingtoneByJobID(jobID string) (*Ringtone, error) {
	query := `
		SELECT id, job_id, file_name, file_path, format, duration_seconds, created_at
		FROM ringtones
		WHERE job_id = ?
	`

	ringtone := &Ringtone{}
	err := s.db.QueryRow(query, jobID).Scan(
		&ringtone.ID,
		&ringtone.JobID,
		&ringtone.FileName,
		&ringtone.FilePath,
		&ringtone.Format,
		&ringtone.DurationSeconds,
		&ringtone.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ringtone: %w", err)
	}

	return ringtone, nil
}

// GetRingtoneByFileName retrieves a ringtone by file name
func (s *Store) GetRingtoneByFileName(fileName string) (*Ringtone, error) {
	query := `
		SELECT id, job_id, file_name, file_path, format, duration_seconds, created_at
		FROM ringtones
		WHERE file_name = ?
	`

	ringtone := &Ringtone{}
	err := s.db.QueryRow(query, fileName).Scan(
		&ringtone.ID,
		&ringtone.JobID,
		&ringtone.FileName,
		&ringtone.FilePath,
		&ringtone.Format,
		&ringtone.DurationSeconds,
		&ringtone.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ringtone by file name: %w", err)
	}

	return ringtone, nil
}

// GetJobStats returns basic statistics about jobs
func (s *Store) GetJobStats() (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM jobs
		GROUP BY status
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get job stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan job stats: %w", err)
		}
		stats[status] = count
	}

	return stats, nil
}
