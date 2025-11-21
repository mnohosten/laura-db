package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mnohosten/laura-db/pkg/database"
)

const (
	version = "0.1.0"
	banner  = `
╔══════════════════════════════════════╗
║        LauraDB CLI v%s         ║
║  MongoDB-like Document Database     ║
╚══════════════════════════════════════╝

Type 'help' for available commands
Type 'exit' or 'quit' to exit

`
)

type CLI struct {
	db             *database.Database
	currentDB      string
	currentColl    string
	dataDir        string
	scanner        *bufio.Scanner
	commandHistory []string
}

func NewCLI(dataDir string) (*CLI, error) {
	// Open database
	config := database.DefaultConfig(dataDir)
	db, err := database.Open(config)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &CLI{
		db:             db,
		currentDB:      "default",
		dataDir:        dataDir,
		scanner:        bufio.NewScanner(os.Stdin),
		commandHistory: make([]string, 0),
	}, nil
}

func (c *CLI) Close() error {
	return c.db.Close()
}

func (c *CLI) Run() error {
	fmt.Printf(banner, version)

	for {
		// Display prompt
		prompt := fmt.Sprintf("laura> ")
		if c.currentColl != "" {
			prompt = fmt.Sprintf("laura:%s> ", c.currentColl)
		}
		fmt.Print(prompt)

		// Read input
		if !c.scanner.Scan() {
			break
		}

		line := strings.TrimSpace(c.scanner.Text())
		if line == "" {
			continue
		}

		// Add to history
		c.commandHistory = append(c.commandHistory, line)

		// Execute command
		if err := c.executeCommand(line); err != nil {
			if err.Error() == "exit" {
				fmt.Println("Goodbye!")
				return nil
			}
			fmt.Printf("Error: %v\n", err)
		}
	}

	return c.scanner.Err()
}

func (c *CLI) executeCommand(line string) error {
	// Parse command
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "help", "?":
		return c.showHelp()
	case "exit", "quit":
		return fmt.Errorf("exit")
	case "use":
		return c.useCollection(parts)
	case "show":
		return c.showCommand(parts)
	case "insert", "find", "update", "delete", "count":
		return c.collectionCommand(cmd, line)
	case "createindex", "getindexes", "stats":
		return c.managementCommand(cmd, line)
	case "clear":
		fmt.Print("\033[H\033[2J") // Clear screen
		return nil
	case "version":
		fmt.Printf("LauraDB CLI version %s\n", version)
		return nil
	default:
		// Try to parse as collection.method syntax
		if strings.Contains(line, ".") {
			return c.parseCollectionSyntax(line)
		}
		return fmt.Errorf("unknown command: %s (type 'help' for available commands)", cmd)
	}
}

func (c *CLI) showHelp() error {
	help := `
LauraDB CLI Commands:

Basic Commands:
  help, ?                  Show this help message
  exit, quit               Exit the CLI
  clear                    Clear the screen
  version                  Show CLI version
  use <collection>         Switch to a collection

Collection Operations:
  insert <json>            Insert a document
  find [query]             Find documents (query is optional)
  update <query> <update>  Update documents
  delete <query>           Delete documents
  count [query]            Count documents

Alternative Syntax (MongoDB-like):
  <collection>.find({query})
  <collection>.insert({document})
  <collection>.update({query}, {update})
  <collection>.delete({query})
  <collection>.count()

Index Management:
  createindex <field> [options]    Create an index
  getindexes                       List all indexes
  stats                            Show collection statistics

Information:
  show collections         List all collections

Examples:
  use users
  insert {"name": "Alice", "age": 25}
  find {"age": {"$gte": 21}}
  users.find({"name": "Alice"})
  createindex name {"unique": true}

Note: JSON must be properly formatted with double quotes.
`
	fmt.Println(help)
	return nil
}

func (c *CLI) useCollection(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("usage: use <collection>")
	}
	c.currentColl = parts[1]
	fmt.Printf("Switched to collection '%s'\n", c.currentColl)
	return nil
}

func (c *CLI) showCommand(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("usage: show <collections|...>")
	}

	subCmd := strings.ToLower(parts[1])
	switch subCmd {
	case "collections", "colls":
		// List all collections (this is a simplified version)
		fmt.Println("Collections:")
		fmt.Println("  (LauraDB doesn't track collection names globally)")
		fmt.Println("  Use 'use <name>' to create/access a collection")
		return nil
	default:
		return fmt.Errorf("unknown show command: %s", subCmd)
	}
}

func (c *CLI) collectionCommand(cmd, line string) error {
	if c.currentColl == "" {
		return fmt.Errorf("no collection selected (use 'use <collection>' first)")
	}

	coll := c.db.Collection(c.currentColl)

	// Extract JSON part
	jsonStart := strings.Index(line, "{")
	if jsonStart == -1 && cmd != "find" && cmd != "count" {
		return fmt.Errorf("command requires JSON argument")
	}

	switch cmd {
	case "insert":
		return c.insertDocument(coll, line[jsonStart:])
	case "find":
		if jsonStart == -1 {
			return c.findDocuments(coll, "{}")
		}
		return c.findDocuments(coll, line[jsonStart:])
	case "update":
		return c.updateDocuments(coll, line)
	case "delete":
		return c.deleteDocuments(coll, line[jsonStart:])
	case "count":
		if jsonStart == -1 {
			return c.countDocuments(coll, "{}")
		}
		return c.countDocuments(coll, line[jsonStart:])
	}

	return nil
}

