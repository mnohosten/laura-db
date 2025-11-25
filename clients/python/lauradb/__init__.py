"""
LauraDB Python Client

A Python client library for LauraDB - A MongoDB-like document database.
"""

from .client import Client
from .collection import Collection
from .query import Query
from .aggregation import Aggregation
from .index import Index

__version__ = "1.0.0"
__all__ = [
    "Client",
    "Collection",
    "Query",
    "Aggregation",
    "Index",
]


def create_client(host="localhost", port=8080, **kwargs):
    """
    Create a new LauraDB client with default configuration.

    Args:
        host: Server hostname or IP address (default: 'localhost')
        port: Server port (default: 8080)
        **kwargs: Additional client configuration options

    Returns:
        Client: LauraDB client instance

    Example:
        >>> client = create_client(host='localhost', port=8080)
        >>> users = client.collection('users')
    """
    return Client(host=host, port=port, **kwargs)
