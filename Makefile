.PHONY: test build fmt tidy lint docker-up docker-down

test:
	go test -count=1 ./...

build:
	go build ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

lint:
	golangci-lint run --timeout=5m

docker-up:
	docker compose -f deployments/docker-compose.yml up --build

docker-down:
	docker compose -f deployments/docker-compose.yml down -v

