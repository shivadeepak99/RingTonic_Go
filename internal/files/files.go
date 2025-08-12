package files

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ringtonic-backend/internal/log"
)

// Manager handles file operations
type Manager struct {
	basePath string
	logger   *log.Logger
}

// New creates a new file manager
func New(basePath string, logger *log.Logger) *Manager {
	return &Manager{
		basePath: basePath,
		logger:   logger,
	}
}

// EnsureDirectory creates the storage directory if it doesn't exist
func (m *Manager) EnsureDirectory() error {
	if err := os.MkdirAll(m.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	return nil
}

// GetFilePath returns the full path for a filename
func (m *Manager) GetFilePath(filename string) string {
	return filepath.Join(m.basePath, filename)
}

// FileExists checks if a file exists
func (m *Manager) FileExists(filename string) bool {
	path := m.GetFilePath(filename)
	_, err := os.Stat(path)
	return err == nil
}

// ServeFile serves a file with appropriate headers
func (m *Manager) ServeFile(w http.ResponseWriter, r *http.Request, filename string) error {
	filePath := m.GetFilePath(filename)

	// Check if file exists
	if !m.FileExists(filename) {
		http.NotFound(w, r)
		return fmt.Errorf("file not found: %s", filename)
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Set headers
	m.setFileHeaders(w, filename, fileInfo.Size())

	// Serve file content
	http.ServeContent(w, r, filename, fileInfo.ModTime(), file)

	m.logger.Info("File served", "filename", filename, "size", fileInfo.Size())
	return nil
}

// setFileHeaders sets appropriate headers for file downloads
func (m *Manager) setFileHeaders(w http.ResponseWriter, filename string, size int64) {
	// Set content type based on file extension
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		if strings.HasSuffix(strings.ToLower(filename), ".mp3") {
			contentType = "audio/mpeg"
		} else if strings.HasSuffix(strings.ToLower(filename), ".m4a") {
			contentType = "audio/mp4"
		} else {
			contentType = "application/octet-stream"
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
}

// SaveFile saves uploaded file content
func (m *Manager) SaveFile(filename string, content io.Reader) error {
	if err := m.EnsureDirectory(); err != nil {
		return err
	}

	filePath := m.GetFilePath(filename)

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy content
	_, err = io.Copy(file, content)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	m.logger.Info("File saved", "filename", filename, "path", filePath)
	return nil
}

// DeleteFile removes a file
func (m *Manager) DeleteFile(filename string) error {
	filePath := m.GetFilePath(filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // File already doesn't exist
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	m.logger.Info("File deleted", "filename", filename)
	return nil
}

// CleanupOldFiles removes files older than the specified number of days
// This is a stub implementation for future cleanup policy
func (m *Manager) CleanupOldFiles(days int) error {
	// TODO: Implement cleanup policy
	// This would:
	// 1. Find files older than X days
	// 2. Check if they have corresponding database entries
	// 3. Remove orphaned files
	// 4. Log cleanup activities

	m.logger.Info("File cleanup requested", "days", days)
	return nil
}

// GetFileSize returns the size of a file in bytes
func (m *Manager) GetFileSize(filename string) (int64, error) {
	filePath := m.GetFilePath(filename)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return fileInfo.Size(), nil
}
