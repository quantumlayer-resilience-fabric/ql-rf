# QuantumLayer Resilience Fabric - Makefile
# Build, test, and development commands

.PHONY: all build test lint fmt clean dev dev-down \
        build-api build-connectors build-drift build-orchestrator \
        run-api run-connectors run-drift run-orchestrator \
        migrate-up migrate-down migrate-create sqlc-generate \
        docker-build docker-push \
        opa-test opa-fmt

# Variables
BINARY_DIR := bin
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags "-s -w -X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"

# Docker
DOCKER_REGISTRY ?= ghcr.io/quantumlayerhq
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'latest')

# Database
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/qlrf?sslmode=disable
MIGRATIONS_DIR := migrations

# Default target
all: lint test build

# ============================================================================
# Development
# ============================================================================

## dev: Start local development environment
dev:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo "Development environment is ready!"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis:      localhost:6379"
	@echo "  Kafka:      localhost:9092"

## dev-down: Stop local development environment
dev-down:
	docker compose down -v

## dev-logs: Show logs from development environment
dev-logs:
	docker compose logs -f

# ============================================================================
# Build
# ============================================================================

## build: Build all services
build: build-api build-connectors build-drift build-orchestrator

## build-api: Build API service
build-api:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/api ./services/api/cmd/api

## build-connectors: Build connectors service
build-connectors:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/connectors ./services/connectors/cmd/connectors

## build-drift: Build drift service
build-drift:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/drift ./services/drift/cmd/drift

## build-orchestrator: Build AI orchestrator service
build-orchestrator:
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_DIR)/orchestrator ./services/orchestrator/cmd/orchestrator

# ============================================================================
# Run (for local development)
# ============================================================================

## run-api: Run API service locally
run-api:
	$(GO) run ./services/api/cmd/api

## run-connectors: Run connectors service locally
run-connectors:
	$(GO) run ./services/connectors/cmd/connectors

## run-drift: Run drift service locally
run-drift:
	$(GO) run ./services/drift/cmd/drift

## run-orchestrator: Run AI orchestrator service locally
run-orchestrator:
	$(GO) run ./services/orchestrator/cmd/orchestrator

# ============================================================================
# Testing
# ============================================================================

## test: Run all tests
test:
	$(GO) test ./... -v

## test-unit: Run unit tests only (no integration tests)
test-unit:
	$(GO) test ./... -short -v

## test-integration: Run integration tests (requires docker-compose)
test-integration:
	$(GO) test ./tests/integration/... -tags=integration -v -timeout 10m

## test-e2e: Run full end-to-end tests (requires full environment)
test-e2e:
	$(GO) test ./tests/integration/... -tags=integration -v -timeout 15m -count=1

## test-coverage: Run tests with coverage report
test-coverage:
	$(GO) test ./... -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race detector
test-race:
	$(GO) test ./... -race -v

# ============================================================================
# Code Quality
# ============================================================================

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format code with gofmt
fmt:
	$(GO) fmt ./...
	goimports -w .

## vet: Run go vet
vet:
	$(GO) vet ./...

## tidy: Run go mod tidy on all modules
tidy:
	$(GO) mod tidy
	cd services/api && $(GO) mod tidy
	cd services/connectors && $(GO) mod tidy
	cd services/drift && $(GO) mod tidy
	cd services/orchestrator && $(GO) mod tidy

# ============================================================================
# Database
# ============================================================================

## migrate-up: Run all database migrations
migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

## migrate-down: Rollback the last migration
migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

## migrate-drop: Drop all tables (DANGER!)
migrate-drop:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" drop -f

## migrate-create: Create a new migration (usage: make migrate-create NAME=create_users)
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Error: NAME is required. Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

## migrate-version: Show current migration version
migrate-version:
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

## sqlc-generate: Generate type-safe SQL code
sqlc-generate:
	sqlc generate

# ============================================================================
# Docker
# ============================================================================

## docker-build: Build all Docker images
docker-build: docker-build-api docker-build-connectors docker-build-drift docker-build-orchestrator

## docker-build-api: Build API Docker image
docker-build-api:
	docker build -t $(DOCKER_REGISTRY)/ql-rf-api:$(VERSION) -f services/api/Dockerfile .

## docker-build-connectors: Build connectors Docker image
docker-build-connectors:
	docker build -t $(DOCKER_REGISTRY)/ql-rf-connectors:$(VERSION) -f services/connectors/Dockerfile .

## docker-build-drift: Build drift Docker image
docker-build-drift:
	docker build -t $(DOCKER_REGISTRY)/ql-rf-drift:$(VERSION) -f services/drift/Dockerfile .

## docker-build-orchestrator: Build AI orchestrator Docker image
docker-build-orchestrator:
	docker build -t $(DOCKER_REGISTRY)/ql-rf-orchestrator:$(VERSION) -f services/orchestrator/Dockerfile .

## docker-push: Push all Docker images to registry
docker-push:
	docker push $(DOCKER_REGISTRY)/ql-rf-api:$(VERSION)
	docker push $(DOCKER_REGISTRY)/ql-rf-connectors:$(VERSION)
	docker push $(DOCKER_REGISTRY)/ql-rf-drift:$(VERSION)
	docker push $(DOCKER_REGISTRY)/ql-rf-orchestrator:$(VERSION)

# ============================================================================
# OPA Policy Management
# ============================================================================

## opa-test: Test OPA policies
opa-test:
	opa test policy/ -v

## opa-fmt: Format OPA policies
opa-fmt:
	opa fmt -w policy/

## opa-check: Check OPA policy syntax
opa-check:
	opa check policy/

# ============================================================================
# Tools Installation
# ============================================================================

## install-tools: Install development tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install golang.org/x/tools/cmd/goimports@latest

# ============================================================================
# Cleanup
# ============================================================================

## clean: Remove build artifacts
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# ============================================================================
# Help
# ============================================================================

## help: Show this help message
help:
	@echo "QuantumLayer Resilience Fabric"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
