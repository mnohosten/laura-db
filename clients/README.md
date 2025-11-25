# LauraDB Client Libraries

This directory contains official client libraries for LauraDB in various programming languages.

## Available Clients

### Node.js Client ✅
**Status**: Complete and production-ready

**Location**: `clients/nodejs/`

**Features**:
- Promise-based API
- Full CRUD operations
- Query builder with filtering, sorting, pagination
- Aggregation pipeline support
- Index management (B+ tree, compound, text, geo, TTL, partial)
- Connection pooling
- Zero external dependencies
- Comprehensive examples and tests

**Installation**:
```bash
cd clients/nodejs
npm install
```

**Quick Start**:
```javascript
const { createClient } = require('lauradb-client');

const client = createClient({ host: 'localhost', port: 8080 });
const users = client.collection('users');

const id = await users.insertOne({ name: 'Alice', age: 30 });
console.log('Inserted:', id);

client.close();
```

**Documentation**: See [clients/nodejs/README.md](nodejs/README.md)

---

### Python Client ✅
**Status**: Complete and production-ready

**Location**: `clients/python/`

**Features**:
- Pythonic API similar to pymongo
- Type hints for better IDE support
- Context manager support
- Query builder with all operators
- Aggregation pipeline support
- Index management (B+ tree, compound, text, geo, TTL, partial)
- Connection pooling
- Comprehensive examples and tests

**Installation**:
```bash
cd clients/python
pip install -e .
```

**Quick Start**:
```python
from lauradb import Client

client = Client(host='localhost', port=8080)
users = client.collection('users')

user_id = users.insert_one({'name': 'Alice', 'age': 30})
print('Inserted:', user_id)
```

**Documentation**: See [clients/python/README.md](python/README.md)

---

### Java Client ✅
**Status**: Complete and production-ready

**Location**: `clients/java/`

**Features**:
- Builder pattern API
- CompletableFuture support for async operations
- Query builder with fluent API
- Aggregation pipeline support
- Index management (B+ tree, compound, text, geo, TTL, partial)
- AutoCloseable support
- Comprehensive examples and tests
- Maven/Gradle integration

**Installation**:
```bash
cd clients/java
mvn clean install
```

**Quick Start**:
```java
import com.lauradb.client.*;

try (LauraDBClient client = LauraDBClient.builder()
        .host("localhost")
        .port(8080)
        .build()) {

    Collection users = client.collection("users");

    Map<String, Object> user = Map.of("name", "Alice", "age", 30);
    String id = users.insertOne(user);
    System.out.println("Inserted: " + id);
}
```

**Documentation**: See [clients/java/README.md](java/README.md)

---

## Client Development Guidelines

When developing new client libraries, follow these principles:

### 1. API Consistency
- Follow the conventions of the target language
- Mirror the MongoDB client API where appropriate
- Provide both simple and advanced APIs

### 2. Core Features
All clients should support:
- Connection management
- CRUD operations (Insert, Find, Update, Delete)
- Query building with filters
- Aggregation pipelines
- Index management
- Error handling

### 3. Performance
- Implement connection pooling
- Support batch operations
- Minimize serialization overhead
- Provide async APIs where appropriate

### 4. Documentation
Each client should include:
- README with quick start
- API reference
- Example programs
- Testing instructions

### 5. Testing
- Unit tests for client logic
- Integration tests against live server
- Error handling tests
- Performance benchmarks

## HTTP API Reference

All clients communicate with LauraDB via the HTTP API. See [docs/http-api.md](../docs/http-api.md) for the complete API specification.

### Key Endpoints

#### Health & Admin
- `GET /_health` - Health check
- `GET /_stats` - Database statistics
- `GET /_collections` - List collections

#### Document Operations
- `POST /{collection}/_doc` - Insert document
- `GET /{collection}/_doc/{id}` - Find by ID
- `PUT /{collection}/_doc/{id}` - Update document
- `DELETE /{collection}/_doc/{id}` - Delete document

#### Query & Search
- `POST /{collection}/_query` - Query documents
- `POST /{collection}/_count` - Count documents
- `POST /{collection}/_aggregate` - Aggregation pipeline

#### Index Management
- `POST /{collection}/_index` - Create index
- `GET /{collection}/_index` - List indexes
- `DELETE /{collection}/_index/{field}` - Drop index

## Contributing

We welcome contributions for new client libraries! Please ensure:

1. The client follows the guidelines above
2. Comprehensive tests are included
3. Documentation is complete
4. Examples demonstrate key features
5. Code follows the target language's conventions

See the Node.js client as a reference implementation.

## License

All client libraries are released under the MIT License.

## Support

- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs
