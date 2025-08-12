-- Migration: Create initial tables
-- Created: 2025-08-12
-- Version: 001

-- Jobs table: stores ringtone generation jobs
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    source_url TEXT NOT NULL,
    user_id TEXT,
    status TEXT NOT NULL DEFAULT 'queued',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    attempts INTEGER NOT NULL DEFAULT 0,
    n8n_payload TEXT,
    error_message TEXT
);

-- Ringtones table: stores completed ringtone files
CREATE TABLE IF NOT EXISTS ringtones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    format TEXT NOT NULL,
    duration_seconds INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES jobs (id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs (status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs (created_at);
CREATE INDEX IF NOT EXISTS idx_ringtones_job_id ON ringtones (job_id);
CREATE INDEX IF NOT EXISTS idx_ringtones_file_name ON ringtones (file_name);
