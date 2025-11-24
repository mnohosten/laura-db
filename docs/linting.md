# Static Analysis & Linting

LauraDB uses [golangci-lint](https://golangci-lint.run/) for static code analysis and quality checks.

## Installation

Install golangci-lint using one of these methods:

### Using Go

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Using Homebrew (macOS)

```bash
brew install golangci-lint
```

### Using Binary

See [official installation guide](https://golangci-lint.run/usage/install/) for other platforms.

## Usage

### Run Linter

```bash
make lint
```

This will run all enabled linters on the entire codebase.

### Auto-fix Issues

Many linter issues can be automatically fixed:

```bash
make lint-fix
```

This will attempt to fix issues like:
- Import formatting (goimports)
- Code formatting (gofmt)
- Unnecessary type conversions (unconvert)
- Ineffectual assignments (ineffassign)

### Run Specific Linters

```bash
golangci-lint run --disable-all -E errcheck ./...
golangci-lint run --disable-all -E staticcheck ./...
```

### Check Only Changed Files

```bash
golangci-lint run --new-from-rev=HEAD~1
```

## Enabled Linters

### Error Detection
- **errcheck**: Checks for unchecked errors
- **govet**: Examines code for suspicious constructs
- **staticcheck**: Advanced static analysis
- **typecheck**: Verifies type correctness

### Code Quality
- **gofmt**: Ensures consistent formatting
- **goimports**: Manages import organization
- **ineffassign**: Detects ineffectual assignments
- **misspell**: Finds common spelling errors
- **unconvert**: Removes unnecessary type conversions
- **unparam**: Reports unused function parameters
- **unused**: Finds unused code

### Style
- **gocyclo**: Detects high cyclomatic complexity (threshold: 15)
- **goconst**: Finds repeated strings that could be constants
- **godot**: Ensures comments end with periods
- **gosimple**: Suggests code simplifications

### Performance
- **prealloc**: Identifies slice declarations that could be preallocated

### Best Practices
- **revive**: Configurable linter for Go
- **exportloopref**: Checks for loop variable capture issues
- **nilerr**: Finds incorrect nil error returns

## Configuration

The linter is configured in `.golangci.yml`. Key settings:

- **Timeout**: 5 minutes
- **Complexity threshold**: 15 (gocyclo)
- **Test files**: Excluded from some linters (gocyclo, gosec, goconst)
- **Format**: Colored line numbers with linter names

## Common Issues

### 1. Unchecked Errors (errcheck)

**Bad:**
```go
file.Close()
```

**Good:**
```go
if err := file.Close(); err != nil {
    return fmt.Errorf("failed to close file: %w", err)
}
```

**Acceptable in some cases:**
```go
_ = file.Close()  // Explicitly ignoring error
```

### 2. Ineffectual Assignments (ineffassign)

**Bad:**
```go
x := 5
x = 10  // First assignment never used
```

**Good:**
```go
x := 10
```

### 3. Cyclomatic Complexity (gocyclo)

**Bad:**
```go
func processData(data []byte) error {
    if len(data) == 0 {
        if debug {
            if verbose {
                // ... deeply nested logic
            }
        }
    }
    // Function too complex (>15)
}
```

**Good:**
```go
func processData(data []byte) error {
    if len(data) == 0 {
        return handleEmptyData()  // Extract to separate function
    }
    // ...
}

func handleEmptyData() error {
    if debug && verbose {
        // Simplified logic
    }
    return nil
}
```

### 4. Unused Variables/Functions (unused)

**Bad:**
```go
func helper() string {  // Never called
    return "unused"
}
```

**Good:**
Remove unused code or export it if needed elsewhere.

### 5. Shadow Variables (govet)

**Bad:**
```go
data := []byte("test")
if true {
    data := []byte("shadow")  // Shadows outer variable
    process(data)
}
```

**Good:**
```go
data := []byte("test")
if true {
    newData := []byte("different")
    process(newData)
}
```

## CI Integration

To integrate linting into CI/CD:

### GitHub Actions

```yaml
name: Lint
on: [push, pull_request]
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: latest
```

### GitLab CI

```yaml
lint:
  stage: test
  image: golangci/golangci-lint:latest
  script:
    - golangci-lint run ./...
```

## Skipping Linters

Sometimes you need to skip linter checks for specific lines:

```go
//nolint:errcheck  // Skip errcheck for this line
file.Close()

//nolint  // Skip all linters for this line
complexLegacyCode()
```

Use sparingly and only when necessary!

## Performance

Typical linting times:
- Full codebase: ~10-30 seconds
- Only changed files: ~2-5 seconds

For faster development, use:
```bash
golangci-lint run --fast ./...
```

This runs only fast linters, ideal for pre-commit hooks.

## Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
make lint
if [ $? -ne 0 ]; then
    echo "❌ Linting failed. Fix issues before committing."
    exit 1
fi
```

Make it executable:
```bash
chmod +x .git/hooks/pre-commit
```

## References

- [golangci-lint Documentation](https://golangci-lint.run/)
- [Enabled Linters](https://golangci-lint.run/usage/linters/)
- [Configuration Guide](https://golangci-lint.run/usage/configuration/)
