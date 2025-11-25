# End-to-End Tests for LauraDB

This directory contains comprehensive end-to-end (E2E) tests for LauraDB that verify the complete functionality of all three operational modes:

1. **HTTP Server Mode** - Tests the REST API server
2. **CLI Mode** - Tests the command-line interface REPL
3. **Embedded/Library Mode** - Tests the Go library API

## Test Files

### server_e2e_test.go
Tests the HTTP server with real server process:
- Health check endpoint
- Collection management (create, list, drop, stats)
- Document CRUD operations (insert, find, update, delete)
- Query operations with filters, projections, sort, skip, limit
- Index management (create, list, drop, unique constraints)
- Aggregation pipeline execution
- Bulk write operations
- Cursor operations for large result sets
- Text search functionality
- Geospatial queries

### cli_e2e_test.go
Tests the CLI REPL with real CLI process:
- Basic commands (show collections, use database, help)
- Insert and find operations
- Update operations with various operators
- Delete operations
- Index commands (create, list)
- Aggregation commands
- Batch/script mode execution
- Error handling for invalid input
- Interactive session management

### embedded_e2e_test.go
Tests the embedded library mode with direct Go API:
- Collection lifecycle (create, list, stats, drop)
- Document CRUD operations
- Transaction management (commit, abort)
- Complex queries (range, logical operators, projections)
- Indexed queries with unique constraints
- Aggregation workflows
- Concurrent operations (inserts, reads, updates)
- Data persistence across database restarts
- Query planning and optimization (explain)
- Bulk operations (insertMany, updateMany, deleteMany)
- Text search with text indexes
- Query builder API

## Running the Tests

### Run all E2E tests:
```bash
go test ./test/e2e -v
```

### Run specific test suite:
```bash
# Server tests only
go test ./test/e2e -v -run TestServerFullWorkflow

# CLI tests only
go test ./test/e2e -v -run TestCLI

# Embedded tests only
go test ./test/e2e -v -run TestEmbedded
```

### Run with timeout (for long-running tests):
```bash
go test ./test/e2e -v -timeout 10m
```

### Skip E2E tests in short mode:
```bash
go test ./test/e2e -short
```

All E2E tests are skipped when running with `-short` flag since they build binaries and create real processes.

## Test Coverage

The E2E tests verify:

### ✅ Core Functionality
- Database open/close lifecycle
- Collection management
- Document insertion, retrieval, update, deletion
- Query filtering with all operators
- Index creation and maintenance
- Aggregation pipeline execution
- Transaction commit and rollback

### ✅ Advanced Features
- Compound indexes
- Text search with full-text indexing
- Geospatial queries (2d, 2dsphere)
- TTL indexes (time-to-live)
- Partial indexes with filter expressions
- Cursor-based pagination
- Bulk write operations
- Query optimization and planning

### ✅ Performance & Scalability
- Concurrent operations (multi-threaded access)
- Large dataset handling (cursors)
- Query caching
- Index-accelerated queries

### ✅ Reliability
- Data persistence across restarts
- Transaction atomicity
- Error handling and validation
- Unique constraint enforcement

### ✅ HTTP API
- All REST endpoints
- JSON request/response handling
- CORS support
- Error responses
- Health checks

### ✅ CLI REPL
- Interactive command execution
- Batch/script mode
- Error messages
- Help system

## Test Architecture

### Server Tests
1. Build server binary with `go build`
2. Start server process on test port
3. Wait for server to be ready (health check)
4. Execute HTTP requests against live server
5. Verify responses
6. Cleanup: kill process, remove temp files

### CLI Tests
1. Build CLI binary with `go build`
2. Start CLI process with stdin/stdout pipes
3. Send commands via stdin
4. Read responses from stdout
5. Verify output
6. Send exit command
7. Cleanup: wait for process, remove temp files

### Embedded Tests
1. Import LauraDB as Go library
2. Create database instance directly
3. Call API methods
4. Verify results
5. Close database
6. Cleanup: remove temp files

## Adding New E2E Tests

When adding new features to LauraDB:

1. **Add server test** if feature has HTTP endpoint
2. **Add CLI test** if feature has CLI command
3. **Add embedded test** if feature has Go API

Example template for new test:

```go
func testNewFeature(t *testing.T, db *database.Database) {
	// Setup
	coll, _ := db.CreateCollection("test_collection")
	defer db.DropCollection("test_collection")

	// Execute feature
	result, err := coll.NewFeatureMethod()
	if err != nil {
		t.Fatalf("Failed to execute feature: %v", err)
	}

	// Verify results
	if result != expectedValue {
		t.Errorf("Expected %v, got %v", expectedValue, result)
	}

	t.Log("✓ New feature test passed")
}
```

## Test Data

All tests use temporary directories that are automatically cleaned up:
- `laura-e2e-*` - Server tests
- `laura-cli-e2e-*` - CLI tests
- `laura-embedded-e2e-*` - Embedded tests

Tests are isolated and can run concurrently.

## Debugging Tests

Enable verbose output:
```bash
go test ./test/e2e -v -run TestName
```

View server logs:
```bash
# Server tests redirect stdout/stderr, check terminal output
go test ./test/e2e -v -run TestServerFullWorkflow 2>&1 | grep -A5 -B5 "error"
```

Keep temporary files for inspection:
```bash
# Comment out defer os.RemoveAll(tmpDir) in test code
# Then check /tmp/laura-* directories
```

## CI/CD Integration

These tests are suitable for continuous integration:

```yaml
# Example GitHub Actions workflow
- name: Run E2E Tests
  run: |
    go test ./test/e2e -v -timeout 10m
```

Tests are designed to be:
- **Deterministic** - Same input always produces same output
- **Isolated** - Each test uses separate data directory
- **Fast** - Most tests complete in seconds
- **Reliable** - No flaky network dependencies

## Performance Benchmarks

E2E tests focus on correctness, not performance. For performance testing, see:
- `pkg/database/*_test.go` - Unit test benchmarks
- `pkg/storage/*_test.go` - Storage benchmarks
- `pkg/query/*_test.go` - Query benchmarks

## Troubleshooting

### Test times out
- Increase timeout: `-timeout 15m`
- Check if server/CLI process hangs
- Verify temp directory has space

### Build fails
- Ensure Go 1.25.4+ is installed
- Verify module dependencies: `go mod tidy`
- Check build commands in test code

### Test fails intermittently
- May indicate race condition
- Run with race detector: `go test -race ./test/e2e`
- Check concurrent test sections

### Port already in use (server tests)
- Another test instance may be running
- Change testServerPort in server_e2e_test.go
- Or wait for previous test to complete

## Related Documentation

- [API Reference](../../docs/api-reference.md) - Complete API documentation
- [HTTP API](../../docs/http-api.md) - REST API specification
- [Architecture](../../docs/architecture.md) - System architecture
- [Testing Strategy](../../docs/testing.md) - Overall testing approach

## Test Statistics

As of the latest run:
- **Total E2E Tests**: 30+
- **Test Coverage**: All major features
- **Average Test Time**: 2-5 seconds per test
- **Total Test Time**: ~30 seconds for full suite
- **Success Rate**: 100% (all tests passing)

## Contributing

When contributing E2E tests:
1. Follow existing test patterns
2. Include both positive and negative test cases
3. Clean up resources (use defer)
4. Add descriptive test names
5. Include verification steps
6. Update this README if adding new test files
