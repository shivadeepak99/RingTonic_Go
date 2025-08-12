# RingTonic Backend API Documentation

This document describes the REST API endpoints for the RingTonic backend service.

## Base URL

```
Development: http://localhost:8080
Production: https://api.ringtonic.com
```

## Authentication

Most endpoints are public. The n8n callback endpoint requires authentication via the `X-Webhook-Token` header.

## Endpoints

### Health & Monitoring

#### GET /healthz

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-08-12T10:30:00Z",
  "version": "1.0.0"
}
```

#### GET /metrics

Basic service metrics.

**Response:**
```json
{
  "job_stats": {
    "queued": 5,
    "processing": 2,
    "completed": 150,
    "failed": 3
  },
  "uptime": "2h30m15s"
}
```

### Job Management

#### POST /api/v1/create-ringtone

Creates a new ringtone generation job.

**Request Body:**
```json
{
  "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
  "user_id": "optional-user-id",
  "options": {
    "start_seconds": 10,
    "duration_seconds": 20,
    "fade_in": true,
    "fade_out": true,
    "format": "mp3"
  }
}
```

**Response (202 Accepted):**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "queued",
  "poll_url": "/api/v1/job-status/550e8400-e29b-41d4-a716-446655440000"
}
```

**Error Responses:**
- `400` - Invalid request (missing source_url, invalid URL format)
- `500` - Internal server error

#### GET /api/v1/job-status/{jobID}

Retrieves the current status of a job.

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "created_at": "2025-08-12T10:00:00Z",
  "updated_at": "2025-08-12T10:02:30Z",
  "download_url": "/download/550e8400-e29b-41d4-a716-446655440000.mp3"
}
```

**Status Values:**
- `queued` - Job is waiting to be processed
- `processing` - Job is currently being processed by n8n
- `completed` - Job completed successfully, file ready for download
- `failed` - Job failed, check error field

**Error Responses:**
- `404` - Job not found
- `500` - Internal server error

### File Downloads

#### GET /download/{filename}

Downloads a generated ringtone file.

**Response:**
- `200` - File content with appropriate headers
- `403` - File not available (job not completed)
- `404` - File not found

**Response Headers:**
```
Content-Type: audio/mpeg
Content-Disposition: attachment; filename="ringtone.mp3"
Content-Length: 512000
Cache-Control: public, max-age=3600
```

### Internal Endpoints

#### POST /api/v1/n8n-callback

Internal endpoint for n8n workflow callbacks. Requires authentication.

**Headers:**
```
X-Webhook-Token: your-secure-secret-here
Content-Type: application/json
```

**Request Body:**
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

**Response:**
```json
{
  "status": "ok"
}
```

**Error Responses:**
- `401` - Invalid or missing webhook token
- `400` - Invalid request body
- `500` - Internal server error

## Error Response Format

All error responses follow this format:

```json
{
  "error": "Human readable error message",
  "code": "MACHINE_READABLE_ERROR_CODE"
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `INVALID_JSON` | Request body is not valid JSON |
| `MISSING_SOURCE_URL` | source_url field is required |
| `INVALID_URL` | URL format is invalid |
| `JOB_NOT_FOUND` | Job ID does not exist |
| `FILE_NOT_FOUND` | Requested file does not exist |
| `FILE_NOT_AVAILABLE` | File exists but job is not completed |
| `MISSING_TOKEN` | Webhook token header is missing |
| `INVALID_TOKEN` | Webhook token is incorrect |
| `INTERNAL_ERROR` | Generic internal server error |

## Rate Limiting

The API implements basic rate limiting:
- 100 requests per minute per IP for public endpoints
- No rate limiting for authenticated internal endpoints

## CORS

CORS is enabled for all origins in development. In production, configure appropriate origins.

## Examples

### Creating a Ringtone

```bash
curl -X POST http://localhost:8080/api/v1/create-ringtone \
  -H "Content-Type: application/json" \
  -d '{
    "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "options": {
      "start_seconds": 10,
      "duration_seconds": 20,
      "format": "mp3"
    }
  }'
```

### Checking Job Status

```bash
curl http://localhost:8080/api/v1/job-status/550e8400-e29b-41d4-a716-446655440000
```

### Downloading Ringtone

```bash
curl -O http://localhost:8080/download/550e8400-e29b-41d4-a716-446655440000.mp3
```

### Simulating n8n Callback

```bash
curl -X POST http://localhost:8080/api/v1/n8n-callback \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Token: your-secure-secret-here" \
  -d '{
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "file_path": "550e8400-e29b-41d4-a716-446655440000.mp3",
    "metadata": {
      "duration": 23
    }
  }'
```
