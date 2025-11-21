# LauraDB CLI Tool

A command-line interface (REPL) for interacting with LauraDB - a MongoDB-like document database.

## Building

```bash
# From project root
make cli

# Or directly with go
go build -o ../../bin/laura-cli
```

## Running

```bash
# Run with default data directory (./laura-data)
./bin/laura-cli

# Run with custom data directory
./bin/laura-cli /path/to/data
```

## Features

- **Interactive REPL** with command history
- **MongoDB-like syntax** for familiar commands
- **JSON document support** with full query capabilities
- **Index management** and statistics
- **Pretty-printed results** for easy reading

## Commands

### Basic Commands

| Command | Description |
|---------|-------------|
| `help` or `?` | Show help message |
| `exit` or `quit` | Exit the CLI |
| `clear` | Clear the screen |
| `version` | Show CLI version |
| `use <collection>` | Switch to a collection |

### Collection Operations

#### Insert Documents

```bash
# Select a collection first
use users

# Insert a document
insert {"name": "Alice", "age": 25, "email": "alice@example.com"}
```

#### Find Documents

```bash
# Find all documents in current collection
find

# Find with query
find {"age": {"$gte": 21}}

# Find with regex
find {"name": {"$regex": "^A"}}
```

#### Update Documents

```bash
# Update first matching document
update {"name": "Alice"} {"$set": {"age": 26}}

# Update with multiple operators
update {"name": "Bob"} {"$set": {"status": "active"}, "$inc": {"loginCount": 1}}
```

#### Delete Documents

```bash
# Delete first matching document
delete {"name": "Alice"}

# Delete with complex query
delete {"age": {"$lt": 18}}
```

#### Count Documents

```bash
# Count all documents
count

# Count with filter
count {"status": "active"}
```

### MongoDB-like Syntax

You can also use MongoDB-style collection.method() syntax:

```bash
# Find
users.find({"age": {"$gte": 21}})

# Insert
users.insert({"name": "Charlie", "age": 30})

# Count
users.count()

# Stats
users.stats()
```

### Index Management

```bash
# Select a collection
use users

# Create an index
createindex email {"unique": true}

# Create a non-unique index
createindex name

# List all indexes
getindexes
```

### Statistics

```bash
# Show collection statistics
stats

# Using MongoDB syntax
users.stats()
```

## Query Examples

### Comparison Operators

```bash
# Greater than or equal
find {"age": {"$gte": 21}}

# Less than
find {"score": {"$lt": 100}}

# Not equal
find {"status": {"$ne": "deleted"}}
```

### Logical Operators

```bash
# AND (implicit - all conditions must match)
find {"age": {"$gte": 21}, "status": "active"}

# OR
find {"$or": [{"age": {"$lt": 18}}, {"age": {"$gt": 65}}]}
```

### Array Operators

```bash
# In array
find {"status": {"$in": ["active", "pending"]}}

# Element match
find {"scores": {"$elemMatch": {"$gte": 80, "$lt": 90}}}

# Array size
find {"tags": {"$size": 3}}
```

### Regex Patterns

```bash
# Starts with
find {"name": {"$regex": "^A"}}

# Case insensitive
find {"email": {"$regex": "(?i)@example\\.com$"}}

# Contains
find {"description": {"$regex": "mongodb"}}
```

### Update Operators

```bash
# Set fields
update {"_id": "123"} {"$set": {"status": "active", "lastLogin": "2024-01-01"}}

# Increment number
update {"name": "Alice"} {"$inc": {"score": 10}}

# Push to array
update {"name": "Bob"} {"$push": {"tags": "premium"}}

# Push multiple with $each
update {"name": "Bob"} {"$push": {"tags": {"$each": ["gold", "vip"]}}}

# Add unique to array
update {"name": "Charlie"} {"$addToSet": {"skills": "Go"}}

# Rename field
update {"name": "Diana"} {"$rename": {"old_field": "new_field"}}

# Set current date
update {"name": "Eve"} {"$currentDate": {"lastModified": true}}
```

## Tips

1. **JSON Format**: Always use double quotes for JSON strings, not single quotes
   - ✅ Correct: `{"name": "Alice"}`
   - ❌ Wrong: `{'name': 'Alice'}`

2. **Collection Selection**: Remember to `use <collection>` before running collection operations

3. **Command History**: Use up/down arrows to navigate command history

4. **Complex Queries**: For complex queries, you can format them across multiple lines in your text editor and paste them

5. **Data Directory**: All data is stored in the specified data directory (default: `./laura-data`)

## Example Session

```bash
$ ./bin/laura-cli

╔══════════════════════════════════════╗
║        LauraDB CLI v0.1.0           ║
║  MongoDB-like Document Database     ║
╚══════════════════════════════════════╝

Type 'help' for available commands
Type 'exit' or 'quit' to exit

laura> use users
Switched to collection 'users'

laura:users> insert {"name": "Alice", "age": 25, "email": "alice@example.com"}
Inserted document with _id: 507f1f77bcf86cd799439011

laura:users> insert {"name": "Bob", "age": 30, "email": "bob@example.com"}
Inserted document with _id: 507f1f77bcf86cd799439012

laura:users> find
Found 2 document(s):

[1] {
  "_id": "507f1f77bcf86cd799439011",
  "age": 25,
  "email": "alice@example.com",
  "name": "Alice"
}

[2] {
  "_id": "507f1f77bcf86cd799439012",
  "age": 30,
  "email": "bob@example.com",
  "name": "Bob"
}

laura:users> find {"age": {"$gte": 28}}
Found 1 document(s):

[1] {
  "_id": "507f1f77bcf86cd799439012",
  "age": 30,
  "email": "bob@example.com",
  "name": "Bob"
}

laura:users> createindex email {"unique": true}
Created index on field 'email' (unique=true)

laura:users> update {"name": "Alice"} {"$set": {"age": 26}, "$push": {"tags": "verified"}}
Document updated successfully

laura:users> count
Count: 2 document(s)

laura:users> stats
Collection statistics for 'users':
{
  "document_count": 2,
  "index_count": 2,
  "indexes": [...]
}

laura:users> exit
Goodbye!
```

## Troubleshooting

### "No collection selected" Error

Make sure to run `use <collection>` before executing collection operations.

### JSON Parse Errors

- Check that you're using double quotes for strings
- Ensure JSON is properly formatted
- Use online JSON validators if needed

### Database Lock Issues

If the database is already open by another process (e.g., the server), you may get lock errors. Make sure only one process accesses the database at a time, or use different data directories.

## Architecture

The CLI tool:
- Uses the LauraDB Go library directly (no network calls)
- Opens the database in embedded mode
- Supports all query and update operators
- Provides a simple REPL for interactive use

## Limitations

- Single-user: Only one CLI instance can access a database at a time
- No transaction control: Each command auto-commits
- Limited editing: Basic readline functionality (no advanced editing features)

## Future Enhancements

Planned improvements:
- Tab completion for commands and collection names
- Query history persistence across sessions
- Script file execution (batch mode)
- Export/import commands for data migration
- Connection to remote LauraDB servers via HTTP API
- Better error messages with suggestions

## Contributing

The CLI tool is part of the LauraDB project. See the main project README for contribution guidelines.

## License

Same as LauraDB project.
