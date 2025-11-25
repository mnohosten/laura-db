# LauraDB GraphQL API

Complete guide to using LauraDB's GraphQL API for queries and mutations.

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
- [GraphQL Schema](#graphql-schema)
- [Queries](#queries)
- [Mutations](#mutations)
- [Subscriptions](#subscriptions)
- [Examples](#examples)
- [GraphiQL Playground](#graphiql-playground)
- [Error Handling](#error-handling)

---

## Overview

LauraDB provides a **GraphQL API** as an alternative to the REST API. GraphQL offers:

- **Single endpoint** (`/graphql`) for all operations
- **Flexible queries** - request only the data you need
- **Strong typing** - schema validation at query time
- **Real-time subscriptions** - watch for collection changes
- **Interactive playground** (GraphiQL) for testing and exploration

## Getting Started

### Enable GraphQL

Start the LauraDB server with GraphQL enabled:

```bash
./bin/laura-server -graphql
```

The following endpoints will be available:

- **GraphQL API**: `http://localhost:8080/graphql`
- **GraphiQL Playground**: `http://localhost:8080/graphiql`

### Basic Query

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ listCollections }"
  }'
```

Response:

```json
{
  "data": {
    "listCollections": ["users", "products", "orders"]
  }
}
```

---

## GraphQL Schema

### Custom Scalars

#### `JSON`

The `JSON` scalar type represents arbitrary JSON data.

```graphql
scalar JSON
```

### Object Types

#### `Document`

Represents a document in LauraDB.

```graphql
type Document {
  _id: String!        # Unique document identifier
  data: JSON!         # Document data as JSON
}
```

#### `InsertResult`

Result of an insert operation.

```graphql
type InsertResult {
  insertedId: String!
}
```

#### `InsertManyResult`

Result of an insertMany operation.

```graphql
type InsertManyResult {
  insertedIds: [String!]
  insertedCount: Int!
}
```

#### `UpdateResult`

Result of an update operation.

```graphql
type UpdateResult {
  matchedCount: Int!
  modifiedCount: Int!
}
```

#### `DeleteResult`

Result of a delete operation.

```graphql
type DeleteResult {
  deletedCount: Int!
}
```

#### `IndexInfo`

Information about an index.

```graphql
type IndexInfo {
  name: String!
  field: String!
  unique: Boolean!
  type: String!
}
```

#### `CollectionStats`

Statistics about a collection.

```graphql
type CollectionStats {
  name: String!
  documentCount: Int!
  indexCount: Int!
}
```

#### `AggregationResult`

Result of an aggregation operation.

```graphql
type AggregationResult {
  results: [JSON]
}
```

---

## Queries

### `findOne`

Find a single document.

```graphql
query {
  findOne(
    collection: String!
    filter: JSON
  ): Document
}
```

**Example**:

```graphql
query {
  findOne(collection: "users", filter: {name: "Alice"}) {
    _id
    data
  }
}
```

### `find`

Find multiple documents.

```graphql
query {
  find(
    collection: String!
    filter: JSON
    sort: JSON
    limit: Int
    skip: Int
  ): [Document]
}
```

**Example**:

```graphql
query FindUsers($collection: String!, $filter: JSON) {
  find(collection: $collection, filter: $filter, limit: 10) {
    _id
    data
  }
}
```

Variables:

```json
{
  "collection": "users",
  "filter": {
    "age": {"$gte": 18}
  }
}
```

### `count`

Count documents matching a filter.

```graphql
query {
  count(
    collection: String!
    filter: JSON
  ): Int!
}
```

**Example**:

```graphql
query {
  count(collection: "users", filter: {status: "active"})
}
```

### `listCollections`

List all collections.

```graphql
query {
  listCollections: [String!]
}
```

**Example**:

```graphql
query {
  listCollections
}
```

### `collectionStats`

Get collection statistics.

```graphql
query {
  collectionStats(
    collection: String!
  ): CollectionStats
}
```

**Example**:

```graphql
query {
  collectionStats(collection: "users") {
    name
    documentCount
    indexCount
  }
}
```

### `listIndexes`

List all indexes in a collection.

```graphql
query {
  listIndexes(
    collection: String!
  ): [IndexInfo]
}
```

**Example**:

```graphql
query {
  listIndexes(collection: "users") {
    name
    field
    unique
    type
  }
}
```

### `aggregate`

Run an aggregation pipeline.

```graphql
query {
  aggregate(
    collection: String!
    pipeline: [JSON]
  ): AggregationResult
}
```

**Example**:

```graphql
query AggregateOrders($collection: String!, $pipeline: [JSON]) {
  aggregate(collection: $collection, pipeline: $pipeline) {
    results
  }
}
```

Variables:

```json
{
  "collection": "orders",
  "pipeline": [
    {"$group": {"_id": "$product", "total": {"$sum": "$quantity"}}}
  ]
}
```

---

## Mutations

### `createCollection`

Create a new collection.

```graphql
mutation {
  createCollection(
    name: String!
  ): Boolean!
}
```

**Example**:

```graphql
mutation {
  createCollection(name: "users")
}
```

### `dropCollection`

Drop a collection.

```graphql
mutation {
  dropCollection(
    name: String!
  ): Boolean!
}
```

**Example**:

```graphql
mutation {
  dropCollection(name: "temp_data")
}
```

### `insertOne`

Insert a single document.

```graphql
mutation {
  insertOne(
    collection: String!
    document: JSON!
  ): InsertResult
}
```

**Example**:

```graphql
mutation InsertUser($collection: String!, $document: JSON!) {
  insertOne(collection: $collection, document: $document) {
    insertedId
  }
}
```

Variables:

```json
{
  "collection": "users",
  "document": {
    "name": "Alice",
    "email": "alice@example.com",
    "age": 30
  }
}
```

### `insertMany`

Insert multiple documents.

```graphql
mutation {
  insertMany(
    collection: String!
    documents: [JSON!]
  ): InsertManyResult
}
```

**Example**:

```graphql
mutation InsertUsers($collection: String!, $documents: [JSON!]) {
  insertMany(collection: $collection, documents: $documents) {
    insertedIds
    insertedCount
  }
}
```

Variables:

```json
{
  "collection": "users",
  "documents": [
    {"name": "Bob", "age": 25},
    {"name": "Charlie", "age": 35}
  ]
}
```

### `updateOne`

Update a single document.

```graphql
mutation {
  updateOne(
    collection: String!
    filter: JSON!
    update: JSON!
  ): UpdateResult
}
```

**Example**:

```graphql
mutation UpdateUser($collection: String!, $filter: JSON!, $update: JSON!) {
  updateOne(collection: $collection, filter: $filter, update: $update) {
    matchedCount
    modifiedCount
  }
}
```

Variables:

```json
{
  "collection": "users",
  "filter": {"name": "Alice"},
  "update": {"$set": {"age": 31}}
}
```

### `updateMany`

Update multiple documents.

```graphql
mutation {
  updateMany(
    collection: String!
    filter: JSON!
    update: JSON!
  ): UpdateResult
}
```

**Example**:

```graphql
mutation {
  updateMany(
    collection: "users"
    filter: {status: "inactive"}
    update: {$set: {status: "archived"}}
  ) {
    matchedCount
    modifiedCount
  }
}
```

### `deleteOne`

Delete a single document.

```graphql
mutation {
  deleteOne(
    collection: String!
    filter: JSON!
  ): DeleteResult
}
```

**Example**:

```graphql
mutation DeleteUser($collection: String!, $filter: JSON!) {
  deleteOne(collection: $collection, filter: $filter) {
    deletedCount
  }
}
```

Variables:

```json
{
  "collection": "users",
  "filter": {"_id": "507f1f77bcf86cd799439011"}
}
```

### `deleteMany`

Delete multiple documents.

```graphql
mutation {
  deleteMany(
    collection: String!
    filter: JSON!
  ): DeleteResult
}
```

**Example**:

```graphql
mutation {
  deleteMany(
    collection: "users"
    filter: {status: "deleted"}
  ) {
    deletedCount
  }
}
```

### `createIndex`

Create an index.

```graphql
mutation {
  createIndex(
    collection: String!
    field: String!
    unique: Boolean
    name: String
  ): Boolean!
}
```

**Example**:

```graphql
mutation {
  createIndex(
    collection: "users"
    field: "email"
    unique: true
  )
}
```

### `dropIndex`

Drop an index.

```graphql
mutation {
  dropIndex(
    collection: String!
    name: String!
  ): Boolean!
}
```

**Example**:

```graphql
mutation {
  dropIndex(
    collection: "users"
    name: "email_1"
  )
}
```

---

## Subscriptions

### `watchCollection`

Watch for changes in a collection (real-time updates).

```graphql
subscription {
  watchCollection(
    collection: String!
    filter: JSON
  ): Document
}
```

**Example**:

```graphql
subscription {
  watchCollection(collection: "orders", filter: {status: "pending"}) {
    _id
    data
  }
}
```

**Note**: Subscriptions require WebSocket support. Use a GraphQL client library like Apollo Client or Relay.

---

## Examples

### Complete CRUD Workflow

```graphql
# 1. Create collection
mutation {
  createCollection(name: "products")
}

# 2. Insert products
mutation {
  insertMany(
    collection: "products"
    documents: [
      {name: "Laptop", price: 999, category: "electronics"},
      {name: "Mouse", price: 25, category: "electronics"},
      {name: "Desk", price: 299, category: "furniture"}
    ]
  ) {
    insertedCount
  }
}

# 3. Create index
mutation {
  createIndex(collection: "products", field: "category")
}

# 4. Query products
query {
  find(collection: "products", filter: {category: "electronics"}) {
    _id
    data
  }
}

# 5. Update product
mutation {
  updateOne(
    collection: "products"
    filter: {name: "Laptop"}
    update: {$set: {price: 899}}
  ) {
    modifiedCount
  }
}

# 6. Delete product
mutation {
  deleteOne(collection: "products", filter: {name: "Mouse"}) {
    deletedCount
  }
}

# 7. Get stats
query {
  collectionStats(collection: "products") {
    documentCount
    indexCount
  }
}
```

### Aggregation Example

```graphql
query OrdersByProduct {
  aggregate(
    collection: "orders"
    pipeline: [
      {$match: {status: "completed"}},
      {$group: {_id: "$productId", totalSales: {$sum: "$amount"}}},
      {$sort: {totalSales: -1}},
      {$limit: 10}
    ]
  ) {
    results
  }
}
```

### Using Variables

```graphql
query FindDocuments($coll: String!, $minAge: Int!) {
  find(
    collection: $coll
    filter: {age: {$gte: $minAge}}
    limit: 100
  ) {
    _id
    data
  }
}
```

Variables:

```json
{
  "coll": "users",
  "minAge": 18
}
```

---

## GraphiQL Playground

GraphiQL is an interactive GraphQL IDE available at `/graphiql`.

### Features

- **Auto-completion**: Press `Ctrl+Space` for field suggestions
- **Documentation**: Click "Docs" to explore the schema
- **Query history**: Access previous queries
- **Variables editor**: Test queries with variables
- **Prettify**: Format queries with `Shift+Ctrl+P`

### Quick Start

1. Open `http://localhost:8080/graphiql`
2. Try this query:

```graphql
{
  listCollections
}
```

3. Click the "Play" button (or press `Ctrl+Enter`)
4. Explore the schema using the "Docs" panel

---

## Error Handling

GraphQL errors are returned in the `errors` array:

```json
{
  "errors": [
    {
      "message": "collection not found: invalid_collection",
      "path": ["findOne"]
    }
  ],
  "data": null
}
```

### Common Errors

#### Collection Not Found

```json
{
  "errors": [
    {
      "message": "collection not found: users"
    }
  ]
}
```

**Solution**: Create the collection first with `createCollection`.

#### Invalid Filter

```json
{
  "errors": [
    {
      "message": "invalid filter format"
    }
  ]
}
```

**Solution**: Ensure filter is valid JSON and uses correct MongoDB operators.

#### Document Not Found

```json
{
  "data": {
    "findOne": null
  }
}
```

**Note**: Returns `null` instead of an error when no document matches.

---

## Best Practices

### 1. Use Variables

Always use variables for dynamic values:

```graphql
# Good
query FindUsers($filter: JSON!) {
  find(collection: "users", filter: $filter) {
    _id
    data
  }
}

# Avoid
query {
  find(collection: "users", filter: {name: "Alice"}) {
    _id
    data
  }
}
```

### 2. Request Only Needed Fields

```graphql
# Good - only request _id
query {
  find(collection: "users") {
    _id
  }
}

# Avoid - requesting all data
query {
  find(collection: "users") {
    _id
    data
  }
}
```

### 3. Use Aliases

```graphql
query {
  activeUsers: find(collection: "users", filter: {status: "active"}) {
    _id
  }
  inactiveUsers: find(collection: "users", filter: {status: "inactive"}) {
    _id
  }
}
```

### 4. Use Fragments

```graphql
fragment UserFields on Document {
  _id
  data
}

query {
  activeUsers: find(collection: "users", filter: {status: "active"}) {
    ...UserFields
  }
}
```

### 5. Handle Errors Gracefully

```javascript
const result = await fetch('/graphql', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({query, variables})
});

const json = await result.json();

if (json.errors) {
  console.error('GraphQL errors:', json.errors);
}
```

---

## GraphQL vs REST

| Feature | GraphQL | REST |
|---------|---------|------|
| **Endpoints** | Single (`/graphql`) | Multiple (per resource) |
| **Data Fetching** | Exact fields requested | Fixed response structure |
| **Over-fetching** | No | Yes (fetch unused data) |
| **Under-fetching** | No | Yes (multiple requests) |
| **Versioning** | Schema evolution | URL versioning |
| **Real-time** | Subscriptions | WebSocket or polling |
| **Learning Curve** | Steeper | Gentler |

---

## Client Libraries

### JavaScript (Node.js / Browser)

```bash
npm install graphql-request
```

```javascript
import { request, gql } from 'graphql-request'

const query = gql`
  query {
    listCollections
  }
`

const data = await request('http://localhost:8080/graphql', query)
console.log(data.listCollections)
```

### Python

```bash
pip install gql[all]
```

```python
from gql import gql, Client
from gql.transport.requests import RequestsHTTPTransport

transport = RequestsHTTPTransport(url='http://localhost:8080/graphql')
client = Client(transport=transport)

query = gql("""
  query {
    listCollections
  }
""")

result = client.execute(query)
print(result['listCollections'])
```

### Go

```bash
go get github.com/machinebox/graphql
```

```go
package main

import (
    "context"
    "github.com/machinebox/graphql"
)

func main() {
    client := graphql.NewClient("http://localhost:8080/graphql")

    req := graphql.NewRequest(`
        query {
            listCollections
        }
    `)

    var resp struct {
        ListCollections []string
    }

    if err := client.Run(context.Background(), req, &resp); err != nil {
        panic(err)
    }

    println(resp.ListCollections)
}
```

---

## See Also

- [HTTP API Reference](http-api.md) - REST API documentation
- [Query Language](query-engine.md) - MongoDB-style query operators
- [Aggregation Pipeline](aggregation.md) - Aggregation framework
- [Indexing](indexing.md) - Index types and performance

---

**Version**: 1.0
**Last Updated**: 2025-01-15
