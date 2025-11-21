# LauraDB Testing Guide

## Test Coverage Summary

**Overall Coverage**: 64.3% of statements

LauraDB has a comprehensive test suite with 80+ test cases covering all major components.

## Coverage by Package

| Package | Coverage | Status | Test Files |
|---------|----------|--------|------------|
| `pkg/aggregation` | 72.9% | ✅ Good | pipeline_test.go |
| `pkg/database` | 77.7% | ✅ Excellent | database_test.go, array_operators_test.go, numeric_operators_test.go |
| `pkg/document` | 69.5% | ✅ Good | document_test.go, bson_test.go, objectid_test.go |
| `pkg/index` | 73.0% | ✅ Good | btree_test.go, index_test.go |
| `pkg/mvcc` | 68.1% | ✅ Good | mvcc_test.go |
| `pkg/query` | 55.4% | ⚠️ Moderate | query_test.go, planner_test.go |
| `pkg/server` | 92.4% | 🎉 Excellent | server_integration_test.go |
| `pkg/server/handlers` | 70.2% | ✅ Good | All handler tests |
| `pkg/storage` | 66.1% | ✅ Good | storage_test.go |

## Running Tests

### Basic Test Commands

```bash
# Run all tests
make test

# Run tests with coverage summary
make test-coverage

# Generate detailed coverage report
make coverage

# Generate and open HTML coverage report
make coverage-html

# Run integration tests only
make test-integration

# Run specific package tests
go test ./pkg/database
go test ./pkg/query

# Run with verbose output
go test -v ./pkg/...

# Run specific test
go test ./pkg/database -run TestInsertOne
```

### Test Organization

```
pkg/
├── aggregation/
│   └── pipeline_test.go        # Aggregation pipeline tests
├── database/
│   ├── database_test.go        # Core CRUD operations (17 tests)
│   ├── array_operators_test.go # Array update operators (4 tests)
│   └── numeric_operators_test.go # Numeric operators (4 tests)
├── document/
│   ├── document_test.go        # Document operations (7 tests)
│   ├── bson_test.go           # BSON encoding (4 tests)
│   └── objectid_test.go       # ObjectID generation (7 tests)
├── index/
│   └── btree_test.go          # B+ tree and indexing (9 tests)
├── mvcc/
│   └── mvcc_test.go           # Transaction tests (6 tests)
├── query/
│   ├── query_test.go          # Query operations (14 tests)
│   └── planner_test.go        # Query planner (5 tests)
├── server/
│   ├── server_integration_test.go  # Full integration tests (5 tests)
│   └── handlers/
│       ├── admin_test.go       # Health & stats (4 tests)
│       ├── aggregate_test.go   # Aggregation API (6 tests)
│       ├── collection_test.go  # Collection API (10 tests)
│       ├── document_test.go    # Document API (5 tests)
│       └── query_test.go       # Query API (7 tests)
└── storage/
    └── storage_test.go        # Storage engine (5 tests)
```

## Test Categories

### Unit Tests (70+ tests)

Test individual components in isolation:
- Document encoding/decoding
- B+ tree operations
- Query matching
- Update operators
- MVCC transactions
- Storage operations

### Integration Tests (10+ tests)

Test components working together:
- HTTP API endpoints
- Query planner with indexes
- Aggregation pipeline
- Full CRUD workflows

### Coverage Highlights

**Best Coverage (>70%)**:
- ✅ **Server Integration** (92.4%) - Excellent HTTP API coverage
- ✅ **Database Operations** (77.7%) - Comprehensive CRUD testing
- ✅ **Aggregation Pipeline** (72.9%) - Good pipeline coverage
- ✅ **Indexing** (73.0%) - B+ tree well tested

**Areas for Improvement (<70%)**:
- ⚠️ **Query Engine** (55.4%) - Needs more operator tests
- ⚠️ **Storage** (66.1%) - WAL and recovery need more tests
- ⚠️ **MVCC** (68.1%) - Edge cases in concurrency

## Test Fixtures and Utilities

### Test Database Setup

```go
// pkg/server/testutil/testutil.go provides:

// Create temporary test database
db, cleanup := testutil.SetupTestDB(t)
defer cleanup()

// Make HTTP requests
req, err := testutil.MakeJSONRequest("POST", "/users/_doc", doc)

// Execute requests
rr := testutil.ExecuteRequest(router, req)

// Assert responses
testutil.AssertStatusCode(t, rr, http.StatusOK)
response := testutil.AssertNoError(t, rr)
```

### Common Test Patterns

**Database Tests**:
```go
func TestSomeOperation(t *testing.T) {
    // Setup
    testDir := "./test_data"
    defer os.RemoveAll(testDir)

    config := database.DefaultConfig(testDir)
    db, err := database.Open(config)
    if err != nil {
        t.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()

    // Test logic
    coll := db.Collection("test")
    _, err = coll.InsertOne(map[string]interface{}{"name": "Alice"})

    // Assertions
    if err != nil {
        t.Errorf("Insert failed: %v", err)
    }
}
```

