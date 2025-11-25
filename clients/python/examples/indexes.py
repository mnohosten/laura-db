#!/usr/bin/env python3
"""
LauraDB Python Client - Index Management Example

This example demonstrates:
- Creating B+ tree indexes (unique and non-unique)
- Creating compound indexes
- Creating text indexes for full-text search
- Creating geospatial indexes
- Creating TTL indexes for automatic expiration
- Creating partial indexes
- Listing and dropping indexes
"""

from lauradb import Client
import time
from datetime import datetime


def main():
    print("=" * 60)
    print("LauraDB Python Client - Index Management Example")
    print("=" * 60)

    # Connect to LauraDB
    client = Client(host='localhost', port=8080)

    if not client.ping():
        print("✗ Failed to connect to LauraDB server")
        return

    print("✓ Connected to LauraDB server")

    # Example 1: B+ Tree Index (Unique)
    print("\n1. B+ TREE INDEX - Unique Email")
    print("-" * 60)
    collection_name = 'users'
    client.create_collection(collection_name)
    users = client.collection(collection_name)

    # Create unique index on email
    print("Creating unique index on 'email' field...")
    users.create_index('email', unique=True, name='email_unique')

    # Insert users
    users.insert_many([
        {'name': 'Alice', 'email': 'alice@example.com', 'age': 30},
        {'name': 'Bob', 'email': 'bob@example.com', 'age': 25},
        {'name': 'Charlie', 'email': 'charlie@example.com', 'age': 35}
    ])
    print("✓ Inserted 3 users with unique emails")

    # Try to insert duplicate email (should fail)
    print("Attempting to insert duplicate email...")
    try:
        users.insert_one({'name': 'David', 'email': 'alice@example.com', 'age': 28})
        print("⚠️  Duplicate insertion succeeded (unexpected)")
    except Exception as e:
        print(f"✓ Duplicate rejected: {type(e).__name__}")

    # List indexes
    print("\nCurrent indexes:")
    indexes = users.list_indexes()
    for idx in indexes:
        print(f"  - {idx.get('name', 'unnamed')}: {idx.get('field', idx.get('fields', 'N/A'))}")

    # Example 2: Compound Index
    print("\n2. COMPOUND INDEX - City and Age")
    print("-" * 60)

    # Add city data
    users.update_many({}, {'$set': {'city': 'New York'}})
    users.update_one({'name': 'Bob'}, {'$set': {'city': 'San Francisco'}})

    # Create compound index
    print("Creating compound index on ['city', 'age']...")
    users.create_compound_index(['city', 'age'], name='city_age_idx')
    print("✓ Compound index created")

    # Query using compound index
    results = users.find({'city': 'New York', 'age': {'$gte': 30}})
    print(f"Found {len(results)} users in New York aged 30+")

    # Example 3: Text Index
    print("\n3. TEXT INDEX - Full-Text Search")
    print("-" * 60)
    collection_name = 'posts'
    client.create_collection(collection_name)
    posts = client.collection(collection_name)

    # Insert blog posts
    posts.insert_many([
        {
            'title': 'Introduction to Python',
            'content': 'Python is a powerful programming language for data science and web development',
            'tags': ['python', 'programming']
        },
        {
            'title': 'Database Indexing Best Practices',
            'content': 'Learn how to optimize your database queries with proper indexing strategies',
            'tags': ['database', 'performance']
        },
        {
            'title': 'Go Language Tutorial',
            'content': 'Go is a statically typed language designed for building scalable systems',
            'tags': ['golang', 'programming']
        },
        {
            'title': 'Python Data Analysis',
            'content': 'Using pandas and numpy for data analysis and visualization in Python',
            'tags': ['python', 'data-science']
        }
    ])
    print("Inserted 4 blog posts")

    # Create text index
    print("Creating text index on ['title', 'content']...")
    posts.create_text_index(['title', 'content'], name='posts_text')
    print("✓ Text index created")

    # Text search
    print("\nSearching for 'python'...")
    results = posts.find({'$text': {'$search': 'python'}})
    print(f"Found {len(results)} posts:")
    for post in results:
        print(f"  - {post['title']}")

    # Example 4: Geospatial Index
    print("\n4. GEOSPATIAL INDEX - 2dsphere")
    print("-" * 60)
    collection_name = 'locations'
    client.create_collection(collection_name)
    locations = client.collection(collection_name)

    # Insert locations with coordinates [longitude, latitude]
    locations.insert_many([
        {
            'name': 'Empire State Building',
            'coordinates': [-73.9857, 40.7484],
            'type': 'landmark'
        },
        {
            'name': 'Golden Gate Bridge',
            'coordinates': [-122.4783, 37.8199],
            'type': 'landmark'
        },
        {
            'name': 'Space Needle',
            'coordinates': [-122.3493, 47.6205],
            'type': 'landmark'
        },
        {
            'name': 'Statue of Liberty',
            'coordinates': [-74.0445, 40.6892],
            'type': 'landmark'
        }
    ])
    print("Inserted 4 locations")

    # Create geospatial index
    print("Creating 2dsphere index on 'coordinates'...")
    locations.create_geo_index('coordinates', geo_type='2dsphere', name='coords_2dsphere')
    print("✓ Geospatial index created")

    # Find nearby locations (near New York City: -74.006, 40.7128)
    print("\nFinding landmarks near NYC (within 50km)...")
    results = locations.find({
        'coordinates': {
            '$near': {
                'coordinates': [-74.006, 40.7128],
                'maxDistance': 50000  # 50km in meters
            }
        }
    })
    print(f"Found {len(results)} nearby landmarks:")
    for loc in results:
        print(f"  - {loc['name']}")

    # Example 5: TTL Index
    print("\n5. TTL INDEX - Auto-Expiring Sessions")
    print("-" * 60)
    collection_name = 'sessions'
    client.create_collection(collection_name)
    sessions = client.collection(collection_name)

    # Create TTL index (expire after 3600 seconds = 1 hour)
    print("Creating TTL index on 'createdAt' (expire after 3600s)...")
    sessions.create_ttl_index('createdAt', expire_after_seconds=3600, name='session_ttl')
    print("✓ TTL index created")

    # Insert sessions
    now = datetime.now().isoformat()
    sessions.insert_many([
        {'sessionId': 'sess1', 'userId': 'user1', 'createdAt': now},
        {'sessionId': 'sess2', 'userId': 'user2', 'createdAt': now},
        {'sessionId': 'sess3', 'userId': 'user3', 'createdAt': now}
    ])
    print(f"✓ Inserted 3 sessions (will expire after 1 hour)")

    count = sessions.count()
    print(f"Current active sessions: {count}")

    # Example 6: Partial Index
    print("\n6. PARTIAL INDEX - Active Users Only")
    print("-" * 60)

    # Go back to users collection
    users = client.collection('users')

    # Add active flag to all users
    users.update_many({}, {'$set': {'active': True}})
    users.update_one({'name': 'Charlie'}, {'$set': {'active': False}})

    # Create partial index (index only active users)
    print("Creating partial index on 'email' for active users...")
    users.create_partial_index(
        'email',
        filter_expr={'active': True},
        unique=True,
        name='active_email_idx'
    )
    print("✓ Partial index created")

    # Count active users
    active = users.count({'active': True})
    total = users.count()
    print(f"Active users: {active}/{total}")
    print("(Partial index covers only active users)")

    # Example 7: Non-unique Index for Performance
    print("\n7. NON-UNIQUE INDEX - Age for Range Queries")
    print("-" * 60)

    # Create non-unique index on age
    print("Creating non-unique index on 'age'...")
    users.create_index('age', unique=False, name='age_idx')
    print("✓ Non-unique index created")

    # Range query on age
    results = users.find({'age': {'$gte': 25, '$lte': 35}})
    print(f"Found {len(results)} users aged 25-35:")
    for user in results:
        print(f"  - {user['name']}, age {user['age']}")

    # Example 8: List All Indexes
    print("\n8. LIST ALL INDEXES")
    print("-" * 60)
    print("Indexes in 'users' collection:")
    indexes = users.list_indexes()
    for idx in indexes:
        print(f"  - {idx.get('name', 'unnamed')}:")
        print(f"    Field: {idx.get('field', idx.get('fields', 'N/A'))}")
        print(f"    Type: {idx.get('type', 'unknown')}")
        if 'unique' in idx:
            print(f"    Unique: {idx['unique']}")

    # Example 9: Drop Index
    print("\n9. DROP INDEX")
    print("-" * 60)
    print("Dropping 'age_idx'...")
    users.drop_index('age_idx')
    print("✓ Index dropped")

    # Verify it's gone
    indexes = users.list_indexes()
    age_idx_exists = any(idx.get('name') == 'age_idx' for idx in indexes)
    print(f"'age_idx' exists: {age_idx_exists}")

    # Example 10: Index Statistics
    print("\n10. COLLECTION STATISTICS")
    print("-" * 60)
    stats = users.stats()
    print("Users collection statistics:")
    for key, value in stats.items():
        if key == 'indexes':
            print(f"  {key}: {len(value) if isinstance(value, list) else value}")
        else:
            print(f"  {key}: {value}")

    # Cleanup
    print("\n11. CLEANUP")
    print("-" * 60)
    for coll_name in ['users', 'posts', 'locations', 'sessions']:
        try:
            client.drop_collection(coll_name)
            print(f"✓ Dropped collection '{coll_name}'")
        except:
            pass

    client.close()
    print("\n✓ Example completed successfully")
    print("=" * 60)


if __name__ == '__main__':
    main()
