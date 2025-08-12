# n8n Integration Guide for RingTonic Backend

This document provides instructions for Person B (n8n workflow developer) on how to integrate with the RingTonic Go backend.

## Overview

The RingTonic backend triggers n8n workflows via webhook and expects callbacks when processing is complete. This document covers the exact payloads and headers required for integration.

## Workflow Trigger

### Webhook URL
Your n8n instance should expose a webhook at:
```
http://n8n:5678/webhook/ringtonic
```

### Incoming Payload (Backend → n8n)

When a user creates a ringtone job, the backend will POST the following JSON to your webhook:

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

### Headers Sent by Backend
```
Content-Type: application/json
X-Webhook-Token: your-secure-secret-here
X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

## Processing Workflow

Your n8n workflow should:

1. **Receive the webhook** with the payload above
2. **Extract audio** using yt-dlp:
   ```bash
   yt-dlp -x --audio-format mp3 --audio-quality 0 -o "/storage/%(id)s.%(ext)s" "${source_url}"
   ```
3. **Process audio** using FFmpeg (if trimming/effects needed):
   ```bash
   ffmpeg -i input.mp3 -ss ${start_seconds} -t ${duration_seconds} -c copy output.mp3
   ```
4. **Save file** to storage directory with filename: `${job_id}.mp3`
5. **Send callback** to the backend (see below)

## Callback Requirements

### Callback URL
Send POST request to:
```
http://backend:8080/api/v1/n8n-callback
```

### Required Headers
```
Content-Type: application/json
X-Webhook-Token: your-secure-secret-here
```

⚠️ **Important**: The `X-Webhook-Token` must match the `N8N_WEBHOOK_SECRET` environment variable set in the backend.

### Success Callback Payload

When processing completes successfully:

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

### Failure Callback Payload

When processing fails:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "failed",
  "metadata": {
    "error": "Failed to download video: Video not available"
  }
}
```

## Field Descriptions

### Incoming Fields (Backend → n8n)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `job_id` | string | Yes | Unique identifier for this job |
| `source_url` | string | Yes | Video URL to process |
| `callback_url` | string | Yes | URL to send completion callback |
| `options.start_seconds` | int | No | Start time for audio extraction |
| `options.duration_seconds` | int | No | Duration of output audio |
| `options.fade_in` | bool | No | Apply fade-in effect |
| `options.fade_out` | bool | No | Apply fade-out effect |
| `options.format` | string | No | Output format (default: mp3) |

### Callback Fields (n8n → Backend)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `job_id` | string | Yes | Same job_id from incoming payload |
| `status` | string | Yes | Either "completed" or "failed" |
| `file_path` | string | Yes* | Filename in storage (*required for completed) |
| `metadata.duration` | int | No | Audio duration in seconds |
| `metadata.original_title` | string | No | Original video title |
| `metadata.file_size` | int | No | File size in bytes |
| `metadata.error` | string | No | Error message (for failed status) |

## File Storage

- Store all generated files in the `/storage` directory
- Use the `job_id` as the filename: `${job_id}.mp3`
- The backend expects to find files at `/app/storage/${filename}`

## Error Handling

Handle these common error scenarios:

1. **Video not available**: Send failed callback with error message
2. **Audio extraction failed**: Send failed callback with yt-dlp error
3. **File processing failed**: Send failed callback with FFmpeg error
4. **Network issues**: Retry callback up to 3 times with exponential backoff

## Testing

### Test Webhook Trigger

You can test the n8n webhook by sending:

```bash
curl -X POST http://localhost:5678/webhook/ringtonic \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Token: your-secure-secret-here" \
  -d '{
    "job_id": "test-123",
    "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
    "options": {"format": "mp3"},
    "callback_url": "http://backend:8080/api/v1/n8n-callback"
  }'
```

### Test Callback

Test the callback endpoint:

```bash
curl -X POST http://backend:8080/api/v1/n8n-callback \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Token: your-secure-secret-here" \
  -d '{
    "job_id": "test-123",
    "status": "completed",
    "file_path": "test-123.mp3",
    "metadata": {"duration": 30}
  }'
```

## Example n8n Workflow Structure

```
1. Webhook Trigger
   ↓
2. Extract Variables (job_id, source_url, options)
   ↓
3. Execute yt-dlp Command
   ↓
4. [If successful] Execute FFmpeg Command (optional)
   ↓
5. [If successful] HTTP Request - Success Callback
   ↓
6. [If failed] HTTP Request - Failure Callback
```

## Environment Variables

Make sure these are set in your environment:

- `N8N_WEBHOOK_SECRET`: Shared secret for authentication
- Storage directory should be mounted to `/storage`

## Security Notes

1. Always verify the `X-Webhook-Token` header matches your secret
2. Validate the `job_id` format (should be UUID)
3. Sanitize file paths to prevent directory traversal
4. Implement timeouts for long-running processes

## Support

If you encounter issues:

1. Check backend logs for webhook delivery status
2. Verify the callback URL is reachable from n8n
3. Ensure the webhook token matches exactly
4. Test with simple payloads first
