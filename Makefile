.PHONY: all build clean test server examples help

# Default target
all: build

# Build everything
build: server examples

# Build main server
server:
	@echo "Building LauraDB server..."
	@go build -o laura ./cmd/server/main.go
	@echo "✓ Built: laura"

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

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v ./pkg/server -run TestServerIntegration

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

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f laura
	@rm -rf bin/
	@rm -rf data/ test_data* laura_data/
	@rm -f coverage.out coverage.html
	@rm -f test-results.log handler-test-results.log
	@echo "✓ Cleaned"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Run server
run: server
	@echo "Starting LauraDB server..."
	@./laura -port 8080 -data-dir ./data

# Build for production
build-prod:
	@echo "Building for production..."
	@go build -ldflags="-s -w" -o laura ./cmd/server/main.go
	@echo "✓ Built: laura (optimized)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@GOOS=linux GOARCH=amd64 go build -o laura-linux-amd64 ./cmd/server/main.go
	@GOOS=darwin GOARCH=amd64 go build -o laura-darwin-amd64 ./cmd/server/main.go
	@GOOS=windows GOARCH=amd64 go build -o laura-windows-amd64.exe ./cmd/server/main.go
	@echo "✓ Built for Linux, macOS, and Windows"

# Help
help:
	@echo "LauraDB Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build everything"
	@echo "  make build        Build server and examples"
	@echo "  make server       Build main server only"
	@echo "  make examples     Build examples only"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage summary"
	@echo "  make coverage     Generate detailed coverage report"
	@echo "  make coverage-html Generate and open HTML coverage report"
	@echo "  make test-integration Run integration tests"
	@echo "  make bench        Run performance benchmarks"
	@echo "  make bench-all    Run all benchmarks with detailed output"
	@echo "  make bench-insert Run insert benchmarks"
	@echo "  make bench-find   Run find benchmarks"
	@echo "  make bench-index  Run index benchmarks"
	@echo "  make clean        Remove build artifacts"
	@echo "  make deps         Install dependencies"
	@echo "  make run          Build and run server"
	@echo "  make build-prod   Build optimized binary"
	@echo "  make build-all    Build for all platforms"
	@echo "  make help         Show this help"
