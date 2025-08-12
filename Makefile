.PHONY: dev test clean lint build docker-build docker-run help

# Variables
BINARY_NAME=server
DOCKER_IMAGE=ringtonic-backend
STORAGE_DIR=./storage
DATA_DIR=./data

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start development environment with Docker Compose
	@echo "Starting development environment..."
	@mkdir -p $(STORAGE_DIR) $(DATA_DIR)
	docker-compose up --build

dev-down: ## Stop development environment
	@echo "Stopping development environment..."
	docker-compose down

dev-logs: ## Follow logs from development environment
	docker-compose logs -f

test: ## Run unit tests
	@echo "Running tests..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-verbose: ## Run tests with verbose output
	go test -v -race ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down

lint: ## Run linting
	@echo "Running linters..."
	go vet ./...
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	go mod tidy

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux cmd/server/main.go

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@mkdir -p $(STORAGE_DIR) $(DATA_DIR)
	docker run --rm -p 8080:8080 \
		-v $(PWD)/$(STORAGE_DIR):/app/storage \
		-v $(PWD)/$(DATA_DIR):/app/data \
		-e N8N_WEBHOOK_URL=http://host.docker.internal:5678/webhook/ringtonic \
		$(DOCKER_IMAGE):latest

run: build ## Build and run locally
	@echo "Running $(BINARY_NAME)..."
	@mkdir -p $(STORAGE_DIR) $(DATA_DIR)
	./bin/$(BINARY_NAME)

migrate: build ## Run database migrations
	@echo "Running migrations..."
	@mkdir -p $(DATA_DIR)
	./bin/$(BINARY_NAME) -migrate

clean: ## Clean up generated files
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(DATA_DIR)/*.db
	rm -rf $(STORAGE_DIR)/*
	docker-compose down -v
	docker system prune -f

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# API testing targets
test-api: ## Test API endpoints (requires running server)
	@echo "Testing API endpoints..."
	@echo "Health check:"
	curl -s http://localhost:8080/healthz | jq .
	@echo "\nMetrics:"
	curl -s http://localhost:8080/metrics | jq .
	@echo "\nCreating test job:"
	curl -s -X POST http://localhost:8080/api/v1/create-ringtone \
		-H "Content-Type: application/json" \
		-d '{"source_url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","options":{"format":"mp3"}}' | jq .

test-callback: ## Test n8n callback (requires JOB_ID env var)
	@echo "Testing n8n callback for job: $(JOB_ID)"
	curl -s -X POST http://localhost:8080/api/v1/n8n-callback \
		-H "Content-Type: application/json" \
		-H "X-Webhook-Token: your-secure-secret-here" \
		-d '{"job_id":"$(JOB_ID)","status":"completed","file_path":"$(JOB_ID).mp3","metadata":{"duration":30}}' | jq .

# Development workflow targets
dev-reset: clean ## Reset development environment
	@echo "Resetting development environment..."
	docker-compose down -v
	@mkdir -p $(STORAGE_DIR) $(DATA_DIR)
	$(MAKE) dev

watch: ## Run with file watching (requires air)
	@echo "Starting file watcher..."
	air

security-scan: ## Run security scan
	@echo "Running security scan..."
	go install github.com/securecodewarrior/github-action-add-sarif@latest
	gosec ./...
