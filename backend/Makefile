.PHONY: help test test-v test-cover test-integration fmt fmt-check vet lint lint-install check build run clean deps mod-tidy ci pre-commit up down docker-restart docker-build docker-logs docker-ps docker-clean swagger

BINARY_NAME = service

.DEFAULT_GOAL := help

help:
	@echo "Available commands:"
	@echo "  make test              - Run unit tests"
	@echo "  make test-v            - Unit tests with verbose output"
	@echo "  make test-cover        - Test coverage (handlers, usecases, middleware, domain)"
	@echo "  make test-integration  - Run integration tests (Docker required)"
	@echo "  make fmt               - Format code"
	@echo "  make fmt-check         - Check code formatting"
	@echo "  make vet               - Run go vet"
	@echo "  make lint              - Run golangci-lint"
	@echo "  make check             - All checks (fmt, vet, lint, test)"
	@echo "  make build             - Build binary"
	@echo "  make run               - Run application locally"
	@echo "  make clean             - Remove generated files"
	@echo "  make deps              - Install dependencies"
	@echo "  make mod-tidy          - Tidy go.mod"
	@echo "  make ci                - CI pipeline (fmt, vet, lint, test, cover)"
	@echo "  make pre-commit        - Pre-commit checks (fmt, vet, test)"
	@echo ""
	@echo "Docker commands:"
	@echo "  make up                - Start project with all dependencies"
	@echo "  make down              - Stop project"
	@echo "  make docker-restart    - Restart containers"
	@echo "  make docker-build      - Rebuild images"
	@echo "  make docker-logs       - Show container logs"
	@echo "  make docker-ps         - Show container status"
	@echo "  make docker-clean      - Stop and remove everything (incl. volumes)"
	@echo "  make swagger           - Start Swagger UI"

# === Testing ===

test:
	go test -short -timeout 30s ./... -count=1

test-v:
	go test -short -timeout 30s ./... -v -count=1

COVER_PKGS = ./internal/api/handler/... ./internal/usecase/... ./internal/infrastructure/middleware/... ./internal/domain/...

test-cover:
	go test -short -cover $(COVER_PKGS) -count=1
	go test -short -coverprofile=coverage.out $(COVER_PKGS) -count=1
	go tool cover -func=coverage.out

test-integration:
	@echo "Running integration tests..."
	go test -v -timeout 120s -tags integration ./tests/integration/... -count=1

# === Code quality ===

fmt:
	gofmt -w -s .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run: make fmt" && gofmt -l . && exit 1)

vet:
	go vet ./...

lint-install:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: https://golangci-lint.run/welcome/install/" && exit 1)

lint: lint-install
	golangci-lint run --timeout=5m ./...

check: fmt-check vet lint test

# === Build and run ===

build:
	go build -o $(BINARY_NAME) ./cmd/api

run:
	go run ./cmd/api/main.go

clean:
	rm -f $(BINARY_NAME) coverage.out

# === Dependencies ===

deps:
	go mod download
	go mod verify

mod-tidy:
	go mod tidy

# === CI ===

ci: fmt-check vet lint test test-cover

pre-commit: fmt-check vet test

# === Docker ===

up:
	docker compose -f deployments/docker-compose.yml up --build

down:
	docker compose -f deployments/docker-compose.yml down -v

docker-restart:
	docker compose -f deployments/docker-compose.yml restart

docker-build:
	docker compose -f deployments/docker-compose.yml build --no-cache

docker-logs:
	docker compose -f deployments/docker-compose.yml logs -f

docker-ps:
	docker compose -f deployments/docker-compose.yml ps

docker-clean:
	@echo "Stopping and removing containers, volumes and networks..."
	docker compose -f deployments/docker-compose.yml down -v --remove-orphans

swagger:
	docker compose -f deployments/docker-compose.yml up -d swagger-ui
