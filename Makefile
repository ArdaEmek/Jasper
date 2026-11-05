# Simple Makefile for a Go project

# Build the application
build:
	@echo "Building..."
	@go build -o main.exe cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

# Apply pending database migrations
migrate:
	@dbmate up

# Show database migration status
migrate-status:
	@dbmate status

# Create DB and cache container
docker-run:
	@docker compose up --build

# Shutdown DB and cache container
docker-down:
	@docker compose down

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Integration tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binaries
clean:
	@powershell -NoProfile -Command "Remove-Item -Force -ErrorAction SilentlyContinue main.exe, main"

# Live reload
watch:
	@powershell -ExecutionPolicy Bypass -Command "if (Get-Command air -ErrorAction SilentlyContinue) { \
		air; \
		Write-Output 'Watching...'; \
	} else { \
		Write-Output 'Installing air...'; \
		go install github.com/air-verse/air@latest; \
		air; \
		Write-Output 'Watching...'; \
	}"

.PHONY: all build run migrate migrate-status test clean watch docker-run docker-down itest

all: build test
