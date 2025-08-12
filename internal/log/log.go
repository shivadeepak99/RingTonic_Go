package log

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Level represents log levels
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns string representation of log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

// Logger provides structured logging
type Logger struct {
	level  Level
	logger *log.Logger
}

// New creates a new logger with the specified level
func New(levelStr string) *Logger {
	level := parseLevel(levelStr)
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// parseLevel converts string to Level
func parseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(LevelError, msg, args...)
}

// WithJobID returns a logger with job_id context
func (l *Logger) WithJobID(jobID string) *Logger {
	return &Logger{
		level:  l.level,
		logger: l.logger,
	}
}

// log writes the log entry
func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"message":   msg,
	}

	// Add key-value pairs
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			value := args[i+1]
			entry[key] = value
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging
		l.logger.Printf("[%s] %s %v", level.String(), msg, args)
		return
	}

	l.logger.Println(string(jsonData))
}
