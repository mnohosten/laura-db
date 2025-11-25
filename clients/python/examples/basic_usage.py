#!/usr/bin/env python3
"""
LauraDB Python Client - Basic Usage Example

This example demonstrates:
- Connecting to LauraDB server
- Creating collections
- Basic CRUD operations (Insert, Find, Update, Delete)
- Query operators
- Projections and sorting
"""

from lauradb import Client, Query


def main():
    # Connect to LauraDB server
    print("=" * 60)
    print("LauraDB Python Client - Basic Usage Example")
    print("=" * 60)

    client = Client(host='localhost', port=8080)

    # Check connection
    if client.ping():
        print("✓ Successfully connected to LauraDB server")
    else:
        print("✗ Failed to connect to LauraDB server")
        return

    # Create a collection
    collection_name = 'users'
    print(f"\nCreating collection '{collection_name}'...")
    client.create_collection(collection_name)
    users = client.collection(collection_name)

    # Insert a single document
    print("\n1. INSERT ONE")
    print("-" * 60)
    user = {
        'name': 'Alice Johnson',
        'email': 'alice@example.com',
        'age': 30,
        'city': 'New York',
        'active': True,
        'tags': ['python', 'golang', 'databases']
    }
    user_id = users.insert_one(user)
    print(f"Inserted user with ID: {user_id}")

    # Insert multiple documents
    print("\n2. INSERT MANY")
    print("-" * 60)
    more_users = [
        {
            'name': 'Bob Smith',
            'email': 'bob@example.com',
            'age': 25,
            'city': 'San Francisco',
            'active': True,
            'tags': ['javascript', 'react']
        },
        {
            'name': 'Charlie Davis',
            'email': 'charlie@example.com',
            'age': 35,
            'city': 'New York',
            'active': False,
            'tags': ['java', 'spring']
        },
        {
            'name': 'Diana Wilson',
            'email': 'diana@example.com',
            'age': 28,
            'city': 'Seattle',
            'active': True,
            'tags': ['python', 'machine-learning']
        },
        {
            'name': 'Eve Martinez',
            'email': 'eve@example.com',
            'age': 32,
            'city': 'San Francisco',
            'active': True,
            'tags': ['golang', 'kubernetes']
        }
    ]
    ids = users.insert_many(more_users)
    print(f"Inserted {len(ids)} users")

    # Find all documents
    print("\n3. FIND ALL")
    print("-" * 60)
    all_users = users.find()
    print(f"Total users: {len(all_users)}")
    for user in all_users:
        print(f"  - {user['name']}, {user['age']}, {user['city']}")

    # Find with filter
    print("\n4. FIND WITH FILTER")
    print("-" * 60)
    ny_users = users.find({'city': 'New York'})
    print(f"Users in New York: {len(ny_users)}")
    for user in ny_users:
        print(f"  - {user['name']}")

    # Find with comparison operators
    print("\n5. FIND WITH COMPARISON OPERATORS")
    print("-" * 60)
    young_users = users.find({'age': {'$lt': 30}})
    print(f"Users under 30: {len(young_users)}")
    for user in young_users:
        print(f"  - {user['name']}, age {user['age']}")

    # Find with Query builder
    print("\n6. FIND WITH QUERY BUILDER")
    print("-" * 60)
    q = Query()
    active_adults = users.find(
        q.and_(
            q.gte('age', 25),
            q.lt('age', 35),
            q.eq('active', True)
        )
    )
    print(f"Active users aged 25-34: {len(active_adults)}")
    for user in active_adults:
        print(f"  - {user['name']}, age {user['age']}")

    # Find with $in operator
    print("\n7. FIND WITH $IN OPERATOR")
    print("-" * 60)
    west_coast = users.find({'city': {'$in': ['San Francisco', 'Seattle']}})
    print(f"Users on west coast: {len(west_coast)}")
    for user in west_coast:
        print(f"  - {user['name']} in {user['city']}")

    # Find with array operators
    print("\n8. FIND WITH ARRAY OPERATORS")
    print("-" * 60)
    python_users = users.find({'tags': 'python'})
    print(f"Python developers: {len(python_users)}")
    for user in python_users:
        print(f"  - {user['name']}: {', '.join(user['tags'])}")

    # Find one document
    print("\n9. FIND ONE")
    print("-" * 60)
    alice = users.find_one({'name': 'Alice Johnson'})
    if alice:
        print(f"Found: {alice['name']}, {alice['email']}")

    # Find with projection
    print("\n10. FIND WITH PROJECTION")
    print("-" * 60)
    names_only = users.find(
        {},
        projection={'name': 1, 'city': 1, '_id': 0}
    )
    print("Names and cities:")
    for user in names_only:
        print(f"  - {user['name']} from {user['city']}")

    # Find with sorting
    print("\n11. FIND WITH SORTING")
    print("-" * 60)
    sorted_users = users.find({}, sort={'age': -1})
    print("Users sorted by age (descending):")
    for user in sorted_users:
        print(f"  - {user['name']}, age {user['age']}")

    # Find with pagination
    print("\n12. FIND WITH PAGINATION")
    print("-" * 60)
    page_size = 2
    page_2 = users.find({}, sort={'name': 1}, skip=2, limit=page_size)
    print(f"Page 2 (showing {page_size} users):")
    for user in page_2:
        print(f"  - {user['name']}")

    # Count documents
    print("\n13. COUNT")
    print("-" * 60)
    total = users.count()
    active_count = users.count({'active': True})
    print(f"Total users: {total}")
    print(f"Active users: {active_count}")

    # Update one document
    print("\n14. UPDATE ONE")
    print("-" * 60)
    users.update_one(
        {'name': 'Alice Johnson'},
        {'$set': {'age': 31, 'city': 'Boston'}}
    )
    alice = users.find_one({'name': 'Alice Johnson'})
    print(f"Updated Alice: age {alice['age']}, city {alice['city']}")

    # Update many documents
    print("\n15. UPDATE MANY")
    print("-" * 60)
    modified = users.update_many(
        {'city': 'San Francisco'},
        {'$set': {'timezone': 'PST'}}
    )
    print(f"Added timezone to {modified} users in San Francisco")

    # Update with $inc operator
    print("\n16. UPDATE WITH $INC")
    print("-" * 60)
    users.update_one(
        {'name': 'Bob Smith'},
        {'$inc': {'age': 1}}
    )
    bob = users.find_one({'name': 'Bob Smith'})
    print(f"Bob's age after increment: {bob['age']}")

    # Update with array operators
    print("\n17. UPDATE WITH ARRAY OPERATORS")
    print("-" * 60)
    users.update_one(
        {'name': 'Alice Johnson'},
        {'$push': {'tags': 'rust'}}
    )
    alice = users.find_one({'name': 'Alice Johnson'})
    print(f"Alice's tags: {', '.join(alice['tags'])}")

    # Delete one document
    print("\n18. DELETE ONE")
    print("-" * 60)
    users.delete_one({'name': 'Charlie Davis'})
    print("Deleted Charlie Davis")
    remaining = users.count()
    print(f"Remaining users: {remaining}")

    # Delete many documents
    print("\n19. DELETE MANY")
    print("-" * 60)
    deleted = users.delete_many({'active': False})
    print(f"Deleted {deleted} inactive users")

    # Final count
    print("\n20. FINAL COUNT")
    print("-" * 60)
    final_count = users.count()
    print(f"Final user count: {final_count}")

    # Collection statistics
    print("\n21. COLLECTION STATISTICS")
    print("-" * 60)
    stats = users.stats()
    print(f"Collection stats:")
    for key, value in stats.items():
        print(f"  {key}: {value}")

    # Cleanup
    print("\n22. CLEANUP")
    print("-" * 60)
    client.drop_collection(collection_name)
    print(f"Dropped collection '{collection_name}'")

    # Close connection
    client.close()
    print("\n✓ Example completed successfully")
    print("=" * 60)


if __name__ == '__main__':
    main()
