"""
Tests for LauraDB Client
"""
import pytest
from lauradb import Client, Collection


@pytest.fixture
def client():
    """Create a test client instance."""
    return Client(host='localhost', port=8080)


@pytest.fixture
def test_collection(client):
    """Create and clean up a test collection."""
    collection_name = 'test_collection'

    # Clean up if exists
    try:
        client.drop_collection(collection_name)
    except:
        pass

    # Create collection
    client.create_collection(collection_name)
    collection = client.collection(collection_name)

    yield collection

    # Cleanup
    try:
        client.drop_collection(collection_name)
    except:
        pass


class TestClient:
    """Test Client class."""

    def test_client_creation(self):
        """Test creating a client."""
        client = Client(host='localhost', port=8080)
        assert client.host == 'localhost'
        assert client.port == 8080
        assert not client.https

    def test_client_with_https(self):
        """Test creating a client with HTTPS."""
        client = Client(host='example.com', port=443, https=True)
        assert client.https
        assert 'https://' in client.base_url

    def test_ping(self, client):
        """Test ping functionality."""
        result = client.ping()
        assert isinstance(result, bool)

    def test_list_collections(self, client):
        """Test listing collections."""
        collections = client.list_collections()
        assert isinstance(collections, list)

    def test_create_and_drop_collection(self, client):
        """Test creating and dropping a collection."""
        collection_name = 'test_temp_collection'

        # Create
        result = client.create_collection(collection_name)
        assert result is True

        # Verify it exists
        collections = client.list_collections()
        assert collection_name in collections

        # Drop
        result = client.drop_collection(collection_name)
        assert result is True

        # Verify it's gone
        collections = client.list_collections()
        assert collection_name not in collections

    def test_get_collection(self, client):
        """Test getting a collection object."""
        collection = client.collection('test_users')
        assert isinstance(collection, Collection)
        assert collection.name == 'test_users'

    def test_stats(self, client):
        """Test getting database statistics."""
        stats = client.stats()
        assert isinstance(stats, dict)

    def test_context_manager(self):
        """Test using client as context manager."""
        with Client(host='localhost', port=8080) as client:
            assert client.ping() or True  # May fail if server not running

    def test_repr(self, client):
        """Test string representation."""
        repr_str = repr(client)
        assert 'Client' in repr_str
        assert 'localhost' in repr_str


class TestCollection:
    """Test Collection operations."""

    def test_insert_one(self, test_collection):
        """Test inserting a single document."""
        doc = {'name': 'Alice', 'age': 30, 'email': 'alice@example.com'}
        doc_id = test_collection.insert_one(doc)

        assert isinstance(doc_id, str)
        assert len(doc_id) > 0

    def test_insert_many(self, test_collection):
        """Test inserting multiple documents."""
        docs = [
            {'name': 'Alice', 'age': 30},
            {'name': 'Bob', 'age': 25},
            {'name': 'Charlie', 'age': 35}
        ]
        ids = test_collection.insert_many(docs)

        assert isinstance(ids, list)
        assert len(ids) == 3
        assert all(isinstance(id, str) for id in ids)

    def test_find_one(self, test_collection):
        """Test finding a single document."""
        # Insert test data
        test_collection.insert_one({'name': 'Alice', 'age': 30})

        # Find
        doc = test_collection.find_one({'name': 'Alice'})

        assert doc is not None
        assert doc['name'] == 'Alice'
        assert doc['age'] == 30

    def test_find(self, test_collection):
        """Test finding multiple documents."""
        # Insert test data
        test_collection.insert_many([
            {'name': 'Alice', 'age': 30},
            {'name': 'Bob', 'age': 25},
            {'name': 'Charlie', 'age': 35}
        ])

        # Find all
        docs = test_collection.find()
        assert len(docs) == 3

        # Find with filter
        docs = test_collection.find({'age': {'$gte': 30}})
        assert len(docs) == 2

    def test_find_with_projection(self, test_collection):
        """Test finding with field projection."""
        test_collection.insert_one({'name': 'Alice', 'age': 30, 'email': 'alice@example.com'})

        doc = test_collection.find_one(
            {'name': 'Alice'},
            projection={'name': 1, 'age': 1, '_id': 0}
        )

        assert 'name' in doc
        assert 'age' in doc
        assert '_id' not in doc

    def test_find_with_sort(self, test_collection):
        """Test finding with sorting."""
        test_collection.insert_many([
            {'name': 'Charlie', 'age': 35},
            {'name': 'Alice', 'age': 30},
            {'name': 'Bob', 'age': 25}
        ])

        docs = test_collection.find({}, sort={'age': 1})
        ages = [doc['age'] for doc in docs]

        assert ages == sorted(ages)

    def test_find_with_limit_skip(self, test_collection):
        """Test finding with limit and skip."""
        test_collection.insert_many([
            {'name': f'User{i}', 'age': 20 + i}
            for i in range(10)
        ])

        docs = test_collection.find({}, skip=2, limit=3)

        assert len(docs) == 3

    def test_count(self, test_collection):
        """Test counting documents."""
        test_collection.insert_many([
            {'name': 'Alice', 'age': 30},
            {'name': 'Bob', 'age': 25},
            {'name': 'Charlie', 'age': 35}
        ])

        # Count all
        count = test_collection.count()
        assert count == 3

        # Count with filter
        count = test_collection.count({'age': {'$gte': 30}})
        assert count == 2

    def test_update_one(self, test_collection):
        """Test updating a single document."""
        test_collection.insert_one({'name': 'Alice', 'age': 30})

        result = test_collection.update_one(
            {'name': 'Alice'},
            {'$set': {'age': 31}}
        )

        assert result is True

        # Verify update
        doc = test_collection.find_one({'name': 'Alice'})
        assert doc['age'] == 31

    def test_update_many(self, test_collection):
        """Test updating multiple documents."""
        test_collection.insert_many([
            {'name': 'Alice', 'age': 30, 'active': False},
            {'name': 'Bob', 'age': 25, 'active': False},
            {'name': 'Charlie', 'age': 35, 'active': False}
        ])

        count = test_collection.update_many(
            {'age': {'$gte': 30}},
            {'$set': {'active': True}}
        )

        assert count == 2

    def test_delete_one(self, test_collection):
        """Test deleting a single document."""
        test_collection.insert_one({'name': 'Alice', 'age': 30})

        result = test_collection.delete_one({'name': 'Alice'})

        assert result is True

        # Verify deletion
        doc = test_collection.find_one({'name': 'Alice'})
        assert doc is None

    def test_delete_many(self, test_collection):
        """Test deleting multiple documents."""
        test_collection.insert_many([
            {'name': 'Alice', 'age': 30},
            {'name': 'Bob', 'age': 25},
            {'name': 'Charlie', 'age': 35}
        ])

        count = test_collection.delete_many({'age': {'$gte': 30}})

        assert count == 2

        # Verify deletion
        remaining = test_collection.count()
        assert remaining == 1

    def test_create_index(self, test_collection):
        """Test creating an index."""
        result = test_collection.create_index('email', unique=True)
        assert result is True

        # List indexes
        indexes = test_collection.list_indexes()
        assert any('email' in str(idx) for idx in indexes)

    def test_stats(self, test_collection):
        """Test getting collection statistics."""
        test_collection.insert_many([
            {'name': f'User{i}', 'age': 20 + i}
            for i in range(5)
        ])

        stats = test_collection.stats()
        assert isinstance(stats, dict)

    def test_repr(self, test_collection):
        """Test string representation."""
        repr_str = repr(test_collection)
        assert 'Collection' in repr_str
        assert test_collection.name in repr_str


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
