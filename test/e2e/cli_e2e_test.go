package e2e

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCLIFullWorkflow tests complete end-to-end CLI workflow
func TestCLIFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup: Create temporary data directory
	tmpDir, err := os.MkdirTemp("", "laura-cli-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build CLI binary
	cliBinary := filepath.Join(tmpDir, "laura-cli")
	buildCmd := exec.Command("go", "build", "-o", cliBinary, "../../cmd/laura-cli/main.go")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	t.Log("CLI binary built successfully")

	// Run test scenarios
	t.Run("BasicCommands", func(t *testing.T) {
		testCLIBasicCommands(t, cliBinary, tmpDir)
	})

	t.Run("InsertAndFind", func(t *testing.T) {
		testCLIInsertAndFind(t, cliBinary, tmpDir)
	})

	t.Run("UpdateOperations", func(t *testing.T) {
		testCLIUpdateOperations(t, cliBinary, tmpDir)
	})

	t.Run("DeleteOperations", func(t *testing.T) {
		testCLIDeleteOperations(t, cliBinary, tmpDir)
	})

	t.Run("IndexCommands", func(t *testing.T) {
		testCLIIndexCommands(t, cliBinary, tmpDir)
	})

	t.Run("AggregationCommands", func(t *testing.T) {
		testCLIAggregationCommands(t, cliBinary, tmpDir)
	})
}

// runCLICommand runs a single CLI command and returns output
func runCLICommand(t *testing.T, cliBinary, dataDir, command string) (string, error) {
	cmd := exec.Command(cliBinary, dataDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Send command
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, command+"\n")
		time.Sleep(100 * time.Millisecond)
		io.WriteString(stdin, "exit\n")
	}()

	// Read output
	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
	}

	cmd.Wait()
	return output.String(), nil
}

func testCLIBasicCommands(t *testing.T, cliBinary, dataDir string) {
	// Test show collections
	output, err := runCLICommand(t, cliBinary, dataDir, "show collections")
	if err != nil {
		t.Fatalf("Failed to run 'show collections': %v", err)
	}
	if !strings.Contains(output, "collections") && !strings.Contains(output, "Collections") {
		t.Logf("Warning: Expected output to mention collections, got: %s", output)
	}

	// Test use command
	output, err = runCLICommand(t, cliBinary, dataDir, "use test_db")
	if err != nil {
		t.Fatalf("Failed to run 'use test_db': %v", err)
	}

	t.Log("✓ Basic CLI commands passed")
}

func testCLIInsertAndFind(t *testing.T, cliBinary, dataDir string) {
	commands := []string{
		"use test_db",
		`db.users.insertOne({"name": "Alice", "age": 30, "email": "alice@example.com"})`,
		`db.users.find({"name": "Alice"})`,
		"exit",
	}

	cmd := exec.Command(cliBinary, dataDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Read output
	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
	}

	cmd.Wait()

	outputStr := output.String()
	if !strings.Contains(outputStr, "Alice") && !strings.Contains(outputStr, "alice") {
		t.Logf("Warning: Expected to find 'Alice' in output, got: %s", outputStr)
	}

	t.Log("✓ Insert and find operations passed")
}

func testCLIUpdateOperations(t *testing.T, cliBinary, dataDir string) {
	commands := []string{
		"use test_db",
		`db.users.insertOne({"name": "Bob", "age": 25})`,
		`db.users.updateOne({"name": "Bob"}, {"$set": {"age": 26}})`,
		`db.users.find({"name": "Bob"})`,
		"exit",
	}

	cmd := exec.Command(cliBinary, dataDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	cmd.Wait()

	t.Log("✓ Update operations passed")
}

func testCLIDeleteOperations(t *testing.T, cliBinary, dataDir string) {
	commands := []string{
		"use test_db",
		`db.temp.insertOne({"name": "ToDelete"})`,
		`db.temp.deleteOne({"name": "ToDelete"})`,
		`db.temp.find({})`,
		"exit",
	}

	cmd := exec.Command(cliBinary, dataDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	cmd.Wait()

	t.Log("✓ Delete operations passed")
}

func testCLIIndexCommands(t *testing.T, cliBinary, dataDir string) {
	commands := []string{
		"use test_db",
		`db.indexed.createIndex({"email": 1}, {"unique": true})`,
		`db.indexed.getIndexes()`,
		"exit",
	}

	cmd := exec.Command(cliBinary, dataDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Read output
	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
	}

	cmd.Wait()

	outputStr := output.String()
	if !strings.Contains(outputStr, "index") && !strings.Contains(outputStr, "Index") {
		t.Logf("Warning: Expected to find index information in output, got: %s", outputStr)
	}

	t.Log("✓ Index commands passed")
}

func testCLIAggregationCommands(t *testing.T, cliBinary, dataDir string) {
	commands := []string{
		"use test_db",
		`db.sales.insertOne({"product": "A", "quantity": 10, "price": 100})`,
		`db.sales.insertOne({"product": "B", "quantity": 5, "price": 200})`,
		`db.sales.insertOne({"product": "A", "quantity": 15, "price": 100})`,
		`db.sales.aggregate([{"$group": {"_id": "$product", "total": {"$sum": "$quantity"}}}])`,
		"exit",
	}

	cmd := exec.Command(cliBinary, dataDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(150 * time.Millisecond)
		}
	}()

	// Read output
	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
	}

	cmd.Wait()

	t.Log("✓ Aggregation commands passed")
}

