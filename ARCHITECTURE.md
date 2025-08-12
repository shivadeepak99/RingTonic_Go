# RingTonic Go Backend - Technical Architecture Documentation

**Version:** 1.0.0  
**Date:** August 12, 2025  
**Purpose:** Internal developer guide for understanding the RingTonic Go backend architecture, execution flow, and integration points.

---

## 🚀 How to Run & Execute

### Quick Start
```bash
# 1. Clone and setup
git clone <repository>
cd RingTonic_Go
cp .env.example .env

# 2. Using Docker Compose (Recommended)
docker-compose up --build

# 3. Manual execution
go mod tidy
go build -o bin/server cmd/server/main.go
./bin/server

# 4. Run with migrations
./bin/server -migrate

# 5. Development with live reload
make dev
# or
air
```

### Testing
```bash
# Unit tests
go test ./...

# Integration tests
bash scripts/integration-test.sh

# Specific package tests
go test ./internal/jobs/jobs_test -v
```

### Environment Setup
- Copy `.env.example` to `.env`
- Configure `N8N_WEBHOOK_URL` and `N8N_WEBHOOK_SECRET`
- Ensure storage and data directories exist

---

## 📁 Architecture Overview

The RingTonic backend follows a clean architecture pattern with clear separation of concerns:

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Frontend  │───▶│  API Layer   │───▶│  Job Logic  │
│  (Next.js/  │    │ (handlers.go)│    │ (jobs.go)   │
│   Flutter)  │    │              │    │             │
└─────────────┘    └──────────────┘    └─────────────┘
                           │                    │
                           ▼                    ▼
                   ┌──────────────┐    ┌─────────────┐
                   │ File Manager │    │  Database   │
                   │  (files.go)  │    │ (store.go)  │
                   └──────────────┘    └─────────────┘
                           │                    │
                           ▼                    ▼
                   ┌──────────────┐    ┌─────────────┐
                   │   Storage    │    │   SQLite    │
                   │  Directory   │    │  Database   │
                   └──────────────┘    └─────────────┘
                           ▲
                           │
                   ┌──────────────┐
                   │  n8n Client  │◀─── n8n Workflows
                   │ (client.go)  │
                   └──────────────┘
```

---

## 📂 Root Level Files

### **`.env.example`**
- **Role:** Environment variable template
- **Execution Timing:** Manual setup before deployment
- **Triggers:** Developer configuration setup
- **Interconnections:** Read by `config.go` at runtime
- **Integration Points:** Contains n8n webhook URL and secrets for workflow integration
- **Example Use Case:** Developer copies this to `.env` and configures for local development

### **`docker-compose.yml`**
- **Role:** Orchestrates backend + n8n services for development
- **Execution Timing:** On `docker-compose up`
- **Triggers:** Manual execution or CI/CD pipeline
- **Interconnections:** Mounts `storage/` and `data/` directories, networks backend with n8n
- **Integration Points:** Provides complete development environment including n8n service
- **Example Use Case:** Developer runs `docker-compose up` to start both backend and n8n for testing

### **`Dockerfile`**
- **Role:** Multi-stage build for production-ready Go binary
- **Execution Timing:** During Docker build process
- **Triggers:** `docker build` command or CI/CD pipeline
- **Interconnections:** Copies `migrations/` and builds from `cmd/server/main.go`
- **Integration Points:** Deployed to production environments, connects to external n8n instances
- **Example Use Case:** CI/CD builds container for deployment to production VPS

### **`Makefile`**
- **Role:** Development automation and build commands
- **Execution Timing:** When `make <target>` is executed
- **Triggers:** Developer commands (`make dev`, `make test`, etc.)
- **Interconnections:** Orchestrates Docker, Go build, and testing tools
- **Integration Points:** Provides shortcuts for integration testing with n8n
- **Example Use Case:** Developer runs `make dev` to start full development environment

---

## 📂 `cmd/server/` - Application Entrypoint

### **`main.go`**
- **Role:** Application bootstrap and HTTP server lifecycle management
- **Execution Timing:** On application startup (`./bin/server`)
- **Triggers:** Direct execution or system service start
- **Interconnections:** 
  - Imports all `internal/` packages
  - Initializes database, file manager, job manager, n8n client
  - Sets up API routes and HTTP server
- **Integration Points:** Entry point for all external connections (HTTP clients, n8n webhooks)
- **Example Use Case:** 
  ```bash
  ./bin/server
  # Server starts, runs migrations, listens on :8080
  # Ready to receive API requests and n8n callbacks
  ```

**Key Execution Flow:**
1. Parse command flags (`-migrate`)
2. Load configuration from environment
3. Initialize SQLite database and run migrations
4. Create file manager, n8n client, job manager
5. Start HTTP server with graceful shutdown handling

---

## 📂 `internal/config/` - Configuration Management

### **`config.go`**
- **Role:** Environment variable parsing and configuration struct
- **Execution Timing:** Early in `main.go` startup sequence
- **Triggers:** `config.Load()` call from main function
- **Interconnections:** 
  - Provides configuration to all other packages
  - Read by database, file manager, n8n client, API server
- **Integration Points:** Contains n8n webhook URLs and secrets for external service integration
- **Example Use Case:**
  ```go
  cfg := config.Load()
  // cfg.N8NWebhookURL = "http://n8n:5678/webhook/ringtonic"
  // cfg.N8NWebhookSecret = "secure-token"
  ```

**Configuration Fields:**
- `Port`: HTTP server port
- `DBPath`: SQLite database file location
- `StoragePath`: File storage directory
- `N8NWebhookURL`: Target n8n webhook endpoint
- `N8NWebhookSecret`: Shared secret for authentication

---

## 📂 `internal/log/` - Structured Logging

### **`log.go`**
- **Role:** Structured JSON logging with context support
- **Execution Timing:** Throughout application lifecycle
- **Triggers:** Log calls from any package (`logger.Info()`, `logger.Error()`, etc.)
- **Interconnections:** Used by all packages for logging
- **Integration Points:** Logs n8n webhook calls, job processing, API requests
- **Example Use Case:**
  ```go
  logger.Info("Job created", "job_id", jobID, "source_url", url)
  // Output: {"timestamp":"2025-08-12T10:00:00Z","level":"info","message":"Job created","job_id":"123","source_url":"https://..."}
  ```

**Log Levels:** debug, info, warn, error  
**Context Support:** Job ID tracking, request tracing

---

## 📂 `internal/store/` - Database Layer

### **`store.go`**
- **Role:** SQLite database operations and data persistence
- **Execution Timing:** On database operations (job creation, status updates, file records)
- **Triggers:** 
  - Job creation from API requests
  - Status updates from n8n callbacks
  - File record creation on completion
- **Interconnections:**
  - Used by `jobs.go` for job management
  - Used by `handlers.go` for data retrieval
  - Initialized by `main.go`
- **Integration Points:** Stores job state for frontend polling, tracks n8n processing results
- **Example Use Case:**
  ```go
  // When user creates ringtone job
  job := &store.Job{ID: jobID, SourceURL: url, Status: "queued"}
  store.CreateJob(job)
  
  // When n8n completes processing
  store.UpdateJobStatus(jobID, "completed", nil)
  ```

**Database Schema:**
- `jobs`: Job lifecycle and metadata
- `ringtones`: Processed file information

### **`store_test/store_test.go`**
- **Role:** Database layer unit tests
- **Execution Timing:** During `go test` execution
- **Triggers:** Test runner or CI pipeline
- **Interconnections:** Tests `store.go` functions with temporary SQLite database
- **Integration Points:** Validates data integrity for frontend and n8n integration
- **Example Use Case:** Ensures job status transitions work correctly for polling endpoints

---

## 📂 `internal/files/` - File Management

### **`files.go`**
- **Role:** File storage, serving, and cleanup operations
- **Execution Timing:** 
  - On file downloads (HTTP requests to `/download/*`)
  - When n8n completes processing (file storage)
- **Triggers:**
  - HTTP GET requests to download endpoints
  - File cleanup routines (planned)
- **Interconnections:**
  - Used by `handlers.go` for file serving
  - Configured by `config.go` for storage path
- **Integration Points:**
  - Serves files to Flutter/Next.js clients
  - Stores files created by n8n workflows
- **Example Use Case:**
  ```go
  // When user downloads completed ringtone
  fileManager.ServeFile(w, r, "job-123.mp3")
  // Sets proper headers, streams file content
  ```

**Key Functions:**
- File serving with proper MIME types
- Security checks (only completed jobs)
- Future cleanup policies

---

## 📂 `internal/jobs/` - Job Orchestration

### **`jobs.go`**
- **Role:** Job lifecycle management and n8n workflow coordination
- **Execution Timing:**
  - On job creation (API requests)
  - On n8n callback processing
  - During status polling
- **Triggers:**
  - `POST /api/v1/create-ringtone` requests
  - `POST /api/v1/n8n-callback` webhooks
  - `GET /api/v1/job-status/*` polling
- **Interconnections:**
  - Uses `store.go` for persistence
  - Uses `n8n/client.go` for webhook triggers
  - Used by `handlers.go` for API operations
- **Integration Points:**
  - **Frontend**: Provides job status for polling
  - **n8n**: Triggers workflows and processes callbacks
- **Example Use Case:**
  ```go
  // User submits YouTube URL
  req := &CreateJobRequest{SourceURL: "https://youtube.com/watch?v=..."}
  response := jobManager.CreateJob(req)
  // Returns job_id, triggers n8n workflow asynchronously
  
  // n8n completes processing
  callback := &CallbackRequest{JobID: jobID, Status: "completed", FilePath: "job.mp3"}
  jobManager.HandleCallback(callback)
  // Updates database, file ready for download
  ```

**State Machine:** `queued` → `processing` → `completed`/`failed`

### **`jobs_test/jobs_test.go`**
- **Role:** Job manager unit tests with mocked dependencies
- **Execution Timing:** During test execution
- **Triggers:** Test runner
- **Interconnections:** Tests job logic with mocked store and n8n client
- **Integration Points:** Validates n8n callback handling and frontend response format
- **Example Use Case:** Ensures job creation returns proper polling URLs for frontend

---

## 📂 `internal/n8n/` - n8n Integration

### **`client.go`**
- **Role:** HTTP client for n8n webhook communication
- **Execution Timing:** When jobs are created (async webhook triggers)
- **Triggers:** Job creation in `jobs.go`
- **Interconnections:**
  - Used by `jobs.go` for workflow triggering
  - Configured by `config.go` for webhook URL and secrets
- **Integration Points:**
  - **Primary**: Triggers n8n workflows for video processing
  - **Security**: Webhook token verification for callbacks
- **Example Use Case:**
  ```go
  // When job is created, trigger n8n workflow
  payload := map[string]interface{}{
    "job_id": jobID,
    "source_url": "https://youtube.com/watch?v=...",
    "options": {"format": "mp3", "duration_seconds": 20},
    "callback_url": "http://backend:8080/api/v1/n8n-callback"
  }
  n8nClient.TriggerWebhook(payload)
  ```

**Key Features:**
- Retry logic with exponential backoff
- Webhook signature verification
- Request timeout handling

---

## 📂 `internal/api/` - HTTP API Layer

### **`handlers.go`**
- **Role:** HTTP request handling and API endpoint implementation
- **Execution Timing:** On incoming HTTP requests
- **Triggers:**
  - Frontend API calls
  - n8n webhook callbacks
  - File download requests
- **Interconnections:**
  - Uses `jobs.go` for job operations
  - Uses `files.go` for file serving
  - Uses `store.go` for direct database access
- **Integration Points:**
  - **Next.js/Flutter**: Primary API interface
  - **n8n**: Webhook callback endpoint
  - **Download clients**: File serving
- **Example Use Case:**
  ```go
  // Frontend creates ringtone job
  POST /api/v1/create-ringtone
  // Validates request, calls jobManager.CreateJob(), returns 202 with job_id
  
  // Frontend polls status
  GET /api/v1/job-status/{jobID}
  // Returns current status, download_url if completed
  
  // n8n reports completion
  POST /api/v1/n8n-callback
  // Verifies token, calls jobManager.HandleCallback()
  ```

**API Endpoints:**
- **Public**: `/healthz`, `/metrics`, `/api/v1/create-ringtone`, `/api/v1/job-status/*`, `/download/*`
- **Internal**: `/api/v1/n8n-callback`

### **`api_test/api_test.go`**
- **Role:** HTTP API integration tests
- **Execution Timing:** During test execution
- **Triggers:** Test runner
- **Interconnections:** Full HTTP request simulation with test database
- **Integration Points:** Validates complete request/response cycle for frontend integration
- **Example Use Case:** Tests full job creation → status polling → file download flow

---

## 📂 `migrations/` - Database Schema

### **`001_initial_tables.sql`**
- **Role:** Database schema definition and initial setup
- **Execution Timing:** On application startup (if `-migrate` flag) or first database access
- **Triggers:** `store.Migrate()` call from main.go
- **Interconnections:** Creates tables used by `store.go`
- **Integration Points:** Establishes data structure for frontend polling and n8n result storage
- **Example Use Case:**
  ```sql
  -- Creates jobs table for frontend status polling
  -- Creates ringtones table for completed file tracking
  ```

---

## 📂 `scripts/` - Development Tools

### **`integration-test.sh`**
- **Role:** End-to-end integration testing
- **Execution Timing:** Manual execution or CI pipeline
- **Triggers:** Developer testing or automated testing
- **Interconnections:** Tests complete API flow, simulates n8n callbacks
- **Integration Points:** Validates frontend API contracts and n8n webhook handling
- **Example Use Case:**
  ```bash
  # Starts server, tests job creation, simulates n8n callback, tests download
  bash scripts/integration-test.sh
  ```

### **`dev-check.ps1` / `dev-check.sh`**
- **Role:** Development environment validation
- **Execution Timing:** Before development or in CI
- **Triggers:** Manual execution
- **Interconnections:** Checks Go installation, dependencies, environment setup
- **Integration Points:** Validates development setup for n8n integration
- **Example Use Case:** New developer validates environment before contributing

---

## 📂 `storage/` & `data/` - Runtime Directories

### **`storage/`**
- **Role:** Runtime storage for processed audio files
- **Execution Timing:** When n8n workflows complete
- **Triggers:** File creation by n8n, file serving to clients
- **Interconnections:** Used by `files.go` for serving, written by n8n workflows
- **Integration Points:** 
  - **n8n**: Stores processed MP3 files
  - **Frontend**: Source for file downloads
- **Example Use Case:** n8n saves `job-123.mp3` here, frontend downloads via `/download/job-123.mp3`

### **`data/`**
- **Role:** SQLite database file storage
- **Execution Timing:** Database operations
- **Triggers:** Application startup, job operations
- **Interconnections:** Used by `store.go` for SQLite database
- **Integration Points:** Persistent storage for job state and metadata
- **Example Use Case:** `ringtonic.db` file stores all job and ringtone data

---

## 🔄 Complete Integration Flow

### 1. **User Creates Ringtone (Frontend → Backend)**
```
Frontend POST /api/v1/create-ringtone
  ↓
handlers.go → jobs.go → store.go (create job)
  ↓
jobs.go → n8n/client.go (trigger webhook)
  ↓
Response: job_id + polling URL
```

### 2. **n8n Processing (Backend → n8n → Backend)**
```
n8n receives webhook → processes video → saves file to storage/
  ↓
n8n POST /api/v1/n8n-callback
  ↓
handlers.go → jobs.go → store.go (update status + create ringtone record)
```

### 3. **Frontend Polling & Download**
```
Frontend GET /api/v1/job-status/{jobID}
  ↓
handlers.go → jobs.go → store.go (get status + download URL)
  ↓
Frontend GET /download/{filename}
  ↓
handlers.go → files.go (serve file)
```

---

## 🚀 Deployment & Integration Points

### **Development Environment**
- `docker-compose up` starts backend + n8n
- Environment variables configure service URLs
- Local file storage and SQLite database

### **Production Environment**
- Docker container deployment
- External n8n instance integration
- Persistent storage volumes
- Environment-specific configuration

### **Testing Strategy**
- Unit tests for each package
- Integration tests for API flow
- Mock objects for n8n communication
- Database isolation for tests

This architecture provides a robust, scalable foundation for the RingTonic service with clear separation of concerns and well-defined integration points for external services.
