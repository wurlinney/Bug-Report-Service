.PHONY: help test test-v test-cover test-integration fmt fmt-check vet lint lint-install check build run clean deps mod-tidy ci pre-commit up down docker-restart docker-build docker-logs docker-ps docker-clean swagger

.DEFAULT_GOAL := help

help:
	@cd backend && make help
test:
	@cd backend && make test
test-v:
	@cd backend && make test-v
test-cover:
	@cd backend && make test-cover
test-integration:
	@cd backend && make test-integration
fmt:
	@cd backend && make fmt
fmt-check:
	@cd backend && make fmt-check
vet:
	@cd backend && make vet
lint:
	@cd backend && make lint
lint-install:
	@cd backend && make lint-install
check:
	@cd backend && make check
build:
	@cd backend && make build
run:
	@cd backend && make run
clean:
	@cd backend && make clean
deps:
	@cd backend && make deps
mod-tidy:
	@cd backend && make mod-tidy
ci:
	@cd backend && make ci
pre-commit:
	@cd backend && make pre-commit
up:
	@cd backend && make up
down:
	@cd backend && make down
docker-restart:
	@cd backend && make docker-restart
docker-build:
	@cd backend && make docker-build
docker-logs:
	@cd backend && make docker-logs
docker-ps:
	@cd backend && make docker-ps
docker-clean:
	@cd backend && make docker-clean
swagger:
	@cd backend && make swagger
