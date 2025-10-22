.PHONY: test-unit test-integration test-all migrate-up migrate-down swagger gen-mocks run build docker-build docker-up docker-down k6-constant k6-spike k6-stress k6-soak k6-multi k6-all

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

migrate-down:
	@echo "Rolling back database migrations..."
	go run cmd/migrator/main.go -dsn "postgres://postgres:postgres@localhost:5433/wallet?sslmode=disable" -migrations-path "migrations" -down

docker-build:
	@echo "Building Docker image..."
	docker-compose build

docker-up:
	@echo "Starting Docker environment..."
	docker-compose up --build -d

docker-down:
	@echo "Stopping Docker environment..."
	docker-compose down -v

swagger:
	@echo "Generating Swagger documentation..."
	swag init --parseDependency --parseInternal --output .static/swagger --outputTypes json -g ./cmd/wallet/main.go

gen-mocks:
	@echo "Generating mocks..."
	go generate ./internal/repository/...
	go generate ./internal/service/...
	go generate ./internal/api/handlers/...

run:
	@echo "Running application..."
	go run cmd/wallet/main.go

build:
	@echo "Building binaries..."
	go build -o bin/wallet ./cmd/wallet
	go build -o bin/migrator ./cmd/migrator

k6-constant:
	@echo "Running k6 constant load test..."
	@docker run --rm -i --network=itk_default -v $$(pwd)/tests/k6:/tests -e BASE_URL=http://wallet-api:8080 grafana/k6 run /tests/constant_load.js

k6-spike:
	@echo "Running k6 spike test..."
	@docker run --rm -i --network=itk_default -v $$(pwd)/tests/k6:/tests -e BASE_URL=http://wallet-api:8080 grafana/k6 run /tests/spike_test.js

k6-stress:
	@echo "Running k6 stress test..."
	@docker run --rm -i --network=itk_default -v $$(pwd)/tests/k6:/tests -e BASE_URL=http://wallet-api:8080 grafana/k6 run /tests/stress_test.js

k6-soak:
	@echo "Running k6 soak test (30+ minutes)..."
	@docker run --rm -i --network=itk_default -v $$(pwd)/tests/k6:/tests -e BASE_URL=http://wallet-api:8080 grafana/k6 run /tests/soak_test.js

k6-multi:
	@echo "Running k6 multi-wallet test..."
	@docker run --rm -i --network=itk_default -v $$(pwd)/tests/k6:/tests -e BASE_URL=http://wallet-api:8080 grafana/k6 run /tests/multi_wallet_test.js

k6-all:
	@echo "Running all k6 tests (excluding soak)..."
	$(MAKE) k6-multi
	$(MAKE) k6-constant
	$(MAKE) k6-spike
	$(MAKE) k6-stress

