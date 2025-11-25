# LauraDB Python Client Documentation

Complete documentation for the LauraDB Python client library.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Client API](#client-api)
4. [Collection API](#collection-api)
5. [Query Building](#query-building)
6. [Aggregation Pipeline](#aggregation-pipeline)
7. [Index Management](#index-management)
8. [Update Operators](#update-operators)
9. [Error Handling](#error-handling)
10. [Best Practices](#best-practices)
11. [Examples](#examples)

---

## Installation

### Using pip

```bash
pip install lauradb-client
```

### From source

```bash
cd clients/python
pip install -e .
```

### Development installation

```bash
cd clients/python
pip install -e ".[dev]"
```

## Quick Start

```python
from lauradb import Client

# Connect to LauraDB server
client = Client(host='localhost', port=8080)

# Get a collection
users = client.collection('users')

# Insert a document
user_id = users.insert_one({
    'name': 'Alice',
    'email': 'alice@example.com',
    'age': 30
})

# Find documents
results = users.find({'age': {'$gte': 25}})

# Update documents
users.update_one(
    {'name': 'Alice'},
    {'$set': {'age': 31}}
)

# Delete documents
users.delete_one({'name': 'Alice'})

# Close connection
client.close()
```

---

## Client API

### Creating a Client

```python
from lauradb import Client

# Default configuration (localhost:8080)
client = Client()

# Custom configuration
client = Client(
    host='db.example.com',
    port=8080,
    https=False,
    timeout=30,              # seconds
    max_connections=10       # connection pool size
)

# Using context manager (recommended)
with Client(host='localhost', port=8080) as client:
    users = client.collection('users')
    # ... perform operations
# Connection automatically closed
```

### Client Methods

#### `ping() -> bool`
Check if server is reachable.

```python
if client.ping():
    print("Server is reachable")
```

#### `stats() -> Dict[str, Any]`
Get database statistics.

```python
stats = client.stats()
print(f"Collections: {stats.get('collections', [])}")
```

#### `list_collections() -> List[str]`
List all collections.

```python
collections = client.list_collections()
for name in collections:
    print(name)
```

#### `create_collection(name: str) -> bool`
Create a new collection.

```python
client.create_collection('users')
```

#### `drop_collection(name: str) -> bool`
Delete a collection.

```python
client.drop_collection('users')
```

#### `collection(name: str) -> Collection`
Get a collection object.

```python
users = client.collection('users')
```

#### `close()`
Close the client and release resources.

```python
client.close()
```

---

## Collection API

### Insert Operations

#### `insert_one(document: Dict[str, Any]) -> str`
Insert a single document.

```python
doc_id = users.insert_one({
    'name': 'Alice',
    'email': 'alice@example.com',
    'age': 30,
    'tags': ['python', 'golang']
})
print(f"Inserted ID: {doc_id}")
```

#### `insert_many(documents: List[Dict[str, Any]]) -> List[str]`
Insert multiple documents.

```python
docs = [
    {'name': 'Bob', 'age': 25},
    {'name': 'Charlie', 'age': 35}
]
ids = users.insert_many(docs)
print(f"Inserted {len(ids)} documents")
```

### Find Operations

#### `find_one(filter: Dict, projection: Dict = None) -> Optional[Dict]`
Find a single document.

```python
# Find by exact match
user = users.find_one({'name': 'Alice'})

# With projection
user = users.find_one(
    {'name': 'Alice'},
    projection={'name': 1, 'email': 1, '_id': 0}
)
```

#### `find(filter: Dict, projection: Dict = None, sort: Dict = None, skip: int = None, limit: int = None) -> List[Dict]`
Find multiple documents.

```python
# Find all
all_users = users.find()

# With filter
adults = users.find({'age': {'$gte': 18}})

# With multiple options
results = users.find(
    filter={'age': {'$gte': 25}},
    projection={'name': 1, 'age': 1},
    sort={'age': -1},
    skip=10,
    limit=20
)
```

#### `count(filter: Dict = None) -> int`
Count documents.

```python
# Count all
total = users.count()

# Count with filter
adults = users.count({'age': {'$gte': 18}})
```

### Update Operations

#### `update_one(filter: Dict, update: Dict) -> bool`
Update a single document.

```python
result = users.update_one(
    {'name': 'Alice'},
    {'$set': {'age': 31, 'city': 'Boston'}}
)
if result:
    print("Updated successfully")
```

#### `update_many(filter: Dict, update: Dict) -> int`
Update multiple documents.

```python
count = users.update_many(
    {'city': 'New York'},
    {'$set': {'timezone': 'EST'}}
)
print(f"Updated {count} documents")
```

### Delete Operations

#### `delete_one(filter: Dict) -> bool`
Delete a single document.

```python
result = users.delete_one({'name': 'Alice'})
if result:
    print("Deleted successfully")
```

#### `delete_many(filter: Dict) -> int`
Delete multiple documents.

```python
count = users.delete_many({'age': {'$lt': 18}})
print(f"Deleted {count} documents")
```

### Aggregation

#### `aggregate(pipeline: List[Dict]) -> List[Dict]`
Execute an aggregation pipeline.

```python
pipeline = [
    {'$match': {'age': {'$gte': 18}}},
    {'$group': {
        '_id': '$city',
        'avgAge': {'$avg': '$age'},
        'count': {'$count': {}}
    }},
    {'$sort': {'avgAge': -1}}
]

results = users.aggregate(pipeline)
```

### Statistics

#### `stats() -> Dict[str, Any]`
Get collection statistics.

```python
stats = users.stats()
print(f"Document count: {stats.get('documentCount', 0)}")
```

---

## Query Building

### Query Class

The `Query` class provides methods for building MongoDB-style queries.

```python
from lauradb import Query

q = Query()
```

### Comparison Operators

```python
# Equal
q.eq('age', 30)
# {'age': {'$eq': 30}}

# Not equal
q.ne('status', 'inactive')
# {'status': {'$ne': 'inactive'}}

# Greater than
q.gt('age', 25)
# {'age': {'$gt': 25}}

# Greater than or equal
q.gte('age', 18)
# {'age': {'$gte': 18}}

# Less than
q.lt('age', 65)
# {'age': {'$lt': 65}}

# Less than or equal
q.lte('price', 100)
# {'price': {'$lte': 100}}

# In array
q.in_('status', ['active', 'pending'])
# {'status': {'$in': ['active', 'pending']}}

# Not in array
q.nin('role', ['guest', 'anonymous'])
# {'role': {'$nin': ['guest', 'anonymous']}}
```

### Logical Operators

```python
# AND
q.and_(
    q.gte('age', 18),
    q.lt('age', 65),
    q.eq('active', True)
)
# {'$and': [{'age': {'$gte': 18}}, {'age': {'$lt': 65}}, {'active': {'$eq': True}}]}

# OR
q.or_(
    q.eq('role', 'admin'),
    q.eq('role', 'moderator')
)
# {'$or': [{'role': {'$eq': 'admin'}}, {'role': {'$eq': 'moderator'}}]}

# NOT
q.not_(q.eq('active', True))
# {'$not': {'active': {'$eq': True}}}
```

### Element Operators

```python
# Field exists
q.exists('email', True)
# {'email': {'$exists': True}}

# Field type
q.type_('age', 'number')
# {'age': {'$type': 'number'}}
```

### Array Operators

```python
# Array contains all values
q.all_('tags', ['python', 'database'])
# {'tags': {'$all': ['python', 'database']}}

# Array element matches condition
q.elem_match('scores', {'$gte': 80})
# {'scores': {'$elemMatch': {'$gte': 80}}}

# Array size
q.size('tags', 3)
# {'tags': {'$size': 3}}
```

### Evaluation Operators

```python
# Regular expression
q.regex('name', '^A.*')
# {'name': {'$regex': '^A.*'}}

# Text search
q.text('python database')
# {'$text': {'$search': 'python database'}}
```

### Geospatial Operators

```python
# Near point
q.near('location', -73.9857, 40.7580, 5000)
# {'location': {'$near': {'coordinates': [-73.9857, 40.7580], 'maxDistance': 5000}}}

# Within polygon
polygon = [[0, 0], [0, 10], [10, 10], [10, 0], [0, 0]]
q.geo_within('location', polygon)
# {'location': {'$geoWithin': {'coordinates': polygon}}}
```

### Complex Queries

```python
# Combine multiple conditions
query = q.and_(
    q.gte('age', 18),
    q.lt('age', 65),
    q.in_('role', ['user', 'admin']),
    q.or_(
        q.exists('email', True),
        q.exists('phone', True)
    )
)

results = users.find(query)
```

---

## Aggregation Pipeline

### Aggregation Class

The `Aggregation` class provides methods for building aggregation pipelines.

```python
from lauradb import Aggregation

agg = Aggregation()
```

### Pipeline Stages

#### $match - Filter Documents

```python
agg.match({'age': {'$gte': 18}})
```

#### $group - Group and Aggregate

```python
# Group by single field
agg.group(
    '$city',
    {
        'avgAge': agg.avg('$age'),
        'total': agg.sum(1),
        'names': agg.push('$name')
    }
)

# Group by compound key
agg.group(
    {'city': '$city', 'year': '$year'},
    {'revenue': agg.sum('$amount')}
)
```

#### $project - Transform Fields

```python
agg.project({
    'name': 1,
    'age': 1,
    'email': 1,
    '_id': 0
})

# With computed fields
agg.project({
    'name': 1,
    'fullName': agg.concat('$firstName', ' ', '$lastName'),
    'isAdult': agg.cond(
        {'$gte': ['$age', 18]},
        True,
        False
    )
})
```

#### $sort - Sort Documents

```python
agg.sort({'age': -1, 'name': 1})
```

#### $limit - Limit Results

```python
agg.limit(10)
```

#### $skip - Skip Documents

```python
agg.skip(20)
```

#### $unwind - Deconstruct Arrays

```python
# Basic unwind
agg.unwind('$tags')

# Preserve null and empty arrays
agg.unwind('$tags', preserve_null=True)
```

#### $lookup - Join Collections

```python
agg.lookup(
    from_collection='orders',
    local_field='userId',
    foreign_field='_id',
    as_field='userOrders'
)
```

### Aggregation Operators

#### Accumulator Operators

```python
# Sum
agg.sum('$amount')    # Sum field
agg.sum(1)            # Count

# Average
agg.avg('$age')

# Min/Max
agg.min_('$price')
agg.max_('$price')

# Count
agg.count()

# Array operators
agg.push('$name')          # Create array
agg.add_to_set('$category')  # Create array of unique values

# First/Last
agg.first('$createdAt')
agg.last('$updatedAt')
```

#### String Operators

```python
# Concatenate
agg.concat('$firstName', ' ', '$lastName')

# Substring
agg.substring('$name', 0, 5)

# Case conversion
agg.to_upper('$name')
agg.to_lower('$email')
```

#### Conditional Operator

```python
agg.cond(
    condition={'$gte': ['$age', 18]},
    true_expr='adult',
    false_expr='minor'
)
```

### Complete Pipeline Example

```python
pipeline = [
    # Filter adults
    agg.match({'age': {'$gte': 18}}),

    # Group by city
    agg.group(
        '$city',
        {
            'avgAge': agg.avg('$age'),
            'totalUsers': agg.count(),
            'minAge': agg.min_('$age'),
            'maxAge': agg.max_('$age'),
            'users': agg.push('$name')
        }
    ),

    # Add computed fields
    agg.project({
        'city': '$_id',
        'avgAge': 1,
        'totalUsers': 1,
        'ageRange': agg.concat(
            {'$toString': '$minAge'},
            '-',
            {'$toString': '$maxAge'}
        ),
        '_id': 0
    }),

    # Sort by average age
    agg.sort({'avgAge': -1}),

    # Top 10 cities
    agg.limit(10)
]

results = users.aggregate(pipeline)
```

---

## Index Management

### Creating Indexes

#### B+ Tree Index

```python
# Non-unique index
users.create_index('email')

# Unique index
users.create_index('email', unique=True, name='email_unique')

# Sparse index (index only documents with the field)
users.create_index('phone', sparse=True)
```

#### Compound Index

```python
users.create_compound_index(
    ['city', 'age'],
    unique=False,
    name='city_age_idx'
)
```

#### Text Index

```python
# Single field
posts.create_text_index(['title'], name='title_text')

# Multiple fields
posts.create_text_index(
    ['title', 'content'],
    name='posts_text'
)
```

#### Geospatial Index

```python
# 2d (planar coordinates)
locations.create_geo_index(
    'coordinates',
    geo_type='2d',
    name='coords_2d'
)

# 2dsphere (spherical/geographic coordinates)
locations.create_geo_index(
    'coordinates',
    geo_type='2dsphere',
    name='coords_sphere'
)
```

#### TTL Index

```python
# Auto-expire after 1 hour
sessions.create_ttl_index(
    'createdAt',
    expire_after_seconds=3600,
    name='session_ttl'
)
```

#### Partial Index

```python
# Index only active users
users.create_partial_index(
    'email',
    filter_expr={'active': True},
    unique=True,
    name='active_email_idx'
)
```

### Managing Indexes

#### List Indexes

```python
indexes = users.list_indexes()
for idx in indexes:
    print(f"Name: {idx['name']}")
    print(f"Field: {idx.get('field', idx.get('fields'))}")
    print(f"Type: {idx.get('type')}")
```

#### Drop Index

```python
users.drop_index('email_unique')
```

---

## Update Operators

### UpdateBuilder Class

```python
from lauradb import UpdateBuilder

u = UpdateBuilder()
```

### Field Update Operators

```python
# Set field value
u.set('name', 'Alice')
# {'$set': {'name': 'Alice'}}

# Remove field
u.unset('tempField')
# {'$unset': {'tempField': ''}}

# Rename field
u.rename('oldName', 'newName')
# {'$rename': {'oldName': 'newName'}}

# Set to current date/time
u.current_date('updatedAt')
# {'$currentDate': {'updatedAt': True}}
```

### Numeric Update Operators

```python
# Increment
u.inc('views', 1)
# {'$inc': {'views': 1}}

# Multiply
u.mul('price', 1.1)
# {'$mul': {'price': 1.1}}

# Update if less than current
u.min_('score', 100)
# {'$min': {'score': 100}}

# Update if greater than current
u.max_('score', 0)
# {'$max': {'score': 0}}
```

### Array Update Operators

```python
# Add to array
u.push('tags', 'python')
# {'$push': {'tags': 'python'}}

# Remove from array
u.pull('tags', 'deprecated')
# {'$pull': {'tags': 'deprecated'}}

# Remove multiple values
u.pull_all('tags', ['old', 'deprecated'])
# {'$pullAll': {'tags': ['old', 'deprecated']}}

# Add unique value to array
u.add_to_set('tags', 'python')
# {'$addToSet': {'tags': 'python'}}

# Remove first (-1) or last (1) element
u.pop('items', -1)
# {'$pop': {'items': -1}}
```

### Bitwise Operators

```python
# Bitwise AND
u.bit_and('flags', 0b1010)

# Bitwise OR
u.bit_or('flags', 0b0101)

# Bitwise XOR
u.bit_xor('flags', 0b1111)
```

### Combining Operations

```python
# Combine multiple update operations
update = u.combine(
    u.set('name', 'Alice'),
    u.inc('views', 1),
    u.push('tags', 'python'),
    u.current_date('updatedAt')
)

users.update_one({'_id': user_id}, update)
```

---

## Error Handling

### Connection Errors

```python
from lauradb import Client

try:
    client = Client(host='invalid-host', port=8080)
    client.ping()
except RuntimeError as e:
    print(f"Connection failed: {e}")
```

### API Errors

```python
try:
    users.insert_one({'name': 'Alice', 'email': 'alice@example.com'})
    # Try to insert duplicate (if unique index exists)
    users.insert_one({'name': 'Bob', 'email': 'alice@example.com'})
except ValueError as e:
    print(f"API error: {e}")
```

### Timeout Configuration

```python
# Set custom timeout
client = Client(
    host='localhost',
    port=8080,
    timeout=60  # 60 seconds
)
```

---

## Best Practices

### 1. Use Context Managers

```python
# Good
with Client(host='localhost', port=8080) as client:
    users = client.collection('users')
    users.insert_one({'name': 'Alice'})
# Connection automatically closed

# Avoid
client = Client(host='localhost', port=8080)
users = client.collection('users')
users.insert_one({'name': 'Alice'})
# Must remember to call client.close()
```

### 2. Create Indexes for Common Queries

```python
# If you frequently query by email
users.create_index('email', unique=True)

# If you frequently query by city and age
users.create_compound_index(['city', 'age'])
```

### 3. Use Projections to Limit Data Transfer

```python
# Good - only fetch needed fields
users.find(
    {'age': {'$gte': 18}},
    projection={'name': 1, 'email': 1, '_id': 0}
)

# Avoid - fetches all fields
users.find({'age': {'$gte': 18}})
```

### 4. Use Aggregation for Complex Analysis

```python
# Better performance for grouping and statistics
pipeline = [
    agg.match({'age': {'$gte': 18}}),
    agg.group('$city', {'avgAge': agg.avg('$age')})
]
results = users.aggregate(pipeline)
```

### 5. Implement Pagination for Large Result Sets

```python
page_size = 20
page = 1

results = users.find(
    {},
    sort={'createdAt': -1},
    skip=(page - 1) * page_size,
    limit=page_size
)
```

### 6. Use Bulk Inserts for Multiple Documents

```python
# Good
users.insert_many(documents)

# Avoid
for doc in documents:
    users.insert_one(doc)
```

### 7. Handle Errors Appropriately

```python
try:
    user_id = users.insert_one(document)
except ValueError as e:
    logger.error(f"Failed to insert document: {e}")
    # Handle error appropriately
```

---

## Examples

See the `examples/` directory for complete working examples:

- **basic_usage.py**: CRUD operations and query examples
- **aggregation.py**: Aggregation pipeline examples
- **indexes.py**: Index management examples

### Running Examples

```bash
# Make sure LauraDB server is running
cd examples
python basic_usage.py
python aggregation.py
python indexes.py
```

---

## API Reference Summary

### Client
- `__init__(host, port, https, timeout, max_connections)`
- `ping() -> bool`
- `stats() -> Dict`
- `list_collections() -> List[str]`
- `create_collection(name) -> bool`
- `drop_collection(name) -> bool`
- `collection(name) -> Collection`
- `close()`

### Collection
- `insert_one(document) -> str`
- `insert_many(documents) -> List[str]`
- `find_one(filter, projection) -> Optional[Dict]`
- `find(filter, projection, sort, skip, limit) -> List[Dict]`
- `count(filter) -> int`
- `update_one(filter, update) -> bool`
- `update_many(filter, update) -> int`
- `delete_one(filter) -> bool`
- `delete_many(filter) -> int`
- `aggregate(pipeline) -> List[Dict]`
- `create_index(field, unique, sparse, name) -> bool`
- `create_compound_index(fields, unique, name) -> bool`
- `create_text_index(fields, name) -> bool`
- `create_geo_index(field, geo_type, name) -> bool`
- `create_ttl_index(field, expire_after_seconds, name) -> bool`
- `create_partial_index(field, filter_expr, unique, name) -> bool`
- `list_indexes() -> List[Dict]`
- `drop_index(name) -> bool`
- `stats() -> Dict`

---

## Support

For issues and questions:
- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs

## License

MIT License - See LICENSE file for details.