**HTTP Handler Tests**:
```go
func TestHandler(t *testing.T) {
    db, router, cleanup := setupTest(t)
    defer cleanup()

    req, _ := testutil.MakeJSONRequest("POST", "/users/_doc", doc)
    rr := testutil.ExecuteRequest(router, req)

    testutil.AssertStatusCode(t, rr, http.StatusCreated)
    response := testutil.AssertNoError(t, rr)
}
```

## Recent Test Fixes

### Fixed Issues (All tests now passing!)

1. **Handler Test EOF Errors** ✅
   - **Problem**: Response body being read twice
   - **Solution**: Modified `AssertNoError` to return parsed response
   - **Impact**: Fixed 19 failing handler tests

2. **Sort Order Issues** ✅
   - **Problem**: Aggregation sort not handling float64 from JSON
   - **Solution**: Added float64 case in sort order parsing
   - **Impact**: Fixed `TestAggregationWithSort`

3. **Query Sort Format** ✅
   - **Problem**: Tests using `"order": "asc"` but code expected boolean
   - **Solution**: Added custom JSON unmarshaling for flexible sort format
   - **Impact**: Fixed `TestSearchWithSort`

## Test Results

### Latest Test Run

```
✅ pkg/aggregation        PASS  0.003s  (8 tests)
✅ pkg/database          PASS  0.134s  (25 tests)
✅ pkg/document          PASS  0.004s  (18 tests)
✅ pkg/index             PASS  0.005s  (9 tests)
✅ pkg/mvcc              PASS  0.005s  (6 tests)
✅ pkg/query             PASS  0.004s  (19 tests)
✅ pkg/server            PASS  0.127s  (1 integration test, 5 subtests)
✅ pkg/server/handlers   PASS  0.023s  (32 tests)
✅ pkg/storage           PASS  0.029s  (5 tests)

Total: ALL TESTS PASSING (100% success rate)
```

## Adding New Tests

### Test File Naming

- `*_test.go` - Unit tests in same package
- `*_integration_test.go` - Integration tests
- Use descriptive test names: `TestFeatureScenario`

### Test Structure

```go
func TestFeatureName(t *testing.T) {
    // Arrange - Setup test data

    // Act - Execute the operation

    // Assert - Verify results
    if got != want {
        t.Errorf("Expected %v, got %v", want, got)
    }
}
```

### Table-Driven Tests

```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name  string
        input interface{}
        want  interface{}
    }{
        {"case1", input1, expected1},
        {"case2", input2, expected2},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := functionUnderTest(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Continuous Integration

### GitHub Actions (Planned)

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - run: make test
      - run: make coverage
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Coverage Goals

### Current Status
- ✅ Overall: 64.3% (Good baseline)
- 🎯 Target: 80% by v0.2.0
- 🎯 Target: 90% by v1.0.0

### Improvement Plan

**Priority 1: Query Engine (55.4% → 70%)**
- Add tests for all comparison operators
- Test error cases and edge conditions
- More complex query combinations

**Priority 2: Storage (66.1% → 75%)**
- WAL replay scenarios
- Recovery edge cases
- Corruption handling

**Priority 3: MVCC (68.1% → 80%)**
- Concurrent transaction conflicts
- Garbage collection edge cases
- Transaction isolation verification

## Performance Testing

See [BENCHMARKS.md](./BENCHMARKS.md) for performance testing guide.

## Debugging Failed Tests

### Common Issues

**1. Temporary Files Not Cleaned**
```bash
# Clean up test directories
rm -rf test_data* laura_data/ data/
```

**2. Port Already in Use (Integration Tests)**
```bash
# Find and kill process using port 8080
lsof -ti:8080 | xargs kill -9
```

**3. Race Conditions**
```bash
# Run with race detector
go test -race ./pkg/...
```

### Verbose Output

```bash
# See detailed test output
go test -v ./pkg/database -run TestSpecific

# See test timing
go test -v -timeout 30s ./pkg/...
```

## Best Practices

1. **Always clean up resources**
   ```go
   defer cleanup()
   defer db.Close()
   defer os.RemoveAll(testDir)
   ```

2. **Use meaningful test names**
   ```go
   TestInsertDocumentWithDuplicateID  // Good
   TestInsert  // Too vague
   ```

3. **Test error paths**
   ```go
   // Test both success and failure cases
   TestInsertValidDocument
   TestInsertInvalidDocument
   ```

4. **Use test helpers**
   ```go
   t.Helper()  // Mark function as test helper
   ```

5. **Parallel tests when possible**
   ```go
   func TestParallel(t *testing.T) {
       t.Parallel()  // Run in parallel with other tests
       // ...
   }
   ```

## Resources

- Go Testing: https://go.dev/doc/tutorial/add-a-test
- Table-Driven Tests: https://go.dev/wiki/TableDrivenTests
- Test Coverage: https://go.dev/blog/cover

---

**Last Updated**: Test fixes completed - All 80+ tests passing
**Test Status**: ✅ ALL PASSING
**Coverage**: 64.3% overall, trending towards 80% goal
