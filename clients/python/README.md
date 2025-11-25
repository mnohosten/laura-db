# LauraDB Python Client

Python client library for [LauraDB](https://github.com/mnohosten/laura-db) - A MongoDB-like document database written in Go.

## Features

- 🔌 **Simple Connection Management**: Easy-to-use client with connection pooling
- 📦 **Complete CRUD Operations**: Insert, Find, Update, Delete with MongoDB-like API
- 🔍 **Rich Query Language**: Full support for comparison, logical, array, and element operators
- 📊 **Aggregation Pipeline**: $match, $group, $project, $sort, $limit, $skip stages
- 🗂️ **Index Management**: B+ tree, compound, text, geospatial, TTL, and partial indexes
- ⚡ **High Performance**: Connection pooling and efficient HTTP communication
- 🐍 **Pythonic API**: Follows Python conventions with type hints
- ✨ **Zero Dependencies**: Only requires `requests` library

## Installation

```bash
pip install lauradb-client
```

Or from source:

```bash
cd clients/python
pip install -e .
```

## Quick Start

```python
from lauradb import Client

# Connect to LauraDB server
client = Client(host='localhost', port=8080)

# Get a collection
users = client.collection('users')

# Insert documents
user_id = users.insert_one({
    'name': 'Alice',
    'email': 'alice@example.com',
    'age': 30
})

# Find documents
results = users.find({'age': {'$gte': 25}})
for doc in results:
    print(doc)

# Update documents
users.update_one(
    {'name': 'Alice'},
    {'$set': {'age': 31}}
)

# Delete documents
users.delete_one({'name': 'Alice'})
```

## Advanced Usage

### Query Operators

```python
# Comparison operators
users.find({'age': {'$gt': 25, '$lt': 40}})
users.find({'name': {'$in': ['Alice', 'Bob']}})

# Logical operators
users.find({
    '$and': [
        {'age': {'$gte': 25}},
        {'active': True}
    ]
})

# Array operators
posts.find({'tags': {'$all': ['python', 'database']}})
posts.find({'comments': {'$size': 5}})

# Element operators
users.find({'email': {'$exists': True}})
users.find({'age': {'$type': 'number'}})
```

### Aggregation Pipeline

```python
# Group by field and calculate aggregates
pipeline = [
    {'$match': {'age': {'$gte': 18}}},
    {'$group': {
        '_id': '$city',
        'avgAge': {'$avg': '$age'},
        'count': {'$count': {}}
    }},
    {'$sort': {'avgAge': -1}},
    {'$limit': 10}
]

results = users.aggregate(pipeline)
```

### Index Management

```python
# Create B+ tree index
users.create_index('email', unique=True)

# Create compound index
users.create_compound_index(['city', 'age'], name='city_age_idx')

# Create text index for full-text search
posts.create_text_index(['title', 'content'], name='posts_text')

# Create geospatial index
locations.create_geo_index('coordinates', geo_type='2dsphere')

# Create TTL index for automatic expiration
sessions.create_ttl_index('createdAt', expire_after_seconds=3600)

# Create partial index
users.create_partial_index(
    'email',
    filter_expr={'active': True},
    unique=True
)
```

### Query Options

```python
# Projection: select specific fields
users.find(
    {'age': {'$gte': 25}},
    projection={'name': 1, 'email': 1, '_id': 0}
)

# Sorting
users.find({}, sort={'age': -1, 'name': 1})

# Pagination
users.find({}, skip=20, limit=10)

# Combine options
users.find(
    {'active': True},
    projection={'name': 1, 'email': 1},
    sort={'createdAt': -1},
    skip=0,
    limit=50
)
```

## API Reference

See [docs/python-client.md](../../docs/python-client.md) for complete API documentation.

## Examples

Check out the [examples/](examples/) directory for:
- **basic-usage.py**: Basic CRUD operations
- **aggregation.py**: Aggregation pipeline examples
- **indexes.py**: Index management examples

## Requirements

- Python 3.7+
- LauraDB server running (default: localhost:8080)
- `requests` library

## Development

```bash
# Install development dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run tests with coverage
pytest --cov=lauradb --cov-report=html

# Format code
black lauradb tests examples

# Type checking
mypy lauradb
```

## License

MIT License - see LICENSE file for details.

## Links

- [LauraDB Repository](https://github.com/mnohosten/laura-db)
- [Documentation](../../docs/)
- [HTTP API Reference](../../docs/http-api.md)
