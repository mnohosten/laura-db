# Node.js Client Library

The LauraDB Node.js client provides a clean, Promise-based API for interacting with LauraDB servers from Node.js applications.

## Installation

```bash
cd clients/nodejs
npm install
```

Or from your application:

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

// Use the client
const users = client.collection('users');
const id = await users.insertOne({ name: 'Alice', age: 30 });
console.log('Inserted:', id);

// Close when done
client.close();
```

## Architecture

The Node.js client is organized into several modules:

### Core Modules

1. **Client** (`lib/client.js`)
   - Main entry point for database connections
   - HTTP connection pooling
   - Health checks and statistics
   - Collection management

2. **Collection** (`lib/collection.js`)
   - CRUD operations (Insert, Find, Update, Delete)
   - Bulk operations
   - Collection statistics
   - Factory for Query, Aggregation, and Index builders

3. **Query** (`lib/query.js`)
   - Fluent query builder
   - Filtering, projection, sorting, pagination
   - Query execution and explanation
   - Batch update/delete operations

4. **Aggregation** (`lib/aggregation.js`)
   - Pipeline builder
   - Stages: $match, $group, $project, $sort, $limit, $skip
   - Aggregation operators: $sum, $avg, $min, $max, $push

5. **Index** (`lib/index.js`)
   - Index creation and management
   - Support for all index types (B+ tree, compound, text, geo, TTL, partial)
   - Index listing and deletion

## API Overview

### Client Configuration

```javascript
const client = createClient({
  host: 'localhost',        // Server hostname (default: 'localhost')
  port: 8080,               // Server port (default: 8080)
  https: false,             // Use HTTPS (default: false)
  timeout: 30000,           // Request timeout in ms (default: 30000)
  maxSockets: 10            // Max concurrent connections (default: 10)
});
```

### Client Methods

```javascript
// Health and statistics
await client.health()
await client.stats()

// Collection management
await client.listCollections()
await client.createCollection(name)
await client.dropCollection(name)
const collection = client.collection(name)

// Cleanup
client.close()
```

### Collection Methods

```javascript
const collection = client.collection('users');

// Insert
await collection.insertOne(doc)
await collection.insertOneWithID(id, doc)
await collection.insertMany(docs)

// Find
await collection.findOne(id)
await collection.count(filter)

// Update
await collection.updateOne(id, update)

// Delete
await collection.deleteOne(id)

// Query builder
collection.find()

// Aggregation
collection.aggregate()

// Index management
collection.indexes()

// Statistics
await collection.stats()

// Drop collection
await collection.drop()
```

### Query Builder

```javascript
const results = await collection.find()
  .filter({ age: { $gt: 25 } })
  .project({ name: 1, age: 1 })
  .sort({ age: -1 })
  .limit(10)
  .skip(20)
  .execute();

// Other query methods
await query.first()          // Get first result
await query.count()          // Count matches
await query.update(update)   // Batch update
await query.delete()         // Batch delete
await query.explain()        // Query plan
```

### Aggregation Pipeline

```javascript
const results = await collection.aggregate()
  .match({ age: { $gt: 25 } })
  .group({
    _id: '$city',
    avgAge: { $avg: '$age' },
    count: { $sum: 1 }
  })
  .sort({ avgAge: -1 })
  .limit(10)
  .execute();
```

### Index Management

```javascript
const indexes = collection.indexes();

// Create indexes
await indexes.create('email', { unique: true })
await indexes.createCompound({ city: 1, age: -1 })
await indexes.createText(['title', 'description'])
await indexes.createGeo('location', '2dsphere')
await indexes.createTTL('createdAt', 86400)
await indexes.createPartial('email', { active: true }, { unique: true })

