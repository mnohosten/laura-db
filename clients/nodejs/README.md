# LauraDB Node.js Client

Official Node.js client library for [LauraDB](https://github.com/mnohosten/laura-db) - A MongoDB-like document database written in Go.

## Features

- **Simple API** - Intuitive, Promise-based API similar to MongoDB drivers
- **Full CRUD Support** - Insert, find, update, and delete operations
- **Query Builder** - Fluent query builder with filtering, sorting, and pagination
- **Aggregation Pipeline** - Powerful aggregation framework with $match, $group, $project, etc.
- **Index Management** - Support for B+ tree, compound, text, geospatial, TTL, and partial indexes
- **Connection Pooling** - Built-in HTTP connection pooling for performance
- **TypeScript Ready** - Includes TypeScript definitions (coming soon)
- **Zero Dependencies** - Pure Node.js implementation using only built-in modules

## Installation

```bash
npm install lauradb-client
```

## Quick Start

```javascript
const { createClient } = require('lauradb-client');

// Create a client
const client = createClient({
  host: 'localhost',
  port: 8080
});

// Get a collection
const users = client.collection('users');

// Insert a document
const userId = await users.insertOne({
  name: 'Alice',
  email: 'alice@example.com',
  age: 30
});

// Find documents
const results = await users.find()
  .filter({ age: { $gt: 25 } })
  .sort({ name: 1 })
  .limit(10)
  .execute();

console.log(results);

// Close the client
client.close();
```

## Configuration

```javascript
const client = createClient({
  host: 'localhost',        // Server hostname (default: 'localhost')
  port: 8080,               // Server port (default: 8080)
  https: false,             // Use HTTPS (default: false)
  timeout: 30000,           // Request timeout in ms (default: 30000)
  maxSockets: 10            // Max concurrent connections (default: 10)
});
```

## API Reference

### Client

#### Health & Statistics

```javascript
// Check server health
const health = await client.health();
console.log(health.status); // 'healthy'

// Get database statistics
const stats = await client.stats();
console.log(stats.collections); // Number of collections
```

#### Collection Management

```javascript
// List all collections
const collections = await client.listCollections();

// Create a collection
await client.createCollection('users');

// Drop a collection
await client.dropCollection('users');

// Get a collection handle
const users = client.collection('users');
```

### Collection

#### Insert Operations

```javascript
// Insert one document with auto-generated ID
const id = await users.insertOne({
  name: 'Alice',
  age: 30
});

// Insert one document with specific ID
await users.insertOneWithID('user1', {
  name: 'Bob',
  age: 25
});

// Insert many documents
const ids = await users.insertMany([
  { name: 'Carol', age: 28 },
  { name: 'Dave', age: 35 }
]);
```

#### Find Operations

```javascript
// Find one document by ID
const user = await users.findOne('user1');

// Find with query builder
const results = await users.find()
  .filter({ age: { $gt: 25 } })
  .project({ name: 1, age: 1 })
  .sort({ age: -1 })
  .limit(10)
  .skip(20)
  .execute();

// Find first matching document
const firstUser = await users.find()
  .filter({ email: 'alice@example.com' })
  .first();

// Count documents
const count = await users.count({ age: { $gt: 25 } });
```

#### Update Operations

```javascript
// Update one document by ID
await users.updateOne('user1', {
  $set: { age: 31 },
  $inc: { loginCount: 1 }
});

// Update many documents
const updatedCount = await users.find()
  .filter({ age: { $lt: 18 } })
  .update({ $set: { minor: true } });
```

#### Delete Operations

```javascript
// Delete one document by ID
await users.deleteOne('user1');

// Delete many documents
const deletedCount = await users.find()
  .filter({ inactive: true })
  .delete();
```

#### Bulk Operations

```javascript
const result = await users.bulk([
  {
    operation: 'insert',
    document: { name: 'Alice', age: 30 }
  },
  {
    operation: 'update',
    _id: 'user1',
    update: { $set: { age: 31 } }
  },
  {
    operation: 'delete',
    _id: 'user2'
  }
]);

console.log(result.inserted); // Number of documents inserted
console.log(result.updated);  // Number of documents updated
console.log(result.deleted);  // Number of documents deleted
```

### Query Builder

The query builder supports all MongoDB query operators:

```javascript
// Comparison operators
await users.find().filter({ age: { $eq: 30 } }).execute();
await users.find().filter({ age: { $ne: 30 } }).execute();
await users.find().filter({ age: { $gt: 25 } }).execute();
await users.find().filter({ age: { $gte: 25 } }).execute();
await users.find().filter({ age: { $lt: 35 } }).execute();
await users.find().filter({ age: { $lte: 35 } }).execute();

// Logical operators
await users.find().filter({
  $and: [
    { age: { $gt: 25 } },
    { city: 'New York' }
  ]
}).execute();

await users.find().filter({
  $or: [
    { age: { $lt: 18 } },
    { age: { $gt: 65 } }
  ]
}).execute();

await users.find().filter({
  age: { $not: { $eq: 30 } }
}).execute();

// Array operators
await users.find().filter({ tags: { $in: ['javascript', 'nodejs'] } }).execute();
await users.find().filter({ tags: { $nin: ['python', 'java'] } }).execute();
await users.find().filter({ tags: { $all: ['javascript', 'nodejs'] } }).execute();
await users.find().filter({ skills: { $size: 3 } }).execute();

// Element operators
await users.find().filter({ email: { $exists: true } }).execute();
await users.find().filter({ age: { $type: 'number' } }).execute();

// Regular expressions
await users.find().filter({ name: { $regex: '^A.*' } }).execute();

// Query plan explanation
const plan = await users.find()
  .filter({ age: { $gt: 25 } })
  .explain();
console.log(plan.index_used); // Shows which index was selected
```

### Aggregation Pipeline

```javascript
// Basic aggregation
const results = await users.aggregate()
  .match({ age: { $gt: 25 } })
  .group({
    _id: '$city',
    avgAge: { $avg: '$age' },
    count: { $sum: 1 }
  })
  .sort({ avgAge: -1 })
  .limit(10)
  .execute();

// Multi-stage pipeline
const results = await users.aggregate()
  .match({ active: true })
  .project({ name: 1, age: 1, city: 1 })
  .group({
    _id: '$city',
    users: { $push: '$name' },
    avgAge: { $avg: '$age' },
    minAge: { $min: '$age' },
    maxAge: { $max: '$age' }
  })
  .sort({ avgAge: -1 })
  .skip(10)
  .limit(5)
  .execute();
```

### Index Management

```javascript
const indexes = users.indexes();

// B+ tree index
await indexes.create('email', { unique: true });
await indexes.create('age', { unique: false, sparse: true });

// Compound index
await indexes.createCompound(
  { city: 1, age: -1 },
  { unique: false }
);

// Text index for full-text search
await indexes.createText(['title', 'description']);

// Geospatial index
await indexes.createGeo('location', '2dsphere');

// TTL index (automatic expiration)
await indexes.createTTL('createdAt', 86400); // Expire after 24 hours

// Partial index (conditional indexing)
await indexes.createPartial(
  'email',
  { active: true },
  { unique: true }
);

// List all indexes
const allIndexes = await indexes.list();

// Drop an index
await indexes.drop('email');
await indexes.dropCompound('city_1_age_1');
```

### Collection Statistics

```javascript
const stats = await users.stats();
console.log(stats.count);   // Number of documents
console.log(stats.indexes);  // Number of indexes
```

## Update Operators

LauraDB supports the following update operators:

### Field Operators
- `$set` - Set field values
- `$unset` - Remove fields
- `$rename` - Rename fields
- `$currentDate` - Set to current date/time

### Numeric Operators
- `$inc` - Increment numeric values
- `$mul` - Multiply numeric values
- `$min` - Update if less than current value
- `$max` - Update if greater than current value

### Array Operators
- `$push` - Add element to array
- `$pull` - Remove matching elements from array
- `$pullAll` - Remove multiple values from array
- `$addToSet` - Add unique element to array
- `$pop` - Remove first or last element

### Bitwise Operators
- `$bit` - Perform bitwise operations (and, or, xor)

## Examples

### Complete CRUD Example

```javascript
const { createClient } = require('lauradb-client');

async function main() {
  const client = createClient({ host: 'localhost', port: 8080 });
  const users = client.collection('users');

  try {
    // Insert
    const id = await users.insertOne({
      name: 'Alice',
      email: 'alice@example.com',
      age: 30,
      city: 'New York'
    });
    console.log('Inserted:', id);

    // Find
    const user = await users.findOne(id);
    console.log('Found:', user);

    // Update
    await users.updateOne(id, {
      $set: { age: 31 },
      $inc: { loginCount: 1 }
    });

    // Query
    const results = await users.find()
      .filter({ city: 'New York' })
      .sort({ age: -1 })
      .execute();
    console.log('Query results:', results);

    // Delete
    await users.deleteOne(id);
    console.log('Deleted');

  } finally {
    client.close();
  }
}

main().catch(console.error);
```

### Aggregation Example

```javascript
// Group users by city and calculate statistics
const results = await users.aggregate()
  .match({ age: { $gt: 18 } })
  .group({
    _id: '$city',
    totalUsers: { $sum: 1 },
    avgAge: { $avg: '$age' },
    minAge: { $min: '$age' },
    maxAge: { $max: '$age' },
    userNames: { $push: '$name' }
  })
  .sort({ totalUsers: -1 })
  .limit(10)
  .execute();

console.log(results);
```

### Error Handling

```javascript
try {
  const user = await users.findOne('nonexistent-id');
  console.log(user); // null if not found
} catch (err) {
  console.error('Error:', err.message);
  console.error('API Error:', err.apiError);
  console.error('Status Code:', err.code);
}
```

## Requirements

- Node.js 14.0.0 or higher
- LauraDB server running (see [main README](https://github.com/mnohosten/laura-db))

## Testing

```bash
# Install dependencies
npm install

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

## License

MIT

## Related

- [LauraDB](https://github.com/mnohosten/laura-db) - Main repository
- [LauraDB Go Client](../../pkg/client) - Official Go client
- [HTTP API Documentation](../../docs/http-api.md) - REST API reference

## Contributing

Contributions are welcome! Please see the [main repository](https://github.com/mnohosten/laura-db) for contribution guidelines.

## Support

- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs
