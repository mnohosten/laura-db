"""
LauraDB Client - Main entry point for interacting with LauraDB server
"""
import json
from typing import Dict, Any, Optional, List
from urllib.parse import urljoin
import requests


class Client:
    """
    LauraDB Client for interacting with LauraDB HTTP server.

    Args:
        host: Server hostname or IP address (default: 'localhost')
        port: Server port (default: 8080)
        https: Use HTTPS instead of HTTP (default: False)
        timeout: Request timeout in seconds (default: 30)
        max_connections: Maximum number of connections in the pool (default: 10)

    Example:
        >>> client = Client(host='localhost', port=8080)
        >>> client.ping()
        True
    """

    def __init__(
        self,
        host: str = "localhost",
        port: int = 8080,
        https: bool = False,
        timeout: int = 30,
        max_connections: int = 10,
    ):
        self.host = host
        self.port = port
        self.https = https
        self.timeout = timeout

        # Build base URL
        protocol = "https" if https else "http"
        self.base_url = f"{protocol}://{host}:{port}"

        # Create session with connection pooling
        self.session = requests.Session()
        adapter = requests.adapters.HTTPAdapter(
            pool_connections=max_connections,
            pool_maxsize=max_connections,
            max_retries=0,
        )
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)

        # Set default headers
        self.session.headers.update({
            "Accept": "application/json",
            "User-Agent": "LauraDB-Python-Client/1.0.0",
        })

    def _request(
        self,
        method: str,
        path: str,
        body: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Perform an HTTP request to the LauraDB server.

        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            path: Request path (relative to base URL)
            body: Request body (will be JSON encoded)
            params: URL query parameters

        Returns:
            API response as dictionary

        Raises:
            requests.HTTPError: If the HTTP request fails
            ValueError: If the API returns an error response
        """
        url = urljoin(self.base_url, path)

        try:
            response = self.session.request(
                method=method,
                url=url,
                json=body,
                params=params,
                timeout=self.timeout,
            )
            response.raise_for_status()

            data = response.json()

            # Check API-level errors
            if not data.get("ok", False):
                error_msg = data.get("message") or data.get("error") or "API request failed"
                raise ValueError(f"LauraDB API error: {error_msg}")

            return data

        except requests.exceptions.RequestException as e:
            raise RuntimeError(f"HTTP request failed: {str(e)}") from e

    def ping(self) -> bool:
        """
        Check if the server is reachable and responding.

        Returns:
            True if server is reachable, False otherwise

        Example:
            >>> client.ping()
            True
        """
        try:
            response = self._request("GET", "/ping")
            return response.get("ok", False)
        except Exception:
            return False

    def stats(self) -> Dict[str, Any]:
        """
        Get database statistics.

        Returns:
            Dictionary containing database statistics

        Example:
            >>> stats = client.stats()
            >>> print(stats['collections'])
        """
        response = self._request("GET", "/stats")
        return response.get("result", {})

    def list_collections(self) -> List[str]:
        """
        List all collections in the database.

        Returns:
            List of collection names

        Example:
            >>> collections = client.list_collections()
            >>> print(collections)
            ['users', 'posts', 'comments']
        """
        response = self._request("GET", "/collections")
        return response.get("result", {}).get("collections", [])

    def create_collection(self, name: str) -> bool:
        """
        Create a new collection.

        Args:
            name: Collection name

        Returns:
            True if successful

        Example:
            >>> client.create_collection('users')
            True
        """
        response = self._request("POST", f"/collections/{name}")
        return response.get("ok", False)

    def drop_collection(self, name: str) -> bool:
        """
        Drop (delete) a collection.

        Args:
            name: Collection name

        Returns:
            True if successful

        Example:
            >>> client.drop_collection('users')
            True
        """
        response = self._request("DELETE", f"/collections/{name}")
        return response.get("ok", False)

    def collection(self, name: str) -> "Collection":
        """
        Get a collection object for performing operations.

        Args:
            name: Collection name

        Returns:
            Collection object

        Example:
            >>> users = client.collection('users')
            >>> users.insert_one({'name': 'Alice'})
        """
        from .collection import Collection
        return Collection(self, name)

    def close(self):
        """
        Close the client and release resources.

        Example:
            >>> client.close()
        """
        self.session.close()

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()

    def __repr__(self) -> str:
        """String representation of the client."""
        return f"Client(host='{self.host}', port={self.port})"
