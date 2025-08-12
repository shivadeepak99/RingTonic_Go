#!/bin/bash
# Development environment check script

echo "🔍 RingTonic Backend - Development Environment Check"
echo "=================================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

# Check Go version
print_info "Checking Go version..."
go version
if [ $? -eq 0 ]; then
    print_status "Go is installed and working"
else
    print_error "Go is not working properly"
    exit 1
fi

# Check if we can build without CGO
print_info "Testing build without CGO (for syntax check)..."
export CGO_ENABLED=0
go build ./...
if [ $? -eq 0 ]; then
    print_status "Code compiles successfully (no CGO)"
else
    print_error "Code has compilation errors"
    exit 1
fi

# Test non-SQLite tests
print_info "Running tests that don't require SQLite..."
go test ./internal/jobs/jobs_test -v
if [ $? -eq 0 ]; then
    print_status "Jobs tests pass"
else
    print_error "Jobs tests failed"
fi

# Check if we can build the main binary
print_info "Building main server binary..."
go build -o bin/server cmd/server/main.go
if [ $? -eq 0 ]; then
    print_status "Main binary built successfully"
else
    print_error "Failed to build main binary"
fi

# Try to build with CGO for SQLite
print_info "Testing CGO build (required for SQLite)..."
export CGO_ENABLED=1
go build ./cmd/server 2>/dev/null
if [ $? -eq 0 ]; then
    print_status "CGO build successful - SQLite will work"
    
    # Run all tests if CGO works
    print_info "Running all tests with CGO..."
    go test ./... -short
    if [ $? -eq 0 ]; then
        print_status "All tests pass with CGO"
    else
        print_error "Some tests failed with CGO"
    fi
else
    print_error "CGO build failed - SQLite tests will not work"
    print_info "You can still run the server but may need to install a 64-bit GCC compiler"
    print_info "Consider installing TDM-GCC 64-bit or using WSL for development"
fi

echo ""
print_info "Development environment check complete!"
echo ""
echo "📋 Next steps:"
echo "   1. For production: Use Docker (CGO works in container)"
echo "   2. For local development: Install 64-bit GCC or use WSL"
echo "   3. Tests can run in mocking mode without SQLite"
echo "   4. Use 'make dev' to start with Docker Compose"
