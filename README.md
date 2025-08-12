# RingTonic Go Backend

A production-ready Go backend service for the RingTonic "paste-link → 1-click ringtone" application. This service handles job orchestration, file management, and integration with n8n workflows.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.20+ (for local development)
- Make (optional, for convenience commands)

### Environment Setup
1. Clone the repository
2. Copy environment file:
   ```bash
   cp .env.example .env
   ```
3. Start the services:
   ```bash
   docker-compose up --build
   ```

The backend will be available at `http://localhost:8080`

### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `BACKEND_PORT` | HTTP server port | `8080` |
| `DB_PATH` | SQLite database file path | `./data/ringtonic.db` |
| `STORAGE_PATH` | Local file storage directory | `./storage` |
| `N8N_WEBHOOK_URL` | n8n webhook endpoint | `http://n8n:5678/webhook/ringtonic` |
| `N8N_WEBHOOK_SECRET` | Shared secret for n8n callbacks | `your-secure-secret-here` |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | `info` |

## API Endpoints

### Create Ringtone Job
```bash
curl -X POST http://localhost:8080/api/v1/create-ringtone \
  -H "Content-Type: application/json" \
  -d '{
    "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "user_id": "user123",
    "options": {
      "start_seconds": 10,
      "duration_seconds": 20,
      "fade_in": true,
      "fade_out": true,
      "format": "mp3"
    }
  }'
```

### Check Job Status
```bash
curl http://localhost:8080/api/v1/job-status/{job_id}
```

### Download Ringtone
```bash
curl -O http://localhost:8080/download/{ringtone_file_name}
```

### Health Check
```bash
curl http://localhost:8080/healthz
```

## Development Commands

### Using Make
```bash
make dev     # Start development environment
make test    # Run unit tests
make clean   # Clean up generated files
make lint    # Run linting
```

### Manual Commands
```bash
# Run tests
go test ./...

# Run with live reload (requires air)
air

# Build binary
go build -o bin/server cmd/server/main.go

# Run migrations
go run cmd/server/main.go -migrate
```

## Testing

### Unit Tests
```bash
go test ./... -v
```

### Integration Test Example
```bash
# Start services
docker-compose up -d

# Test create job
curl -X POST http://localhost:8080/api/v1/create-ringtone \
  -H "Content-Type: application/json" \
  -d '{"source_url": "https://www.youtube.com/watch?v=test"}'

# Test n8n callback simulation
curl -X POST http://localhost:8080/api/v1/n8n-callback \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Token: your-secure-secret-here" \
  -d '{
    "job_id": "your-job-id",
    "status": "completed",
    "file_path": "test.mp3",
    "metadata": {"duration": 30}
  }'
```

## Architecture

```
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── api/            # HTTP handlers and routes
│   ├── config/         # Configuration management
│   ├── files/          # File storage operations
│   ├── jobs/           # Job management and state machine
│   ├── log/            # Structured logging
│   ├── n8n/            # n8n webhook client
│   └── store/          # Database operations
├── migrations/         # SQL migration scripts
├── test/              # Unit and integration tests
├── docker-compose.yml # Local development setup
└── Dockerfile         # Container build instructions
```

## Database Schema

### Jobs Table
- `id` (UUID) - Primary key
- `source_url` (TEXT) - Original video URL
- `user_id` (TEXT) - Optional user identifier
- `status` (TEXT) - Job status: queued, processing, completed, failed
- `created_at` (DATETIME) - Job creation timestamp
- `updated_at` (DATETIME) - Last update timestamp
- `attempts` (INTEGER) - Webhook trigger attempts
- `n8n_payload` (TEXT) - JSON payload sent to n8n
- `error_message` (TEXT) - Error details if failed

### Ringtones Table
- `id` (INTEGER) - Primary key
- `job_id` (TEXT) - Foreign key to jobs table
- `file_name` (TEXT) - Generated filename
- `file_path` (TEXT) - Full file path
- `format` (TEXT) - Audio format (mp3, m4a)
- `duration_seconds` (INTEGER) - Audio duration
- `created_at` (DATETIME) - File creation timestamp

## n8n Integration

### Webhook Payload (Backend → n8n)
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "options": {
    "start_seconds": 10,
    "duration_seconds": 20,
    "fade_in": true,
    "fade_out": true,
    "format": "mp3"
  },
  "callback_url": "http://backend:8080/api/v1/n8n-callback"
}
```

### Callback Payload (n8n → Backend)
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "file_path": "550e8400-e29b-41d4-a716-446655440000.mp3",
  "metadata": {
    "duration": 23,
    "original_title": "Never Gonna Give You Up",
    "file_size": 512000
  }
}
```

### Required Headers
- `X-Webhook-Token`: Must match `N8N_WEBHOOK_SECRET`
- `Content-Type`: `application/json`

## Deployment

### Docker Production Build
```bash
docker build -t ringtonic-backend .
docker run -p 8080:8080 --env-file .env ringtonic-backend
```

### Environment-Specific Configs
- Development: Use `docker-compose.yml`
- Production: Set appropriate environment variables and external database
- Testing: Use in-memory SQLite database

## Security Considerations

1. **Webhook Verification**: All n8n callbacks are verified using HMAC or token comparison
2. **File Access Control**: Downloads are only allowed for completed jobs
3. **Input Validation**: All API inputs are validated and sanitized
4. **Rate Limiting**: Basic rate limiting implemented (configurable)
5. **CORS**: Configured for frontend integration

## Monitoring & Observability

- Health endpoint: `/healthz`
- Metrics endpoint: `/metrics` (basic counters)
- Structured logging with job context
- Request ID tracing
- Performance metrics for webhook calls

## Troubleshooting

### Common Issues
1. **Database locked**: Check if another process is accessing SQLite
2. **Webhook failures**: Verify n8n service is running and reachable
3. **File not found**: Check storage path permissions and disk space
4. **Authentication errors**: Verify webhook secret configuration

### Logs
```bash
# View backend logs
docker-compose logs backend

# Follow logs
docker-compose logs -f backend
```

## Contributing

1. Run tests before submitting: `make test`
2. Follow Go formatting: `go fmt ./...`
3. Update documentation for API changes
4. Add tests for new features

## License

MIT License - see LICENSE file for details