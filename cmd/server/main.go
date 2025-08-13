package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"RingTonic_Go/internal/api"
	"RingTonic_Go/internal/config"
	"RingTonic_Go/internal/files"
	"RingTonic_Go/internal/jobs"
	applog "RingTonic_Go/internal/log"
	"RingTonic_Go/internal/n8n"
	"RingTonic_Go/internal/store"
)

func main() {
	// Parse command line flags
	migrate := flag.Bool("migrate", false, "Run database migrations and exit")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := applog.New(cfg.LogLevel)
	logger.Info("Starting RingTonic Backend", "version", "1.0.0", "port", cfg.Port)

	// Initialize database
	database, err := store.New(cfg.DBPath)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations
	if err := database.Migrate(); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Exit if migration-only mode
	if *migrate {
		logger.Info("Migrations completed successfully")
		return
	}

	// Initialize file storage
	fileManager := files.New(cfg.StoragePath, logger)

	// Initialize n8n client
	n8nClient := n8n.New(cfg.N8NWebhookURL, cfg.N8NWebhookSecret, logger)

	// Initialize job manager
	jobManager := jobs.New(database, n8nClient, logger)

	// Initialize API server
	server := api.New(&api.Config{
		Database:      database,
		FileManager:   fileManager,
		JobManager:    jobManager,
		Logger:        logger,
		WebhookSecret: cfg.N8NWebhookSecret,
	})

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      server.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting HTTP server", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited successfully")
}
