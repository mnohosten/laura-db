.PHONY: all build clean test server cli repair examples docker docker-build docker-run docker-stop docker-clean compose compose-up compose-up-monitoring compose-up-prod compose-down compose-down-volumes compose-logs compose-logs-all compose-restart help

# Default target
all: build

# Build everything
build: server cli repair examples

# Build main server
server:
	@echo "Building LauraDB server..."
	@mkdir -p bin
	@go build -o bin/laura-server ./cmd/server
	@echo "✓ Built: bin/laura-server"

# Build CLI tool
cli:
	@echo "Building LauraDB CLI..."
	@mkdir -p bin
	@go build -o bin/laura-cli ./cmd/laura-cli
	@echo "✓ Built: bin/laura-cli"

# Build repair tool
repair:
	@echo "Building LauraDB repair tool..."
	@mkdir -p bin
	@go build -o bin/laura-repair ./cmd/repair
	@echo "✓ Built: bin/laura-repair"

# Build all examples
examples:
	@echo "Building examples..."
	@mkdir -p bin
	@cd examples/basic && go build -o ../../bin/basic-example basic_usage.go
	@echo "✓ Built: bin/basic-example"
	@cd examples/full_demo && go build -o ../../bin/full-demo full_database_demo.go
	@echo "✓ Built: bin/full-demo"
	@cd examples/aggregation_demo && go build -o ../../bin/aggregation-demo main.go
	@echo "✓ Built: bin/aggregation-demo"
	@cd examples/import-export && go build -o ../../bin/import-export-demo main.go
	@echo "✓ Built: bin/import-export-demo"
	@cd examples/parallel-query && go build -o ../../bin/parallel-query-demo main.go
	@echo "✓ Built: bin/parallel-query-demo"
	@cd examples/compression-demo && go build -o ../../bin/compression-demo main.go
	@echo "✓ Built: bin/compression-demo"
	@cd examples/transaction-demo && go build -o ../../bin/transaction-demo main.go
	@echo "✓ Built: bin/transaction-demo"
	@cd examples/backup-demo && go build -o ../../bin/backup-demo main.go
	@echo "✓ Built: bin/backup-demo"
	@cd examples/mmap-demo && go build -o ../../bin/mmap-demo main.go
	@echo "✓ Built: bin/mmap-demo"
	@cd examples/lsm-demo && go build -o ../../bin/lsm-demo main.go
	@echo "✓ Built: bin/lsm-demo"
	@cd examples/savepoint-demo && go build -o ../../bin/savepoint-demo main.go
	@echo "✓ Built: bin/savepoint-demo"
	@cd examples/distributed-2pc && go build -o ../../bin/distributed-2pc-demo main.go
	@echo "✓ Built: bin/distributed-2pc-demo"
	@cd examples/repair-demo && go build -o ../../bin/repair-demo main.go
	@echo "✓ Built: bin/repair-demo"
	@cd examples/replication-demo && go build -o ../../bin/replication-demo main.go
	@echo "✓ Built: bin/replication-demo"
	@cd examples/replica-set-demo && go build -o ../../bin/replica-set-demo main.go
	@echo "✓ Built: bin/replica-set-demo"
	@cd examples/write-concern-demo && go build -o ../../bin/write-concern-demo main.go
	@echo "✓ Built: bin/write-concern-demo"
	@cd examples/read-preference-demo && go build -o ../../bin/read-preference-demo main.go
	@echo "✓ Built: bin/read-preference-demo"
	@cd examples/sharding-demo && go build -o ../../bin/sharding-demo main.go
	@echo "✓ Built: bin/sharding-demo"
	@cd examples/config-server-demo && go build -o ../../bin/config-server-demo main.go
	@echo "✓ Built: bin/config-server-demo"
	@cd examples/changestream-demo && go build -o ../../bin/changestream-demo main.go
	@echo "✓ Built: bin/changestream-demo"
	@cd examples/auth-demo && go build -o ../../bin/auth-demo main.go
	@echo "✓ Built: bin/auth-demo"
	@cd examples/prometheus-demo && go build -o ../../bin/prometheus-demo main.go
	@echo "✓ Built: bin/prometheus-demo"
	@cd examples/connstring-demo && go build -o ../../bin/connstring-demo main.go
	@echo "✓ Built: bin/connstring-demo"
	@cd examples/migration-demo && go build -o ../../bin/migration-demo main.go
	@echo "✓ Built: bin/migration-demo"
	@cd examples/tls-demo && go build -o ../../bin/tls-demo main.go
	@echo "✓ Built: bin/tls-demo"
	@cd examples/client-demo && go build -o ../../bin/client-demo main.go
	@echo "✓ Built: bin/client-demo"
	@cd examples/cursor-demo && go build -o ../../bin/cursor-demo main.go
	@echo "✓ Built: bin/cursor-demo"

# Run tests
test:
	@echo "Running tests..."
	@go test ./pkg/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./pkg/...

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@go test ./pkg/... -coverprofile=coverage.out
	@go tool cover -func=coverage.out
	@echo ""
	@echo "HTML report generated: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

