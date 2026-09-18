.PHONY: build run test test-e2e vet clean dev db-up db-down

# Variables
BINARY_NAME=tablehub-cloud
CMD_PATH=./cmd/server
BUILD_DIR=./bin

# Build the cloud server binary
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "✅ Built at $(BUILD_DIR)/$(BINARY_NAME)"

# Run the server directly (development)
run:
	go run $(CMD_PATH) serve

# Run all tests with race detection
test:
	go test ./... -v -race -count=1

# Run End-to-End tests (requires Docker)
test-e2e:
	go test -tags e2e ./tests/e2e/... -v -race

# Run go vet for static analysis
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Start backend services via docker-compose (DB, Auth, NATS)
db-up:
	docker compose -f docker-compose.yml up -d
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "✅ PostgreSQL is running on port 5432"

# Stop PostgreSQL
db-down:
	docker compose -f docker-compose.yml down

# Full dev environment (DB + server)
dev: db-up run
