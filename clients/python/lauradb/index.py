"""
LauraDB Index - Index management utilities
"""
from typing import Any, Dict, List, Optional


class Index:
    """
    Index management utilities and configuration.

    This class provides constants and helper methods for working with indexes.

    Example:
        >>> from lauradb import Index
        >>> Index.BTREE
        'btree'
    """

    # Index types
    BTREE = "btree"
    COMPOUND = "compound"
    TEXT = "text"
    GEO_2D = "2d"
    GEO_2DSPHERE = "2dsphere"
    TTL = "ttl"
    PARTIAL = "partial"

    @staticmethod
    def create_btree_config(
        field: str,
        unique: bool = False,
        sparse: bool = False,
        name: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for B+ tree index.

        Args:
            field: Field name to index
            unique: Whether the index should enforce uniqueness
            sparse: Whether to index only documents with the field
            name: Optional index name

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_btree_config('email', unique=True)
        """
        config = {
            "field": field,
            "unique": unique,
            "sparse": sparse
        }
        if name:
            config["name"] = name
        return config

    @staticmethod
    def create_compound_config(
        fields: List[str],
        unique: bool = False,
        name: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for compound index.

        Args:
            fields: List of field names
            unique: Whether the index should enforce uniqueness
            name: Optional index name

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_compound_config(['city', 'age'])
        """
        config = {
            "fields": fields,
            "unique": unique
        }
        if name:
            config["name"] = name
        return config

    @staticmethod
    def create_text_config(
        fields: List[str],
        name: Optional[str] = None,
        weights: Optional[Dict[str, int]] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for text index.

        Args:
            fields: List of text field names
            name: Optional index name
            weights: Optional field weights for relevance scoring

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_text_config(
            ...     ['title', 'content'],
            ...     weights={'title': 10, 'content': 1}
            ... )
        """
        config = {"fields": fields}
        if name:
            config["name"] = name
        if weights:
            config["weights"] = weights
        return config

    @staticmethod
    def create_geo_config(
        field: str,
        geo_type: str = "2d",
        name: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for geospatial index.

        Args:
            field: Field name containing coordinates
            geo_type: Index type ('2d' for planar, '2dsphere' for spherical)
            name: Optional index name

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_geo_config('location', geo_type='2dsphere')
        """
        config = {
            "field": field,
            "geoType": geo_type
        }
        if name:
            config["name"] = name
        return config

    @staticmethod
    def create_ttl_config(
        field: str,
        expire_after_seconds: int,
        name: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for TTL index.

        Args:
            field: Field name containing timestamp
            expire_after_seconds: Seconds after which documents expire
            name: Optional index name

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_ttl_config('createdAt', 3600)
        """
        config = {
            "field": field,
            "expireAfterSeconds": expire_after_seconds
        }
        if name:
            config["name"] = name
        return config

    @staticmethod
    def create_partial_config(
        field: str,
        filter_expr: Dict[str, Any],
        unique: bool = False,
        name: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Create configuration for partial index.

        Args:
            field: Field name to index
            filter_expr: Filter expression for partial indexing
            unique: Whether the index should enforce uniqueness
            name: Optional index name

        Returns:
            Index configuration dictionary

        Example:
            >>> config = Index.create_partial_config(
            ...     'email',
            ...     filter_expr={'active': True},
            ...     unique=True
            ... )
        """
        config = {
            "field": field,
            "filterExpression": filter_expr,
            "unique": unique
        }
        if name:
            config["name"] = name
        return config

    @staticmethod
    def analyze_query_plan(explain_result: Dict[str, Any]) -> str:
        """
        Analyze query execution plan and provide recommendations.

        Args:
            explain_result: Result from query explain

        Returns:
            Human-readable analysis

        Example:
            >>> plan = collection.find({'age': 25}, explain=True)
            >>> analysis = Index.analyze_query_plan(plan)
            >>> print(analysis)
        """
        lines = []

        if not explain_result:
            return "No explain data available"

        # Execution stats
        if "executionStats" in explain_result:
            stats = explain_result["executionStats"]
            lines.append(f"Execution Time: {stats.get('executionTimeMs', 0)}ms")
            lines.append(f"Documents Examined: {stats.get('documentsExamined', 0)}")
            lines.append(f"Documents Returned: {stats.get('documentsReturned', 0)}")

        # Index usage
        if "indexUsed" in explain_result:
            index = explain_result["indexUsed"]
            if index:
                lines.append(f"Index Used: {index.get('name', 'unknown')}")
            else:
                lines.append("Index Used: None (Collection scan)")
                lines.append("⚠️  Consider adding an index for better performance")

        # Query optimization
        if "optimized" in explain_result:
            if explain_result["optimized"]:
                lines.append("✓ Query is optimized")
            else:
                lines.append("⚠️  Query could be optimized")

        # Covered query
        if "coveredQuery" in explain_result:
            if explain_result["coveredQuery"]:
                lines.append("✓ Covered query (index-only)")

        return "\n".join(lines)
