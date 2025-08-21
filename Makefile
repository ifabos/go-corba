.PHONY: build test test-sync clean run-sync-example help

# Build the project
build:
	go build -v ./...

# Run all tests
test:
	go test -v ./corba ./idl

# Run synchronization-specific tests
test-sync:
	go test -v ./corba -run "Test.*Sync.*|Test.*Change.*|Test.*Queue.*"

# Run tests with integration tests (requires Redis)
test-integration:
	go test -v ./corba -run "TestSynchronizedEventService_Integration"

# Clean build artifacts
clean:
	go clean ./...
	rm -f examples/sync/sync_example

# Run synchronization example (requires Redis)
run-sync-example:
	cd examples/sync && go run main.go example-instance

# Run multiple instances for testing (requires Redis)
run-sync-demo:
	@echo "Starting 3 instances of the sync example..."
	@echo "Make sure Redis is running on localhost:6379"
	cd examples/sync && go run main.go instance1 &
	cd examples/sync && go run main.go instance2 &
	cd examples/sync && go run main.go instance3 &
	@echo "All instances started. Press Ctrl+C to stop."

# Install dependencies
deps:
	go mod download
	go mod tidy

# Lint the code (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format the code
fmt:
	go fmt ./...

# Generate documentation
docs:
	godoc -http=:6060

# Run Redis in Docker for testing
redis-docker:
	docker run --name redis-corba -p 6379:6379 -d redis:7-alpine
	@echo "Redis started on localhost:6379"

# Stop Redis Docker container
redis-stop:
	docker stop redis-corba
	docker rm redis-corba

# Help
help:
	@echo "Available targets:"
	@echo "  build            - Build the project"
	@echo "  test             - Run all tests"
	@echo "  test-sync        - Run synchronization tests"
	@echo "  test-integration - Run integration tests (requires Redis)"
	@echo "  clean            - Clean build artifacts"
	@echo "  run-sync-example - Run single sync example instance"
	@echo "  run-sync-demo    - Run multiple sync instances for demo"
	@echo "  deps             - Install/update dependencies"
	@echo "  lint             - Lint the code"
	@echo "  fmt              - Format the code"
	@echo "  docs             - Generate documentation server"
	@echo "  redis-docker     - Start Redis in Docker"
	@echo "  redis-stop       - Stop Redis Docker container"
	@echo "  help             - Show this help"