// List and drop
await indexes.list()
await indexes.drop('email')
await indexes.dropCompound('city_1_age_1')
```

## Query Operators

The client supports all LauraDB query operators:

### Comparison Operators
- `$eq` - Equal
- `$ne` - Not equal
- `$gt` - Greater than
- `$gte` - Greater than or equal
- `$lt` - Less than
- `$lte` - Less than or equal
- `$in` - In array
- `$nin` - Not in array

### Logical Operators
- `$and` - Logical AND
- `$or` - Logical OR
- `$not` - Logical NOT

### Element Operators
- `$exists` - Field exists
- `$type` - Field type check

### Array Operators
- `$all` - Array contains all
- `$elemMatch` - Array element matches
- `$size` - Array size

### Evaluation Operators
- `$regex` - Regular expression match

## Update Operators

### Field Operators
- `$set` - Set field values
- `$unset` - Remove fields
- `$rename` - Rename fields
- `$currentDate` - Set to current date

### Numeric Operators
- `$inc` - Increment
- `$mul` - Multiply
- `$min` - Update if less
- `$max` - Update if greater

### Array Operators
- `$push` - Add to array
- `$pull` - Remove from array
- `$pullAll` - Remove multiple values
- `$addToSet` - Add unique to array
- `$pop` - Remove first/last

### Bitwise Operators
- `$bit` - Bitwise operations (and, or, xor)

## Error Handling

```javascript
try {
  const user = await collection.findOne('invalid-id');
} catch (err) {
  console.error('Error:', err.message);
  console.error('API Error:', err.apiError);
  console.error('Status Code:', err.code);
}
```

Error objects include:
- `message` - Error description
- `apiError` - API error type
- `code` - HTTP status code
- `response` - Full API response

## Connection Pooling

The client automatically manages HTTP connection pooling using Node.js's http.Agent:

- **keepAlive**: Connections are kept alive for reuse
- **maxSockets**: Configurable maximum concurrent connections (default: 10)
- **Automatic cleanup**: Connections are closed when client.close() is called

## Performance Best Practices

1. **Reuse Client Instances**
   ```javascript
   // Good - single client instance
   const client = createClient();

   // Use for multiple operations
   await client.collection('users').insertOne(...);
   await client.collection('products').insertOne(...);

   client.close();
   ```

2. **Use Bulk Operations**
   ```javascript
   // Good - bulk insert
   await collection.insertMany(docs);

   // Avoid - multiple individual inserts
   for (const doc of docs) {
     await collection.insertOne(doc); // Slower
   }
   ```

3. **Use Projections**
   ```javascript
   // Good - only fetch needed fields
   await collection.find()
     .filter({ age: { $gt: 25 } })
     .project({ name: 1, email: 1 })
     .execute();
   ```

4. **Create Appropriate Indexes**
   ```javascript
   // Index frequently queried fields
   await collection.indexes().create('email', { unique: true });
   await collection.indexes().createCompound({ city: 1, age: -1 });
   ```

5. **Use Query Explain**
   ```javascript
   // Check if queries are using indexes
   const plan = await collection.find()
     .filter({ age: { $gt: 25 } })
     .explain();

   console.log('Index used:', plan.index_used);
   ```

## Examples

See the `clients/nodejs/examples/` directory for complete examples:

- `basic-usage.js` - Core CRUD operations
- `aggregation.js` - Aggregation pipeline examples
- `indexes.js` - Index management examples

## Testing

```bash
cd clients/nodejs

# Install dependencies
npm install

# Run tests (requires LauraDB server on localhost:8080)
npm test

# Run with coverage
npm run test:coverage

# Run integration tests
INTEGRATION_TESTS=true npm test
```

## Troubleshooting

### Connection Refused

**Problem**: `Error: connect ECONNREFUSED 127.0.0.1:8080`

**Solution**: Ensure LauraDB server is running:
```bash
./bin/laura-server -port 8080
```

### Timeout Errors

**Problem**: `Error: Request timeout after 30000ms`

**Solution**: Increase timeout in client config:
```javascript
const client = createClient({
  host: 'localhost',
  port: 8080,
  timeout: 60000  // 60 seconds
});
```

### JSON Parse Errors

**Problem**: `Error: Failed to parse response`

**Solution**: Check that the server is returning valid JSON. This could indicate:
- Server is not a LauraDB instance
- Server returned HTML error page
- Network proxy interference

## TypeScript Support

TypeScript definitions are planned for a future release. For now, you can use JSDoc annotations:

```javascript
/**
 * @typedef {Object} User
 * @property {string} _id
 * @property {string} name
 * @property {number} age
 * @property {string} email
 */

/**
 * @type {User[]}
 */
const users = await collection.find().execute();
```

## Contributing

Contributions to the Node.js client are welcome! Please see the main repository for contribution guidelines.

## License

MIT

## See Also

- [HTTP API Documentation](http-api.md)
- [Go Client Documentation](go-client.md)
- [LauraDB Main Documentation](../README.md)
