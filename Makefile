.PHONY: test-unit test-integration test-all docker-up docker-down migrate-up gen-swagger gen-mocks run build

test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/...

test-integration:
	@echo "Running integration tests..."
	INTEGRATION_TESTS=true go test -v -race ./tests/integration/...

test-all: test-unit test-integration

migrate-up:
	@echo "Applying database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/wallet?sslmode=disable" -migrations-path "migrations"

docker-up:
	@echo "Starting Docker environment..."
	docker-compose up --build -d

docker-down:
	@echo "Stopping Docker environment..."
	docker-compose down -v

gen-swagger:
	@echo "Generating Swagger documentation..."
	swag init --parseDependency --parseInternal --output .static/swagger --outputTypes json -g ./cmd/wallet/main.go

gen-mocks:
	@echo "Generating mocks..."
	go generate ./internal/repository/...
	go generate ./internal/service/...

run:
	@echo "Running application..."
	go run cmd/wallet/main.go

build:
	@echo "Building binaries..."
	go build -o bin/wallet ./cmd/wallet
	go build -o bin/migrator ./cmd/migrator

