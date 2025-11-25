"""
LauraDB Collection - Interface for collection operations
"""
from typing import Dict, Any, List, Optional, Union


class Collection:
    """
    Collection object for performing CRUD operations.

    Args:
        client: LauraDB client instance
        name: Collection name

    Example:
        >>> users = client.collection('users')
        >>> users.insert_one({'name': 'Alice', 'age': 30})
    """

    def __init__(self, client, name: str):
        self.client = client
        self.name = name
        self._base_path = f"/collections/{name}"

    def insert_one(self, document: Dict[str, Any]) -> str:
        """
        Insert a single document into the collection.

        Args:
            document: Document to insert

        Returns:
            Inserted document ID

        Example:
            >>> doc_id = users.insert_one({'name': 'Alice', 'age': 30})
            >>> print(doc_id)
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/insert",
            body={"document": document}
        )
        return response.get("result", {}).get("id", "")

    def insert_many(self, documents: List[Dict[str, Any]]) -> List[str]:
        """
        Insert multiple documents into the collection.

        Args:
            documents: List of documents to insert

        Returns:
            List of inserted document IDs

        Example:
            >>> ids = users.insert_many([
            ...     {'name': 'Alice', 'age': 30},
            ...     {'name': 'Bob', 'age': 25}
            ... ])
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/insert-many",
            body={"documents": documents}
        )
        return response.get("result", {}).get("ids", [])

    def find_one(
        self,
        filter: Optional[Dict[str, Any]] = None,
        projection: Optional[Dict[str, int]] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Find a single document matching the filter.

        Args:
            filter: Query filter (default: {})
            projection: Field projection (1 = include, 0 = exclude)

        Returns:
            Matching document or None

        Example:
            >>> doc = users.find_one({'name': 'Alice'})
            >>> print(doc['age'])
        """
        body = {"filter": filter or {}}
        if projection:
            body["projection"] = projection

        response = self.client._request(
            "POST",
            f"{self._base_path}/find-one",
            body=body
        )
        return response.get("result", {}).get("document")

    def find(
        self,
        filter: Optional[Dict[str, Any]] = None,
        projection: Optional[Dict[str, int]] = None,
        sort: Optional[Dict[str, int]] = None,
        skip: Optional[int] = None,
        limit: Optional[int] = None
    ) -> List[Dict[str, Any]]:
        """
        Find multiple documents matching the filter.

        Args:
            filter: Query filter (default: {})
            projection: Field projection (1 = include, 0 = exclude)
            sort: Sort specification (1 = ascending, -1 = descending)
            skip: Number of documents to skip
            limit: Maximum number of documents to return

        Returns:
            List of matching documents

        Example:
            >>> docs = users.find(
            ...     {'age': {'$gte': 25}},
            ...     projection={'name': 1, 'age': 1},
            ...     sort={'age': -1},
            ...     limit=10
            ... )
        """
        body = {"filter": filter or {}}
        if projection:
            body["projection"] = projection
        if sort:
            body["sort"] = sort
        if skip is not None:
            body["skip"] = skip
        if limit is not None:
            body["limit"] = limit

        response = self.client._request(
            "POST",
            f"{self._base_path}/find",
            body=body
        )
        return response.get("result", {}).get("documents", [])

    def count(self, filter: Optional[Dict[str, Any]] = None) -> int:
        """
        Count documents matching the filter.

        Args:
            filter: Query filter (default: {})

        Returns:
            Number of matching documents

        Example:
            >>> count = users.count({'age': {'$gte': 25}})
            >>> print(f"Found {count} users")
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/count",
            body={"filter": filter or {}}
        )
        return response.get("result", {}).get("count", 0)

    def update_one(
        self,
        filter: Dict[str, Any],
        update: Dict[str, Any]
    ) -> bool:
        """
        Update a single document matching the filter.

        Args:
            filter: Query filter
            update: Update operations (must use update operators like $set)

        Returns:
            True if a document was updated

        Example:
            >>> users.update_one(
            ...     {'name': 'Alice'},
            ...     {'$set': {'age': 31}}
            ... )
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/update-one",
            body={"filter": filter, "update": update}
        )
        return response.get("result", {}).get("modified", 0) > 0

    def update_many(
        self,
        filter: Dict[str, Any],
        update: Dict[str, Any]
    ) -> int:
        """
        Update multiple documents matching the filter.

        Args:
            filter: Query filter
            update: Update operations (must use update operators like $set)

        Returns:
            Number of documents updated

        Example:
            >>> count = users.update_many(
            ...     {'age': {'$lt': 18}},
            ...     {'$set': {'minor': True}}
            ... )
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/update-many",
            body={"filter": filter, "update": update}
        )
        return response.get("result", {}).get("modified", 0)

    def delete_one(self, filter: Dict[str, Any]) -> bool:
        """
        Delete a single document matching the filter.

        Args:
            filter: Query filter

        Returns:
            True if a document was deleted

        Example:
            >>> users.delete_one({'name': 'Alice'})
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/delete-one",
            body={"filter": filter}
        )
        return response.get("result", {}).get("deleted", 0) > 0

    def delete_many(self, filter: Dict[str, Any]) -> int:
        """
        Delete multiple documents matching the filter.

        Args:
            filter: Query filter

        Returns:
            Number of documents deleted

        Example:
            >>> count = users.delete_many({'age': {'$lt': 18}})
            >>> print(f"Deleted {count} users")
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/delete-many",
            body={"filter": filter}
        )
        return response.get("result", {}).get("deleted", 0)

    def aggregate(self, pipeline: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """
        Execute an aggregation pipeline.

        Args:
            pipeline: List of aggregation stages

        Returns:
            List of aggregation results

        Example:
            >>> results = users.aggregate([
            ...     {'$match': {'age': {'$gte': 18}}},
            ...     {'$group': {
            ...         '_id': '$city',
            ...         'avgAge': {'$avg': '$age'},
            ...         'count': {'$count': {}}
            ...     }},
            ...     {'$sort': {'avgAge': -1}}
            ... ])
        """
        response = self.client._request(
            "POST",
            f"{self._base_path}/aggregate",
            body={"pipeline": pipeline}
        )
        return response.get("result", {}).get("documents", [])

    def create_index(
        self,
        field: str,
        unique: bool = False,
        sparse: bool = False,
        name: Optional[str] = None
    ) -> bool:
        """
        Create a B+ tree index on a field.

        Args:
            field: Field name to index
            unique: Whether the index should enforce uniqueness
            sparse: Whether to index only documents with the field
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> users.create_index('email', unique=True)
        """
        body = {
            "field": field,
            "unique": unique,
            "sparse": sparse
        }
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes",
            body=body
        )
        return response.get("ok", False)

    def create_compound_index(
        self,
        fields: List[str],
        unique: bool = False,
        name: Optional[str] = None
    ) -> bool:
        """
        Create a compound index on multiple fields.

        Args:
            fields: List of field names
            unique: Whether the index should enforce uniqueness
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> users.create_compound_index(['city', 'age'], name='city_age_idx')
        """
        body = {
            "fields": fields,
            "unique": unique
        }
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes/compound",
            body=body
        )
        return response.get("ok", False)

    def create_text_index(
        self,
        fields: List[str],
        name: Optional[str] = None
    ) -> bool:
        """
        Create a text index for full-text search.

        Args:
            fields: List of text field names
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> posts.create_text_index(['title', 'content'], name='posts_text')
        """
        body = {"fields": fields}
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes/text",
            body=body
        )
        return response.get("ok", False)

    def create_geo_index(
        self,
        field: str,
        geo_type: str = "2d",
        name: Optional[str] = None
    ) -> bool:
        """
        Create a geospatial index.

        Args:
            field: Field name containing coordinates
            geo_type: Index type ('2d' for planar, '2dsphere' for spherical)
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> locations.create_geo_index('coordinates', geo_type='2dsphere')
        """
        body = {
            "field": field,
            "geoType": geo_type
        }
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes/geo",
            body=body
        )
        return response.get("ok", False)

    def create_ttl_index(
        self,
        field: str,
        expire_after_seconds: int,
        name: Optional[str] = None
    ) -> bool:
        """
        Create a TTL (Time-To-Live) index for automatic document expiration.

        Args:
            field: Field name containing timestamp
            expire_after_seconds: Seconds after which documents expire
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> sessions.create_ttl_index('createdAt', expire_after_seconds=3600)
        """
        body = {
            "field": field,
            "expireAfterSeconds": expire_after_seconds
        }
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes/ttl",
            body=body
        )
        return response.get("ok", False)

    def create_partial_index(
        self,
        field: str,
        filter_expr: Dict[str, Any],
        unique: bool = False,
        name: Optional[str] = None
    ) -> bool:
        """
        Create a partial index that indexes only documents matching a filter.

        Args:
            field: Field name to index
            filter_expr: Filter expression for partial indexing
            unique: Whether the index should enforce uniqueness
            name: Optional index name

        Returns:
            True if successful

        Example:
            >>> users.create_partial_index(
            ...     'email',
            ...     filter_expr={'active': True},
            ...     unique=True
            ... )
        """
        body = {
            "field": field,
            "filterExpression": filter_expr,
            "unique": unique
        }
        if name:
            body["name"] = name

        response = self.client._request(
            "POST",
            f"{self._base_path}/indexes/partial",
            body=body
        )
        return response.get("ok", False)

    def list_indexes(self) -> List[Dict[str, Any]]:
        """
        List all indexes in the collection.

        Returns:
            List of index information

        Example:
            >>> indexes = users.list_indexes()
            >>> for idx in indexes:
            ...     print(idx['name'])
        """
        response = self.client._request(
            "GET",
            f"{self._base_path}/indexes"
        )
        return response.get("result", {}).get("indexes", [])

    def drop_index(self, name: str) -> bool:
        """
        Drop an index by name.

        Args:
            name: Index name

        Returns:
            True if successful

        Example:
            >>> users.drop_index('email_1')
        """
        response = self.client._request(
            "DELETE",
            f"{self._base_path}/indexes/{name}"
        )
        return response.get("ok", False)

    def stats(self) -> Dict[str, Any]:
        """
        Get collection statistics.

        Returns:
            Dictionary containing collection statistics

        Example:
            >>> stats = users.stats()
            >>> print(stats['documentCount'])
        """
        response = self.client._request(
            "GET",
            f"{self._base_path}/stats"
        )
        return response.get("result", {})

    def __repr__(self) -> str:
        """String representation of the collection."""
        return f"Collection(name='{self.name}')"