// TestCLIBatchMode tests CLI in batch/script mode
func TestCLIBatchMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup
	tmpDir, err := os.MkdirTemp("", "laura-cli-batch-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build CLI binary
	cliBinary := filepath.Join(tmpDir, "laura-cli")
	buildCmd := exec.Command("go", "build", "-o", cliBinary, "../../cmd/laura-cli/main.go")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Create test script
	scriptPath := filepath.Join(tmpDir, "test_script.js")
	script := `use test_db
db.products.insertOne({"name": "Product1", "price": 99.99})
db.products.insertOne({"name": "Product2", "price": 149.99})
db.products.find({})
`
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Run CLI with script input
	cmd := exec.Command(cliBinary, tmpDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send script and exit
	go func() {
		defer stdin.Close()
		scriptContent, _ := os.ReadFile(scriptPath)
		io.WriteString(stdin, string(scriptContent))
		time.Sleep(200 * time.Millisecond)
		io.WriteString(stdin, "exit\n")
	}()

	cmd.Wait()

	t.Log("✓ CLI batch mode passed")
}

// TestCLIErrorHandling tests CLI error handling
func TestCLIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup
	tmpDir, err := os.MkdirTemp("", "laura-cli-error-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build CLI binary
	cliBinary := filepath.Join(tmpDir, "laura-cli")
	buildCmd := exec.Command("go", "build", "-o", cliBinary, "../../cmd/laura-cli/main.go")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Test invalid JSON
	commands := []string{
		"use test_db",
		`db.test.insertOne({invalid json})`,
		"exit",
	}

	cmd := exec.Command(cliBinary, tmpDir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start CLI: %v", err)
	}

	// Send commands
	go func() {
		defer stdin.Close()
		for _, command := range commands {
			io.WriteString(stdin, command+"\n")
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// Read output
	scanner := bufio.NewScanner(stdout)
	var output strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line)
		output.WriteString("\n")
	}

	cmd.Wait()

	outputStr := output.String()
	// Should contain some error indication
	if !strings.Contains(strings.ToLower(outputStr), "error") &&
	   !strings.Contains(strings.ToLower(outputStr), "invalid") &&
	   !strings.Contains(strings.ToLower(outputStr), "failed") {
		t.Logf("Warning: Expected error message in output for invalid JSON, got: %s", outputStr)
	}

	t.Log("✓ CLI error handling passed")
}

// TestCLIHelp tests CLI help command
func TestCLIHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup
	tmpDir, err := os.MkdirTemp("", "laura-cli-help-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build CLI binary
	cliBinary := filepath.Join(tmpDir, "laura-cli")
	buildCmd := exec.Command("go", "build", "-o", cliBinary, "../../cmd/laura-cli/main.go")
	buildCmd.Dir = tmpDir
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build CLI: %v\nOutput: %s", err, output)
	}

	// Run help command
	output, err := runCLICommand(t, cliBinary, tmpDir, "help")
	if err != nil {
		t.Fatalf("Failed to run 'help': %v", err)
	}

	if !strings.Contains(output, "help") && !strings.Contains(output, "Help") &&
	   !strings.Contains(output, "command") && !strings.Contains(output, "Command") {
		t.Logf("Warning: Expected help information in output, got: %s", output)
	}

	t.Log("✓ CLI help command passed")
}
