# LauraDB Java Client API Reference

Complete API reference for the LauraDB Java client library.

## Table of Contents

- [Installation](#installation)
- [Client](#client)
- [Collection](#collection)
- [Query](#query)
- [FindOptions](#findoptions)
- [Aggregation](#aggregation)
- [IndexBuilder](#indexbuilder)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Installation

### Maven

```xml
<dependency>
    <groupId>com.lauradb</groupId>
    <artifactId>lauradb-client</artifactId>
    <version>1.0.0</version>
</dependency>
```

### Gradle

```gradle
implementation 'com.lauradb:lauradb-client:1.0.0'
```

## Client

### LauraDBClient

Main client class for connecting to LauraDB server.

#### Constructor (Builder Pattern)

```java
LauraDBClient client = LauraDBClient.builder()
        .host("localhost")      // Server hostname (default: "localhost")
        .port(8080)            // Server port (default: 8080)
        .https(false)          // Use HTTPS (default: false)
        .timeout(30000)        // Request timeout in ms (default: 30000)
        .build();
```

#### Methods

##### `boolean ping()`

Check if the server is healthy.

```java
boolean healthy = client.ping();
```

##### `CompletableFuture<Boolean> pingAsync()`

Asynchronously check server health.

```java
CompletableFuture<Boolean> future = client.pingAsync();
Boolean healthy = future.get();
```

##### `Map<String, Object> stats()`

Get database statistics.

```java
Map<String, Object> stats = client.stats();
```

**Throws**: `IOException` if the request fails

##### `List<String> listCollections()`

List all collections in the database.

```java
List<String> collections = client.listCollections();
```

**Throws**: `IOException` if the request fails

##### `void createCollection(String name)`

Create a new collection.

```java
client.createCollection("users");
```

**Throws**: `IOException` if the request fails

##### `void dropCollection(String name)`

Drop a collection and all its data.

```java
client.dropCollection("users");
```

**Throws**: `IOException` if the request fails

##### `Collection collection(String name)`

Get a collection by name.

```java
Collection users = client.collection("users");
```

##### `void close()`

Close the client and release resources. Implements `AutoCloseable`.

```java
client.close();
// Or use try-with-resources:
try (LauraDBClient client = LauraDBClient.builder().build()) {
    // Your code
}
```

## Collection

Represents a collection in LauraDB.

### Insert Operations

#### `String insertOne(Map<String, Object> document)`

Insert a single document.

```java
Map<String, Object> user = new HashMap<>();
user.put("name", "Alice");
user.put("age", 30);

String id = collection.insertOne(user);
```

**Returns**: The inserted document's ID
**Throws**: `IOException` if the request fails

#### `List<String> insertMany(List<Map<String, Object>> documents)`

Insert multiple documents.

```java
List<Map<String, Object>> users = new ArrayList<>();
users.add(Map.of("name", "Alice", "age", 30));
users.add(Map.of("name", "Bob", "age", 25));

List<String> ids = collection.insertMany(users);
```

**Returns**: List of inserted document IDs
**Throws**: `IOException` if the request fails

#### `CompletableFuture<String> insertOneAsync(Map<String, Object> document)`

Asynchronously insert a document.

```java
CompletableFuture<String> future = collection.insertOneAsync(document);
String id = future.get();
```

### Find Operations

#### `Map<String, Object> findById(String id)`

Find a document by its ID.

```java
Map<String, Object> user = collection.findById("507f1f77bcf86cd799439011");
```

**Returns**: The document, or `null` if not found
**Throws**: `IOException` if the request fails

#### `List<Map<String, Object>> find(Query query)`

Find documents matching a query.

```java
Query query = Query.builder()
        .gte("age", 25)
        .build();

List<Map<String, Object>> results = collection.find(query);
```

**Returns**: List of matching documents
**Throws**: `IOException` if the request fails

#### `List<Map<String, Object>> find(Query query, FindOptions options)`

Find documents with options (projection, sort, limit, skip).

```java
FindOptions options = new FindOptions()
        .projection(Map.of("name", 1, "email", 1))
        .sort(Map.of("age", -1))
        .limit(10)
        .skip(0);

List<Map<String, Object>> results = collection.find(query, options);
```

**Returns**: List of matching documents
**Throws**: `IOException` if the request fails

#### `List<Map<String, Object>> findAll()`

Find all documents in the collection.

```java
List<Map<String, Object>> allDocs = collection.findAll();
```

**Returns**: List of all documents
**Throws**: `IOException` if the request fails

#### `Map<String, Object> findOne(Query query)`

Find the first document matching a query.

```java
Map<String, Object> user = collection.findOne(
    Query.builder().eq("email", "alice@example.com").build()
);
```

**Returns**: The first matching document, or `null` if none found
**Throws**: `IOException` if the request fails

#### `CompletableFuture<List<Map<String, Object>>> findAsync(Query query)`

Asynchronously find documents.

```java
CompletableFuture<List<Map<String, Object>>> future = collection.findAsync(query);
List<Map<String, Object>> results = future.get();
```

#### `long count(Query query)`

Count documents matching a query.

```java
long count = collection.count(Query.builder().eq("active", true).build());
```

**Returns**: The count of matching documents
**Throws**: `IOException` if the request fails

### Update Operations

#### `long updateOne(Query query, Map<String, Object> update)`

Update a single document.

```java
Map<String, Object> update = new HashMap<>();
update.put("$set", Map.of("age", 31, "status", "active"));
update.put("$inc", Map.of("loginCount", 1));

long modified = collection.updateOne(
    Query.builder().eq("name", "Alice").build(),
    update
);
```

**Returns**: Number of modified documents (0 or 1)
**Throws**: `IOException` if the request fails

#### `long updateById(String id, Map<String, Object> update)`

Update a document by its ID.

```java
long modified = collection.updateById(
    "507f1f77bcf86cd799439011",
    Map.of("$set", Map.of("status", "verified"))
);
```

**Returns**: Number of modified documents
**Throws**: `IOException` if the request fails

#### `long updateMany(Query query, Map<String, Object> update)`

Update multiple documents.

```java
long modified = collection.updateMany(
    Query.builder().eq("status", "pending").build(),
    Map.of("$set", Map.of("status", "processed"))
);
```

**Returns**: Number of modified documents
**Throws**: `IOException` if the request fails

### Delete Operations

#### `long deleteOne(Query query)`

Delete a single document.

```java
long deleted = collection.deleteOne(
    Query.builder().eq("email", "old@example.com").build()
);
```

**Returns**: Number of deleted documents (0 or 1)
**Throws**: `IOException` if the request fails

#### `long deleteById(String id)`

Delete a document by its ID.

```java
long deleted = collection.deleteById("507f1f77bcf86cd799439011");
```

**Returns**: Number of deleted documents
**Throws**: `IOException` if the request fails

#### `long deleteMany(Query query)`

Delete multiple documents.

```java
long deleted = collection.deleteMany(
    Query.builder().eq("status", "archived").build()
);
```

**Returns**: Number of deleted documents
**Throws**: `IOException` if the request fails

### Aggregation Operations

#### `List<Map<String, Object>> aggregate(Aggregation pipeline)`

Execute an aggregation pipeline.

```java
Aggregation pipeline = Aggregation.builder()
        .match(Query.builder().gte("age", 18).build())
        .group("$city")
            .avg("avgAge", "$age")
            .count("count")
        .sort(Map.of("avgAge", -1))
        .build();

List<Map<String, Object>> results = collection.aggregate(pipeline);
```

**Returns**: List of aggregation results
**Throws**: `IOException` if the request fails

### Index Operations

#### `IndexBuilder createIndex(String field)`

Create an index on a single field.

```java
collection.createIndex("email")
        .unique(true)
        .build();
```

**Returns**: An `IndexBuilder` for further configuration

#### `IndexBuilder createCompoundIndex(List<String> fields)`

Create a compound index on multiple fields.

```java
collection.createCompoundIndex(List.of("city", "age"))
        .name("city_age_idx")
        .build();
```

**Returns**: An `IndexBuilder` for further configuration

#### `List<String> listIndexes()`

List all indexes in the collection.

```java
List<String> indexes = collection.listIndexes();
```

**Returns**: List of index names
**Throws**: `IOException` if the request fails

#### `void dropIndex(String field)`

Drop an index.

```java
collection.dropIndex("email");
```

**Throws**: `IOException` if the request fails

## Query

Builder for constructing MongoDB-style queries.

### Creating Queries

#### `static Builder builder()`

Create a new query builder.

```java
Query query = Query.builder()
        .eq("name", "Alice")
        .gte("age", 25)
        .build();
```

#### `static Query empty()`

Create an empty query (matches all documents).

```java
Query query = Query.empty();
```

### Comparison Operators

#### `Builder eq(String field, Object value)`

Equality condition.

```java
Query.builder().eq("status", "active").build()
```

#### `Builder ne(String field, Object value)`

Not-equals condition.

```java
Query.builder().ne("status", "deleted").build()
```

#### `Builder gt(String field, Object value)`

Greater-than condition.

```java
Query.builder().gt("age", 25).build()
```

#### `Builder gte(String field, Object value)`

Greater-than-or-equal condition.

```java
Query.builder().gte("age", 18).build()
```

#### `Builder lt(String field, Object value)`

Less-than condition.

```java
Query.builder().lt("price", 100).build()
```

#### `Builder lte(String field, Object value)`

Less-than-or-equal condition.

```java
Query.builder().lte("quantity", 50).build()
```

#### `Builder in(String field, Object... values)`

Value in array condition.

```java
Query.builder().in("category", "A", "B", "C").build()
```

#### `Builder nin(String field, Object... values)`

Value not in array condition.

```java
Query.builder().nin("status", "deleted", "archived").build()
```

### Logical Operators

#### `Builder and(Query... queries)`

AND condition (all queries must match).

```java
Query.builder()
        .and(
            Query.builder().gte("age", 25).build(),
            Query.builder().eq("active", true).build()
        )
        .build()
```

#### `Builder or(Query... queries)`

OR condition (any query must match).

```java
Query.builder()
        .or(
            Query.builder().eq("city", "New York").build(),
            Query.builder().eq("city", "Boston").build()
        )
        .build()
```

#### `Builder not(Query query)`

NOT condition (negates the query).

```java
Query.builder()
        .not(Query.builder().eq("status", "deleted").build())
        .build()
```

### Element Operators

#### `Builder exists(String field, boolean exists)`

Field existence condition.

```java
Query.builder().exists("email", true).build()
```

#### `Builder type(String field, String type)`

Field type condition.

Valid types: `"string"`, `"number"`, `"boolean"`, `"array"`, `"document"`, `"null"`

```java
Query.builder().type("age", "number").build()
```

### Array Operators

#### `Builder all(String field, Object... values)`

Array contains all values.

```java
Query.builder().all("tags", "java", "database").build()
```

#### `Builder size(String field, int size)`

Array size condition.

```java
Query.builder().size("items", 5).build()
```

#### `Builder elemMatch(String field, Query query)`

Array element matching condition.

```java
Query elementQuery = Query.builder().gte("score", 80).build();
Query.builder().elemMatch("grades", elementQuery).build()
```

### Evaluation Operators

#### `Builder regex(String field, String pattern)`

Regular expression condition.

```java
Query.builder().regex("email", ".*@example\\.com$").build()
```

### Building the Query

#### `Query build()`

Build the query.

```java
Query query = Query.builder()
        .gte("age", 25)
        .lt("age", 40)
        .build();
```

## FindOptions

Options for find operations.

### Methods

#### `FindOptions projection(Map<String, Object> projection)`

Set field projection (1 = include, 0 = exclude).

```java
new FindOptions().projection(Map.of(
    "name", 1,
    "email", 1,
    "_id", 0
))
```

#### `FindOptions sort(Map<String, Integer> sort)`

Set sort order (1 = ascending, -1 = descending).

```java
new FindOptions().sort(Map.of(
    "age", -1,
    "name", 1
))
```

#### `FindOptions limit(int limit)`

Set maximum number of documents to return.

```java
new FindOptions().limit(10)
```

#### `FindOptions skip(int skip)`

Set number of documents to skip.

```java
new FindOptions().skip(20)
```

### Builder Methods

#### `static ProjectionBuilder projectionBuilder()`

Create a projection builder.

```java
FindOptions.projectionBuilder()
        .include("name")
        .include("email")
        .exclude("_id")
        .build()
```

#### `static SortBuilder sortBuilder()`

Create a sort builder.

```java
FindOptions.sortBuilder()
        .descending("age")
        .ascending("name")
        .build()
```

## Aggregation

Builder for aggregation pipelines.

### Creating Pipelines

#### `static Builder builder()`

Create a new aggregation builder.

```java
Aggregation pipeline = Aggregation.builder()
        .match(Query.builder().gte("age", 18).build())
        .group("$city")
            .avg("avgAge", "$age")
            .count("count")
        .sort(Map.of("avgAge", -1))
        .build();
```

### Pipeline Stages

#### `Builder match(Query query)`

Add a $match stage (filter documents).

```java
Aggregation.builder()
        .match(Query.builder().gte("age", 18).build())
```

#### `GroupBuilder group(String groupBy)`

Add a $group stage (group documents).

```java
Aggregation.builder()
        .group("$city")
            .avg("avgAge", "$age")
            .count("count")
        .build()
```

#### `Builder project(Map<String, Object> projection)`

Add a $project stage (field selection/transformation).

```java
Aggregation.builder()
        .project(Map.of(
            "name", 1,
            "email", 1,
            "_id", 0
        ))
```

#### `Builder sort(Map<String, Integer> sort)`

Add a $sort stage.

```java
Aggregation.builder()
        .sort(Map.of("age", -1))
```

#### `Builder limit(int limit)`

Add a $limit stage.

```java
Aggregation.builder()
        .limit(10)
```

#### `Builder skip(int skip)`

Add a $skip stage.

```java
Aggregation.builder()
        .skip(20)
```

### Group Accumulators

Available on `GroupBuilder`:

#### `GroupBuilder sum(String field, Object expression)`

Sum aggregation.

```java
.group("$category")
    .sum("totalAmount", "$amount")  // Sum amounts
    .sum("count", 1)                 // Count documents
```

#### `GroupBuilder avg(String field, String expression)`

Average aggregation.

```java
.group("$city")
    .avg("avgAge", "$age")
```

#### `GroupBuilder min(String field, String expression)`

Minimum aggregation.

```java
.group("$category")
    .min("minPrice", "$price")
```

#### `GroupBuilder max(String field, String expression)`

Maximum aggregation.

```java
.group("$category")
    .max("maxPrice", "$price")
```

#### `GroupBuilder count(String field)`

Count documents in each group.

```java
.group("$status")
    .count("total")
```

#### `GroupBuilder push(String field, String expression)`

Collect values into an array.

```java
.group("$city")
    .push("names", "$name")
```

## IndexBuilder

Builder for creating indexes.

### Methods

#### `IndexBuilder name(String name)`

Set the index name.

```java
collection.createIndex("email")
        .name("email_unique_idx")
```

#### `IndexBuilder unique(boolean unique)`

Make the index unique.

```java
collection.createIndex("email")
        .unique(true)
```

#### `IndexBuilder text()`

Create a text index for full-text search.

```java
collection.createIndex("content")
        .text()
```

#### `IndexBuilder geo(String geoType)`

Create a geospatial index.

Valid types: `"2d"` (planar), `"2dsphere"` (spherical)

```java
collection.createIndex("location")
        .geo("2dsphere")
```

#### `IndexBuilder ttl(int seconds)`

Create a TTL index for automatic document expiration.

```java
collection.createIndex("createdAt")
        .ttl(3600)  // Expire after 1 hour
```

#### `IndexBuilder partial(Query filter)`

Create a partial index (only index matching documents).

```java
collection.createIndex("email")
        .unique(true)
        .partial(Query.builder().eq("active", true).build())
```

#### `IndexBuilder background(boolean background)`

Build the index in the background (non-blocking).

```java
collection.createIndex("age")
        .background(true)
```

#### `void build()`

Create the index.

```java
collection.createIndex("email")
        .unique(true)
        .build();
```

**Throws**: `IOException` if the request fails

## Error Handling

All I/O operations throw `IOException` on failure. Use try-catch blocks:

```java
try {
    String id = collection.insertOne(document);
    System.out.println("Inserted: " + id);
} catch (IOException e) {
    System.err.println("Failed to insert: " + e.getMessage());
    // Handle error
}
```

For async operations, exceptions are wrapped in `ExecutionException`:

```java
try {
    String id = collection.insertOneAsync(document).get();
} catch (ExecutionException e) {
    Throwable cause = e.getCause();
    if (cause instanceof IOException) {
        System.err.println("IO error: " + cause.getMessage());
    }
} catch (InterruptedException e) {
    Thread.currentThread().interrupt();
}
```

## Best Practices

### 1. Use Try-With-Resources

```java
try (LauraDBClient client = LauraDBClient.builder().build()) {
    // Client automatically closed
}
```

### 2. Reuse Client Instances

```java
// Good
LauraDBClient client = LauraDBClient.builder().build();
Collection users = client.collection("users");
Collection posts = client.collection("posts");
```

### 3. Use Appropriate Indexes

```java
// Index frequently queried fields
collection.createIndex("email").unique(true).build();
collection.createCompoundIndex(List.of("city", "age")).build();
```

### 4. Use Projections

```java
// Only fetch needed fields
FindOptions options = new FindOptions()
        .projection(Map.of("name", 1, "email", 1));
```

### 5. Implement Pagination

```java
int page = 1;
int pageSize = 20;

FindOptions options = new FindOptions()
        .skip((page - 1) * pageSize)
        .limit(pageSize);
```

### 6. Bulk Operations

```java
// More efficient than multiple insertOne() calls
List<Map<String, Object>> docs = new ArrayList<>();
// ... add documents
collection.insertMany(docs);
```

## Complete Example

```java
import com.lauradb.client.*;
import java.util.*;

public class Example {
    public static void main(String[] args) {
        try (LauraDBClient client = LauraDBClient.builder()
                .host("localhost")
                .port(8080)
                .build()) {

            // Get collection
            Collection users = client.collection("users");

            // Insert documents
            Map<String, Object> user = new HashMap<>();
            user.put("name", "Alice");
            user.put("email", "alice@example.com");
            user.put("age", 30);
            user.put("city", "New York");
            user.put("active", true);

            String id = users.insertOne(user);

            // Create index
            users.createIndex("email").unique(true).build();

            // Query with options
            Query query = Query.builder()
                    .gte("age", 25)
                    .eq("active", true)
                    .build();

            FindOptions options = new FindOptions()
                    .projection(Map.of("name", 1, "email", 1))
                    .sort(Map.of("age", -1))
                    .limit(10);

            List<Map<String, Object>> results = users.find(query, options);

            // Aggregation
            Aggregation pipeline = Aggregation.builder()
                    .match(Query.builder().eq("active", true).build())
                    .group("$city")
                        .avg("avgAge", "$age")
                        .count("count")
                    .sort(Map.of("avgAge", -1))
                    .build();

            List<Map<String, Object>> aggResults = users.aggregate(pipeline);

            // Update
            Map<String, Object> update = new HashMap<>();
            update.put("$set", Map.of("lastLogin", new Date().getTime()));
            users.updateOne(Query.builder().eq("_id", id).build(), update);

        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
```

## See Also

- [LauraDB Documentation](https://github.com/mnohosten/laura-db/tree/main/docs)
- [HTTP API Reference](http-api.md)
- [Client README](../clients/java/README.md)
- [Examples](../clients/java/examples/)