# View coverage in browser
coverage-html: coverage
	@echo "Opening coverage report in browser..."
	@which xdg-open > /dev/null && xdg-open coverage.html || open coverage.html || echo "Please open coverage.html manually"

# Generate detailed coverage report with script
coverage-report:
	@./scripts/coverage.sh

# Run integration tests (NOT YET AVAILABLE - server not implemented)
test-integration:
	@echo "❌ Integration tests require HTTP server implementation"
	@echo "   Server (pkg/server) is not yet implemented"
	@exit 1

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./pkg/database ./pkg/index

# Run all benchmarks with detailed output
bench-all:
	@echo "Running all benchmarks..."
	@go test -bench=. -benchmem -benchtime=3s ./pkg/...

# Run specific benchmark
bench-insert:
	@go test -bench=BenchmarkInsert -benchmem ./pkg/database

bench-find:
	@go test -bench=BenchmarkFind -benchmem ./pkg/database

bench-index:
	@go test -bench=. -benchmem ./pkg/index

# Create baseline benchmark for performance tracking
bench-baseline:
	@./scripts/benchmark.sh baseline

# Run benchmarks and compare with baseline
bench-check:
	@./scripts/benchmark.sh check

# Compare two benchmark results
bench-compare:
	@echo "Usage: make bench-compare OLD=<file> NEW=<file>"
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Error: Both OLD and NEW parameters required"; \
		echo "Example: make bench-compare OLD=benchmarks/old.txt NEW=benchmarks/new.txt"; \
		exit 1; \
	fi
	@./scripts/benchmark.sh compare $(OLD) $(NEW)

# Memory leak detection tests
memory-leak:
	@echo "Running memory leak detection tests..."
	@./scripts/memory-profile.sh test

# Generate memory profile for a package
memory-profile:
	@echo "Usage: make memory-profile PKG=<package>"
	@if [ -z "$(PKG)" ]; then \
		echo "Error: PKG parameter required"; \
		echo "Example: make memory-profile PKG=./pkg/database"; \
		exit 1; \
	fi
	@./scripts/memory-profile.sh profile $(PKG)

# Generate heap profile for a package
memory-heap:
	@echo "Usage: make memory-heap PKG=<package>"
	@if [ -z "$(PKG)" ]; then \
		echo "Error: PKG parameter required"; \
		echo "Example: make memory-heap PKG=./pkg/storage"; \
		exit 1; \
	fi
	@./scripts/memory-profile.sh heap $(PKG)

# Run all memory checks (tests + benchmarks)
memory-check:
	@./scripts/memory-profile.sh check

# Analyze a memory profile
memory-analyze:
	@echo "Usage: make memory-analyze PROFILE=<file>"
	@if [ -z "$(PROFILE)" ]; then \
		echo "Error: PROFILE parameter required"; \
		echo "Example: make memory-analyze PROFILE=profiles/mem.prof"; \
		exit 1; \
	fi
	@./scripts/memory-profile.sh analyze $(PROFILE)

# Clean memory profiles
memory-clean:
	@./scripts/memory-profile.sh clean

# Docker targets

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t laura-db:latest .
	@echo "✓ Docker image built: laura-db:latest"

# Run Docker container
docker-run:
	@echo "Running LauraDB in Docker..."
	@docker run -d \
		--name laura-db \
		-p 8080:8080 \
		-v laura-data:/data \
		laura-db:latest
	@echo "✓ LauraDB running at http://localhost:8080"
	@echo "  Admin console: http://localhost:8080/"
	@echo "  Health check: http://localhost:8080/_health"

# Stop Docker container
docker-stop:
	@echo "Stopping LauraDB container..."
	@docker stop laura-db || true
	@docker rm laura-db || true
	@echo "✓ Container stopped and removed"

# Clean Docker images and volumes
docker-clean: docker-stop
	@echo "Cleaning Docker resources..."
	@docker rmi laura-db:latest || true
	@docker volume rm laura-data || true
	@echo "✓ Docker resources cleaned"

# Build and run in one command
docker: docker-build docker-run

# Docker Compose targets

# Start services with Docker Compose (development)
compose-up:
	@echo "Starting LauraDB with Docker Compose..."
	@docker-compose up -d
	@echo "✓ LauraDB running at http://localhost:8080"
	@echo "  Admin console: http://localhost:8080/"
	@echo "  Health check: http://localhost:8080/_health"

# Start services with monitoring
compose-up-monitoring:
	@echo "Starting LauraDB with monitoring stack..."
	@docker-compose --profile monitoring up -d
	@echo "✓ LauraDB running at http://localhost:8080"
	@echo "  Admin console: http://localhost:8080/"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Grafana: http://localhost:3000 (admin/admin)"

# Start services in production mode
compose-up-prod:
	@echo "Starting LauraDB in production mode..."
	@docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo "✓ LauraDB running in production mode"

# Stop Docker Compose services
compose-down:
	@echo "Stopping Docker Compose services..."
	@docker-compose --profile monitoring down
	@echo "✓ Services stopped"

# Stop and remove volumes
compose-down-volumes:
	@echo "Stopping services and removing volumes..."
	@docker-compose --profile monitoring down -v
	@echo "✓ Services stopped and volumes removed"

