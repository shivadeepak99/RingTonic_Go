#!/bin/bash
# Integration test script for RingTonic Backend

set -e

echo "🚀 Starting RingTonic Backend Integration Tests"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"
WEBHOOK_SECRET="your-secure-secret-here"

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Wait for server to be ready
wait_for_server() {
    print_info "Waiting for server to be ready..."
    for i in {1..30}; do
        if curl -s -f "$BASE_URL/healthz" > /dev/null 2>&1; then
            print_status "Server is ready"
            return 0
        fi
        sleep 1
    done
    print_error "Server failed to start within 30 seconds"
    exit 1
}

# Test health endpoint
test_health() {
    print_info "Testing health endpoint..."
    response=$(curl -s "$BASE_URL/healthz")
    if echo "$response" | grep -q "healthy"; then
        print_status "Health check passed"
    else
        print_error "Health check failed"
        echo "Response: $response"
        exit 1
    fi
}

# Test metrics endpoint
test_metrics() {
    print_info "Testing metrics endpoint..."
    response=$(curl -s "$BASE_URL/metrics")
    if echo "$response" | grep -q "job_stats"; then
        print_status "Metrics endpoint working"
    else
        print_error "Metrics endpoint failed"
        echo "Response: $response"
        exit 1
    fi
}

# Test create ringtone job
test_create_job() {
    print_info "Testing job creation..."
    response=$(curl -s -X POST "$BASE_URL/api/v1/create-ringtone" \
        -H "Content-Type: application/json" \
        -d '{
            "source_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
            "options": {
                "format": "mp3",
                "duration_seconds": 20
            }
        }')
    
    job_id=$(echo "$response" | jq -r '.job_id')
    if [[ "$job_id" != "null" && "$job_id" != "" ]]; then
        print_status "Job created successfully: $job_id"
        echo "$job_id" > /tmp/test_job_id
    else
        print_error "Job creation failed"
        echo "Response: $response"
        exit 1
    fi
}

# Test job status
test_job_status() {
    print_info "Testing job status endpoint..."
    job_id=$(cat /tmp/test_job_id)
    response=$(curl -s "$BASE_URL/api/v1/job-status/$job_id")
    
    status=$(echo "$response" | jq -r '.status')
    if [[ "$status" == "queued" || "$status" == "processing" ]]; then
        print_status "Job status endpoint working: $status"
    else
        print_error "Job status endpoint failed"
        echo "Response: $response"
        exit 1
    fi
}

# Test n8n callback simulation
test_n8n_callback() {
    print_info "Testing n8n callback..."
    job_id=$(cat /tmp/test_job_id)
    
    # Create a dummy file for testing
    echo "dummy audio content" > "./storage/${job_id}.mp3"
    
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/n8n-callback" \
        -H "Content-Type: application/json" \
        -H "X-Webhook-Token: $WEBHOOK_SECRET" \
        -d "{
            \"job_id\": \"$job_id\",
            \"status\": \"completed\",
            \"file_path\": \"${job_id}.mp3\",
            \"metadata\": {
                \"duration\": 20,
                \"file_size\": 1024
            }
        }")
    
    http_code="${response: -3}"
    if [[ "$http_code" == "200" ]]; then
        print_status "n8n callback test passed"
    else
        print_error "n8n callback test failed (HTTP $http_code)"
        exit 1
    fi
}

# Test download after completion
test_download() {
    print_info "Testing file download..."
    job_id=$(cat /tmp/test_job_id)
    
    # First check if job status shows download URL
    response=$(curl -s "$BASE_URL/api/v1/job-status/$job_id")
    download_url=$(echo "$response" | jq -r '.download_url')
    
    if [[ "$download_url" != "null" && "$download_url" != "" ]]; then
        print_status "Download URL available: $download_url"
        
        # Test actual download
        http_code=$(curl -s -w "%{http_code}" -o /tmp/test_download.mp3 "$BASE_URL$download_url")
        if [[ "$http_code" == "200" ]]; then
            print_status "File download successful"
        else
            print_error "File download failed (HTTP $http_code)"
            exit 1
        fi
    else
        print_error "Download URL not available"
        echo "Response: $response"
        exit 1
    fi
}

# Test invalid requests
test_invalid_requests() {
    print_info "Testing invalid request handling..."
    
    # Test invalid URL
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/create-ringtone" \
        -H "Content-Type: application/json" \
        -d '{"source_url": "invalid-url"}')
    
    http_code="${response: -3}"
    if [[ "$http_code" == "400" ]]; then
        print_status "Invalid URL handling works"
    else
        print_error "Invalid URL handling failed (expected 400, got $http_code)"
    fi
    
    # Test missing token on callback
    response=$(curl -s -w "%{http_code}" -X POST "$BASE_URL/api/v1/n8n-callback" \
        -H "Content-Type: application/json" \
        -d '{"job_id": "test", "status": "completed"}')
    
    http_code="${response: -3}"
    if [[ "$http_code" == "401" ]]; then
        print_status "Missing token handling works"
    else
        print_error "Missing token handling failed (expected 401, got $http_code)"
    fi
}

# Cleanup
cleanup() {
    print_info "Cleaning up test files..."
    rm -f /tmp/test_job_id /tmp/test_download.mp3
    rm -f ./storage/*.mp3
}

# Main test execution
main() {
    echo "🧪 RingTonic Backend Integration Tests"
    echo "======================================"
    
    # Wait for server
    wait_for_server
    
    # Run tests
    test_health
    test_metrics
    test_create_job
    test_job_status
    test_n8n_callback
    test_download
    test_invalid_requests
    
    # Cleanup
    cleanup
    
    echo ""
    print_status "All integration tests passed! 🎉"
    echo ""
    echo "✅ Backend is ready for integration with:"
    echo "   - n8n workflows"
    echo "   - Frontend applications"
    echo "   - Mobile apps"
}

# Run tests
main "$@"