func (c *CLI) parseCollectionSyntax(line string) error {
	// Parse: collection.method({args})
	dotIdx := strings.Index(line, ".")
	if dotIdx == -1 {
		return fmt.Errorf("invalid syntax")
	}

	collName := line[:dotIdx]
	rest := line[dotIdx+1:]

	parenIdx := strings.Index(rest, "(")
	if parenIdx == -1 {
		return fmt.Errorf("invalid syntax: missing '('")
	}

	method := rest[:parenIdx]
	args := rest[parenIdx+1:]

	// Remove trailing ')'
	args = strings.TrimSuffix(strings.TrimSpace(args), ")")

	// Execute on collection
	coll := c.db.Collection(collName)

	switch strings.ToLower(method) {
	case "find":
		if args == "" {
			args = "{}"
		}
		return c.findDocuments(coll, args)
	case "insert", "insertone":
		return c.insertDocument(coll, args)
	case "count":
		if args == "" {
			args = "{}"
		}
		return c.countDocuments(coll, args)
	case "stats":
		return c.showStats(coll)
	case "getindexes":
		return c.getIndexes(coll)
	default:
		return fmt.Errorf("unknown method: %s", method)
	}
}

func (c *CLI) insertDocument(coll *database.Collection, jsonStr string) error {
	var doc map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &doc); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	id, err := coll.InsertOne(doc)
	if err != nil {
		return err
	}

	fmt.Printf("Inserted document with _id: %v\n", id)
	return nil
}

func (c *CLI) findDocuments(coll *database.Collection, jsonStr string) error {
	var query map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &query); err != nil {
		return fmt.Errorf("invalid JSON query: %w", err)
	}

	docs, err := coll.Find(query)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d document(s):\n", len(docs))
	for i, doc := range docs {
		jsonBytes, _ := json.MarshalIndent(doc.ToMap(), "", "  ")
		fmt.Printf("\n[%d] %s\n", i+1, string(jsonBytes))
	}

	return nil
}

func (c *CLI) updateDocuments(coll *database.Collection, line string) error {
	// Parse: update {query} {update}
	parts := strings.SplitN(line, "}", 2)
	if len(parts) < 2 {
		return fmt.Errorf("usage: update <query> <update>")
	}

	queryJSON := parts[0] + "}"
	updateJSON := strings.TrimSpace(parts[1])

	var query, update map[string]interface{}
	if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
		return fmt.Errorf("invalid query JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(updateJSON), &update); err != nil {
		return fmt.Errorf("invalid update JSON: %w", err)
	}

	err := coll.UpdateOne(query, update)
	if err != nil {
		return err
	}

	fmt.Println("Document updated successfully")
	return nil
}

func (c *CLI) deleteDocuments(coll *database.Collection, jsonStr string) error {
	var query map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &query); err != nil {
		return fmt.Errorf("invalid JSON query: %w", err)
	}

	err := coll.DeleteOne(query)
	if err != nil {
		return err
	}

	fmt.Println("Document deleted successfully")
	return nil
}

func (c *CLI) countDocuments(coll *database.Collection, jsonStr string) error {
	var query map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &query); err != nil {
		return fmt.Errorf("invalid JSON query: %w", err)
	}

	count, err := coll.Count(query)
	if err != nil {
		return err
	}

	fmt.Printf("Count: %d document(s)\n", count)
	return nil
}

func (c *CLI) managementCommand(cmd, line string) error {
	if c.currentColl == "" {
		return fmt.Errorf("no collection selected")
	}

	coll := c.db.Collection(c.currentColl)

	switch cmd {
	case "createindex":
		return c.createIndex(coll, line)
	case "getindexes":
		return c.getIndexes(coll)
	case "stats":
		return c.showStats(coll)
	}

	return nil
}

func (c *CLI) createIndex(coll *database.Collection, line string) error {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return fmt.Errorf("usage: createindex <field> [options]")
	}

	field := parts[1]

	// Default index config
	unique := false

	// Parse options if provided
	if len(parts) > 2 {
		optJSON := strings.Join(parts[2:], " ")
		var opts map[string]interface{}
		if err := json.Unmarshal([]byte(optJSON), &opts); err != nil {
			return fmt.Errorf("invalid options JSON: %w", err)
		}
		if u, ok := opts["unique"].(bool); ok {
			unique = u
		}
	}

	err := coll.CreateIndex(field, unique)
	if err != nil {
		return err
	}

	fmt.Printf("Created index on field '%s' (unique=%v)\n", field, unique)
	return nil
}

func (c *CLI) getIndexes(coll *database.Collection) error {
	indexes := coll.ListIndexes()

	fmt.Printf("Indexes on collection '%s':\n", c.currentColl)
	if len(indexes) == 0 {
		fmt.Println("  (no indexes)")
		return nil
	}

	for i, idx := range indexes {
		jsonBytes, _ := json.MarshalIndent(idx, "  ", "  ")
		fmt.Printf("\n[%d] %s\n", i+1, string(jsonBytes))
	}

	return nil
}

func (c *CLI) showStats(coll *database.Collection) error {
	stats := coll.Stats()

	jsonBytes, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("Collection statistics for '%s':\n%s\n", c.currentColl, string(jsonBytes))
	return nil
}

func main() {
	dataDir := "./laura-data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	cli, err := NewCLI(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer cli.Close()

	if err := cli.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