# View logs
compose-logs:
	@docker-compose logs -f laura-db

# View all logs (including monitoring)
compose-logs-all:
	@docker-compose logs -f

# Restart services
compose-restart:
	@echo "Restarting Docker Compose services..."
	@docker-compose restart
	@echo "✓ Services restarted"

# Build and start services
compose: compose-up

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f laura
	@rm -rf bin/
	@rm -rf data/ test_data* laura_data/
	@rm -f coverage.out coverage.html
	@rm -f test-results.log handler-test-results.log
	@rm -rf benchmarks/
	@echo "✓ Cleaned"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run linter (static analysis)
lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "❌ golangci-lint not found. Install it with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "   Or visit: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi
	@golangci-lint run ./...

# Fix linter issues automatically
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "❌ golangci-lint not found. Install it with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	@golangci-lint run --fix ./...

# Run server (NOT YET AVAILABLE - server not implemented)
run:
	@echo "❌ HTTP server is not yet implemented"
	@echo "   See TODO.md for implementation status"
	@echo ""
	@echo "Available modes:"
	@echo "  - Embedded Mode: import github.com/mnohosten/laura-db/pkg/database"
	@echo "  - CLI Mode: make cli && ./bin/laura-cli"
	@exit 1

# Build for production (NOT YET AVAILABLE - server not implemented)
build-prod:
	@echo "❌ HTTP server is not yet implemented"
	@echo "   Production build not available"
	@exit 1

# Build for all platforms (NOT YET AVAILABLE - server not implemented)
build-all:
	@echo "❌ HTTP server is not yet implemented"
	@echo "   Multi-platform build not available"
	@exit 1

# Help
help:
	@echo "LauraDB Build System"
	@echo ""
	@echo "Available Targets:"
	@echo "  make              Build everything (CLI + repair + examples)"
	@echo "  make build        Build CLI, repair tool, and examples"
	@echo "  make server       Build HTTP server ✅"
	@echo "  make cli          Build CLI tool only ✅"
	@echo "  make repair       Build repair tool only ✅"
	@echo "  make examples     Build examples only ✅"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build Build Docker image ✅"
	@echo "  make docker-run   Run Docker container ✅"
	@echo "  make docker-stop  Stop Docker container ✅"
	@echo "  make docker-clean Remove Docker image and volumes ✅"
	@echo "  make docker       Build and run in one command ✅"
	@echo ""
	@echo "Docker Compose:"
	@echo "  make compose-up            Start services (development) ✅"
	@echo "  make compose-up-monitoring Start with Prometheus + Grafana ✅"
	@echo "  make compose-up-prod       Start in production mode ✅"
	@echo "  make compose-down          Stop all services ✅"
	@echo "  make compose-down-volumes  Stop and remove volumes ✅"
	@echo "  make compose-logs          View LauraDB logs ✅"
	@echo "  make compose-logs-all      View all service logs ✅"
	@echo "  make compose-restart       Restart all services ✅"
	@echo "  make compose               Alias for compose-up ✅"
	@echo ""
	@echo "Testing:"
	@echo "  make test         Run tests ✅"
	@echo "  make test-coverage Run tests with coverage summary ✅"
	@echo "  make coverage     Generate detailed coverage report ✅"
	@echo "  make coverage-html Generate and open HTML coverage report ✅"
	@echo "  make coverage-report Generate detailed coverage report with badges ✅"
	@echo "  make bench          Run performance benchmarks ✅"
	@echo "  make bench-all      Run all benchmarks with detailed output ✅"
	@echo "  make bench-insert   Run insert benchmarks ✅"
	@echo "  make bench-find     Run find benchmarks ✅"
	@echo "  make bench-index    Run index benchmarks ✅"
	@echo "  make bench-baseline Create baseline for performance tracking ✅"
	@echo "  make bench-check    Compare current performance with baseline ✅"
	@echo "  make bench-compare  Compare two benchmark files (OLD=... NEW=...) ✅"
	@echo ""
	@echo "Memory Profiling:"
	@echo "  make memory-leak    Run memory leak detection tests ✅"
	@echo "  make memory-profile Generate memory profile (PKG=...) ✅"
	@echo "  make memory-heap    Generate heap profile (PKG=...) ✅"
	@echo "  make memory-check   Run all memory checks ✅"
	@echo "  make memory-analyze Analyze memory profile (PROFILE=...) ✅"
	@echo "  make memory-clean   Clean memory profile files ✅"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint         Run static analysis with golangci-lint ✅"
	@echo "  make lint-fix     Run linter and auto-fix issues ✅"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean        Remove build artifacts ✅"
	@echo "  make deps         Install dependencies ✅"
	@echo "  make help         Show this help ✅"
	@echo ""
	@echo "Not Yet Implemented (HTTP server required):"
	@echo "  make server       Build main server ❌"
	@echo "  make run          Build and run server ❌"
	@echo "  make build-prod   Build optimized binary ❌"
	@echo "  make build-all    Build for all platforms ❌"
	@echo "  make test-integration Run integration tests ❌"
	@echo ""
	@echo "See TODO.md for implementation status"
