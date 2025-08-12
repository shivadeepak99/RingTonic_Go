# RingTonic Backend - Development Environment Check (PowerShell)

Write-Host "🔍 RingTonic Backend - Development Environment Check" -ForegroundColor Cyan
Write-Host "==================================================" -ForegroundColor Cyan

function Write-Success {
    param($Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-ErrorMsg {
    param($Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param($Message)
    Write-Host "ℹ $Message" -ForegroundColor Yellow
}

# Check Go version
Write-Info "Checking Go version..."
$goVersion = go version 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Success "Go is installed: $goVersion"
} else {
    Write-ErrorMsg "Go is not working properly"
    exit 1
}

# Test build without CGO
Write-Info "Testing build without CGO (syntax check)..."
$env:CGO_ENABLED = "0"
go build ./... 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Success "Code compiles successfully (no CGO)"
} else {
    Write-ErrorMsg "Code has compilation errors"
    exit 1
}

# Test jobs package (no SQLite dependency)
Write-Info "Running jobs tests (no SQLite dependency)..."
go test ./internal/jobs/jobs_test -v 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Success "Jobs tests pass"
} else {
    Write-ErrorMsg "Jobs tests failed"
}

# Build main binary
Write-Info "Building main server binary..."
New-Item -ItemType Directory -Force -Path "bin" | Out-Null
go build -o bin/server.exe cmd/server/main.go 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Success "Main binary built successfully"
} else {
    Write-ErrorMsg "Failed to build main binary"
}

# Test CGO build
Write-Info "Testing CGO build (required for SQLite)..."
$env:CGO_ENABLED = "1"
go build ./cmd/server 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Success "CGO build successful - SQLite will work"
    
    Write-Info "Running CGO-dependent tests..."
    go test ./internal/store/store_test 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Store tests pass with CGO"
    } else {
        Write-ErrorMsg "Store tests failed - this is expected with 32-bit GCC"
    }
} else {
    Write-ErrorMsg "CGO build failed - SQLite tests will not work"
    Write-Info "This is expected with 32-bit GCC on 64-bit Go"
}

Write-Host ""
Write-Info "Development environment check complete!"
Write-Host ""
Write-Host "📋 Development Options:" -ForegroundColor Cyan
Write-Host "   1. ✅ Use Docker: docker-compose up --build" -ForegroundColor Green
Write-Host "   2. ✅ Mock tests: go test ./internal/jobs/jobs_test" -ForegroundColor Green  
Write-Host "   3. ✅ Build binary: go build cmd/server/main.go" -ForegroundColor Green
Write-Host "   4. ⚠️  Full tests need 64-bit GCC or Docker" -ForegroundColor Yellow
Write-Host ""
Write-Host "🚀 Quick Start: docker-compose up --build" -ForegroundColor Green
