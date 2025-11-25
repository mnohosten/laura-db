# LauraDB Java Client

Java client library for [LauraDB](https://github.com/mnohosten/laura-db) - A MongoDB-like document database written in Go.

## Features

- 🔌 **Simple Connection Management**: Builder pattern API with connection configuration
- 📦 **Complete CRUD Operations**: Insert, Find, Update, Delete with MongoDB-like API
- 🔍 **Rich Query Language**: Full support for comparison, logical, array, and element operators
- 📊 **Aggregation Pipeline**: $match, $group, $project, $sort, $limit, $skip stages
- 🗂️ **Index Management**: B+ tree, compound, text, geospatial, TTL, and partial indexes
- ⚡ **High Performance**: Efficient HTTP communication with connection reuse
- ☕ **Idiomatic Java**: Builder patterns, CompletableFuture support, AutoCloseable
- 🎯 **Type-Safe**: Strongly typed API with generics support
- 📝 **Zero Dependencies**: Only requires Gson for JSON serialization

## Requirements

- Java 11 or higher
- LauraDB server running (default: localhost:8080)
- Maven or Gradle for dependency management

## Installation

### Maven

Add to your `pom.xml`:

```xml
<dependency>
    <groupId>com.lauradb</groupId>
    <artifactId>lauradb-client</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

Add to your `build.gradle`:

```gradle
implementation 'com.lauradb:lauradb-client:1.0.0'
```

### Build from Source

```bash
cd clients/java
mvn clean install
```

## Quick Start

```java
import com.lauradb.client.*;
import java.util.*;

public class QuickStart {
    public static void main(String[] args) throws Exception {
        // Connect to LauraDB server
        try (LauraDBClient client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build()) {

            // Get a collection
            Collection users = client.collection("users");

            // Insert a document
            Map<String, Object> user = new HashMap<>();
            user.put("name", "Alice");
            user.put("email", "alice@example.com");
            user.put("age", 30);

            String id = users.insertOne(user);
            System.out.println("Inserted: " + id);

            // Find documents
            Query query = Query.builder()
                    .gte("age", 25)
                    .build();

            List<Map<String, Object>> results = users.find(query);
            for (Map<String, Object> doc : results) {
                System.out.println(doc);
            }

            // Update documents
            Map<String, Object> update = new HashMap<>();
            Map<String, Object> setOp = new HashMap<>();
            setOp.put("age", 31);
            update.put("$set", setOp);

            users.updateOne(Query.builder().eq("name", "Alice").build(), update);

            // Delete documents
            users.deleteOne(Query.builder().eq("name", "Alice").build());
        }
    }
}
```

## Advanced Usage

### Client Configuration

```java
LauraDBClient client = LauraDBClient.builder()
        .host("localhost")
        .port(8080)
        .https(false)
        .timeout(30000) // 30 seconds
        .build();
```

### Query Operators

#### Comparison Operators

```java
// Greater than / Less than
Query query = Query.builder()
        .gt("age", 25)
        .lt("age", 40)
        .build();

// In / Not in
Query query = Query.builder()
        .in("category", "A", "B", "C")
        .nin("status", "deleted", "archived")
        .build();
```

#### Logical Operators

```java
// AND
Query query = Query.builder()
        .and(
            Query.builder().gte("age", 25).build(),
            Query.builder().eq("active", true).build()
        )
        .build();

// OR
Query query = Query.builder()
        .or(
            Query.builder().eq("city", "New York").build(),
            Query.builder().eq("city", "Boston").build()
        )
        .build();

// NOT
Query query = Query.builder()
        .not(Query.builder().eq("status", "deleted").build())
        .build();
```

#### Array Operators

```java
// All elements match
Query query = Query.builder()
        .all("tags", "java", "database")
        .build();

// Array size
Query query = Query.builder()
        .size("items", 5)
        .build();

// Element match
Query elemQuery = Query.builder()
        .gte("score", 80)
        .build();
Query query = Query.builder()
        .elemMatch("grades", elemQuery)
        .build();
```

#### Element Operators

```java
// Field exists
Query query = Query.builder()
        .exists("email", true)
        .build();

// Type check
Query query = Query.builder()
        .type("age", "number")
        .build();
```

#### Evaluation Operators

```java
// Regular expression
Query query = Query.builder()
        .regex("email", ".*@example\\.com$")
        .build();
```

### Find Options

```java
// Projection: select specific fields
FindOptions options = new FindOptions()
        .projection(FindOptions.projectionBuilder()
                .include("name")
                .include("email")
                .exclude("_id")
                .build());

// Sorting
FindOptions options = new FindOptions()
        .sort(FindOptions.sortBuilder()
                .descending("age")
                .ascending("name")
                .build());

// Pagination
FindOptions options = new FindOptions()
        .skip(20)
        .limit(10);

// Combine all options
FindOptions options = new FindOptions()
        .projection(FindOptions.projectionBuilder()
                .include("name")
                .include("email")
                .build())
        .sort(FindOptions.sortBuilder()
                .descending("createdAt")
                .build())
        .skip(0)
        .limit(50);

List<Map<String, Object>> results = collection.find(query, options);
```

### Aggregation Pipeline

```java
// Group by field and calculate aggregates
Aggregation pipeline = Aggregation.builder()
        .match(Query.builder().gte("age", 18).build())
        .group("$city")
            .avg("avgAge", "$age")
            .count("count")
            .sum("totalAmount", "$amount")
            .max("maxAge", "$age")
            .min("minAge", "$age")
        .sort(Map.of("avgAge", -1))
        .limit(10)
        .build();

List<Map<String, Object>> results = collection.aggregate(pipeline);
```

### Index Management

#### B+ Tree Index

```java
// Simple index
collection.createIndex("email")
        .unique(true)
        .build();

// Background index building
collection.createIndex("age")
        .background(true)
        .build();
```

#### Compound Index

```java
collection.createCompoundIndex(List.of("city", "age"))
        .name("city_age_idx")
        .unique(false)
        .build();
```

#### Text Index

```java
collection.createIndex("content")
        .text()
        .name("content_text")
        .build();
```

#### Geospatial Index

```java
// 2dsphere for Earth coordinates
collection.createIndex("location")
        .geo("2dsphere")
        .build();

// 2d for planar coordinates
collection.createIndex("coordinates")
        .geo("2d")
        .build();
```

#### TTL Index

```java
// Documents expire after 3600 seconds
collection.createIndex("createdAt")
        .ttl(3600)
        .build();
```

#### Partial Index

```java
// Only index active documents
collection.createIndex("email")
        .unique(true)
        .partial(Query.builder().eq("active", true).build())
        .build();
```

#### List and Drop Indexes

```java
// List all indexes
List<String> indexes = collection.listIndexes();

// Drop an index
collection.dropIndex("email");
```

### Update Operations

```java
// $set - set field values
Map<String, Object> update = new HashMap<>();
Map<String, Object> setOp = new HashMap<>();
setOp.put("status", "active");
setOp.put("lastModified", new Date().getTime());
update.put("$set", setOp);

// $inc - increment numeric values
Map<String, Object> incOp = new HashMap<>();
incOp.put("visits", 1);
incOp.put("score", 10);
update.put("$inc", incOp);

// $unset - remove fields
Map<String, Object> unsetOp = new HashMap<>();
unsetOp.put("temporaryField", "");
update.put("$unset", unsetOp);

// $push - add to array
Map<String, Object> pushOp = new HashMap<>();
pushOp.put("tags", "java");
update.put("$push", pushOp);

// Apply update
collection.updateOne(query, update);
```

### Asynchronous Operations

```java
// Async insert
CompletableFuture<String> insertFuture = collection.insertOneAsync(document);
String id = insertFuture.get();

// Async find
CompletableFuture<List<Map<String, Object>>> findFuture = collection.findAsync(query);
List<Map<String, Object>> results = findFuture.get();

// Async ping
CompletableFuture<Boolean> pingFuture = client.pingAsync();
Boolean healthy = pingFuture.get();
```

### Bulk Operations

```java
// Insert multiple documents
List<Map<String, Object>> documents = new ArrayList<>();
for (int i = 0; i < 100; i++) {
    Map<String, Object> doc = new HashMap<>();
    doc.put("index", i);
    doc.put("value", i * 10);
    documents.add(doc);
}

List<String> ids = collection.insertMany(documents);
System.out.println("Inserted " + ids.size() + " documents");
```

## Examples

The `examples/` directory contains complete working examples:

- **BasicUsage.java**: CRUD operations, queries, updates, deletes
- **AggregationExample.java**: Aggregation pipelines with grouping and operators
- **IndexExample.java**: Creating and managing various index types

### Running Examples

```bash
cd clients/java

# Compile
mvn clean compile

# Run basic usage example
mvn exec:java -Dexec.mainClass="BasicUsage"

# Run aggregation example
mvn exec:java -Dexec.mainClass="AggregationExample"

# Run index example
mvn exec:java -Dexec.mainClass="IndexExample"
```

## Testing

```bash
# Run all tests
mvn test

# Run tests with coverage
mvn test jacoco:report

# Run specific test
mvn test -Dtest=LauraDBClientTest
```

**Note**: Tests require a running LauraDB server on localhost:8080

## API Reference

See [docs/java-client.md](../../docs/java-client.md) for complete API documentation.

## Error Handling

```java
try (LauraDBClient client = LauraDBClient.builder()
        .host("localhost")
        .port(8080)
        .build()) {

    Collection users = client.collection("users");

    try {
        users.insertOne(document);
    } catch (IOException e) {
        System.err.println("Failed to insert document: " + e.getMessage());
    }

} catch (Exception e) {
    System.err.println("Client error: " + e.getMessage());
}
```

## Best Practices

### 1. Use try-with-resources

```java
try (LauraDBClient client = LauraDBClient.builder().build()) {
    // Your code here
} // Client automatically closed
```

### 2. Reuse Client Instances

```java
// Good - reuse client
LauraDBClient client = LauraDBClient.builder().build();
Collection users = client.collection("users");
Collection posts = client.collection("posts");

// Avoid - creating multiple clients
LauraDBClient client1 = LauraDBClient.builder().build();
LauraDBClient client2 = LauraDBClient.builder().build();
```

### 3. Use Appropriate Indexes

```java
// Create indexes for frequently queried fields
collection.createIndex("email").unique(true).build();
collection.createIndex("createdAt").build();
collection.createCompoundIndex(List.of("category", "status")).build();
```

### 4. Use Projections for Large Documents

```java
// Only fetch needed fields
FindOptions options = new FindOptions()
        .projection(FindOptions.projectionBuilder()
                .include("name")
                .include("email")
                .build());
```

### 5. Handle Pagination Properly

```java
int pageSize = 20;
int pageNumber = 1;

FindOptions options = new FindOptions()
        .skip((pageNumber - 1) * pageSize)
        .limit(pageSize);

List<Map<String, Object>> page = collection.find(query, options);
```

## Performance Tips

1. **Use bulk operations**: `insertMany()` is more efficient than multiple `insertOne()` calls
2. **Create appropriate indexes**: Speed up queries with indexes on frequently queried fields
3. **Use projections**: Reduce network overhead by selecting only needed fields
4. **Implement pagination**: Use skip/limit for large result sets
5. **Use background index building**: Build indexes without blocking operations
6. **Connection pooling**: HttpURLConnection automatically reuses connections

## Contributing

Contributions are welcome! Please ensure:

1. Code follows Java conventions
2. All tests pass
3. New features include tests and documentation
4. Examples demonstrate key features

## License

MIT License - see LICENSE file for details.

## Links

- [LauraDB Repository](https://github.com/mnohosten/laura-db)
- [Documentation](../../docs/)
- [HTTP API Reference](../../docs/http-api.md)
- [Java Client API Reference](../../docs/java-client.md)

## Support

- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs
