package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"ringtonic-backend/internal/files"
	"ringtonic-backend/internal/jobs"
	"ringtonic-backend/internal/log"
	"ringtonic-backend/internal/store"
)

// Server represents the HTTP server
type Server struct {
	config *Config
}

// Config holds server configuration
type Config struct {
	Database      *store.Store
	FileManager   *files.Manager
	JobManager    *jobs.Manager
	Logger        *log.Logger
	WebhookSecret string
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// MetricsResponse represents basic metrics
type MetricsResponse struct {
	JobStats map[string]int `json:"job_stats"`
	Uptime   string         `json:"uptime"`
}

var startTime = time.Now()

// New creates a new API server
func New(config *Config) *Server {
	return &Server{
		config: config,
	}
}

// Config returns the server configuration
func (s *Server) Config() *Config {
	return s.config
}

// Routes configures and returns the router
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health and metrics
	r.Get("/healthz", s.handleHealth)
	r.Get("/metrics", s.handleMetrics)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/create-ringtone", s.handleCreateRingtone)
		r.Get("/job-status/{jobID}", s.handleJobStatus)
		r.Post("/n8n-callback", s.handleN8NCallback)
	})

	// File downloads
	r.Get("/download/{filename}", s.handleDownload)

	return r
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.config.Logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start),
			"bytes", ww.BytesWritten(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMetrics handles metrics requests
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	jobStats, err := s.config.Database.GetJobStats()
	if err != nil {
		s.writeError(w, "Failed to get metrics", "METRICS_ERROR", http.StatusInternalServerError)
		return
	}

	response := MetricsResponse{
		JobStats: jobStats,
		Uptime:   time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateRingtone handles ringtone creation requests
func (s *Server) handleCreateRingtone(w http.ResponseWriter, r *http.Request) {
	var req jobs.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON payload", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.SourceURL == "" {
		s.writeError(w, "source_url is required", "MISSING_SOURCE_URL", http.StatusBadRequest)
		return
	}

	// Validate URL format (basic check)
	if !s.isValidURL(req.SourceURL) {
		s.writeError(w, "Invalid source URL format", "INVALID_URL", http.StatusBadRequest)
		return
	}

	// Create job
	response, err := s.config.JobManager.CreateJob(&req)
	if err != nil {
		s.config.Logger.Error("Failed to create job", "error", err, "source_url", req.SourceURL)
		s.writeError(w, "Failed to create job", "JOB_CREATION_ERROR", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// handleJobStatus handles job status requests
func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		s.writeError(w, "Job ID is required", "MISSING_JOB_ID", http.StatusBadRequest)
		return
	}

	response, err := s.config.JobManager.GetJobStatus(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, "Job not found", "JOB_NOT_FOUND", http.StatusNotFound)
			return
		}
		s.config.Logger.Error("Failed to get job status", "error", err, "job_id", jobID)
		s.writeError(w, "Failed to get job status", "JOB_STATUS_ERROR", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleN8NCallback handles n8n callback requests
func (s *Server) handleN8NCallback(w http.ResponseWriter, r *http.Request) {
	// Verify webhook token
	token := r.Header.Get("X-Webhook-Token")
	if token == "" {
		s.writeError(w, "Missing webhook token", "MISSING_TOKEN", http.StatusUnauthorized)
		return
	}

	if token != s.config.WebhookSecret {
		s.writeError(w, "Invalid webhook token", "INVALID_TOKEN", http.StatusUnauthorized)
		return
	}

	var req jobs.CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid JSON payload", "INVALID_JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.JobID == "" {
		s.writeError(w, "job_id is required", "MISSING_JOB_ID", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		s.writeError(w, "status is required", "MISSING_STATUS", http.StatusBadRequest)
		return
	}

	// Process callback
	if err := s.config.JobManager.HandleCallback(&req); err != nil {
		s.config.Logger.Error("Failed to handle callback", "error", err, "job_id", req.JobID)
		s.writeError(w, "Failed to process callback", "CALLBACK_ERROR", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// handleDownload handles file download requests
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		s.writeError(w, "Filename is required", "MISSING_FILENAME", http.StatusBadRequest)
		return
	}

	// Validate that the file belongs to a completed job
	ringtone, err := s.config.Database.GetRingtoneByFileName(filename)
	if err != nil {
		s.config.Logger.Error("Failed to get ringtone", "error", err, "filename", filename)
		s.writeError(w, "File not found", "FILE_NOT_FOUND", http.StatusNotFound)
		return
	}

	if ringtone == nil {
		s.writeError(w, "File not found", "FILE_NOT_FOUND", http.StatusNotFound)
		return
	}

	// Check job status
	job, err := s.config.Database.GetJob(ringtone.JobID)
	if err != nil {
		s.config.Logger.Error("Failed to get job", "error", err, "job_id", ringtone.JobID)
		s.writeError(w, "Internal server error", "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}

	if job == nil || job.Status != store.StatusCompleted {
		s.writeError(w, "File not available", "FILE_NOT_AVAILABLE", http.StatusForbidden)
		return
	}

	// Serve file
	if err := s.config.FileManager.ServeFile(w, r, filename); err != nil {
		s.config.Logger.Error("Failed to serve file", "error", err, "filename", filename)
		// Error response already handled by ServeFile
	}
}

// writeError writes an error response
func (s *Server) writeError(w http.ResponseWriter, message, code string, statusCode int) {
	response := ErrorResponse{
		Error: message,
		Code:  code,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// isValidURL performs basic URL validation
func (s *Server) isValidURL(url string) bool {
	url = strings.ToLower(url)
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
