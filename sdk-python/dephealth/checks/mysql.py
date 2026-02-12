"""MySQL health checker."""

from __future__ import annotations

from typing import Any
from urllib.parse import urlparse

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class MySQLChecker:
    """Health check for MySQL via SELECT 1."""

    def __init__(
        self,
        timeout: float = 5.0,
        query: str = "SELECT 1",
        pool: Any = None,  # noqa: ANN401
        dsn: str = "",
    ) -> None:
        self._timeout = timeout
        self._query = query
        self._pool = pool
        self._dsn = dsn

    async def check(self, endpoint: Endpoint) -> None:
        """Execute SELECT 1 on MySQL."""
        try:
            import aiomysql
        except ImportError:
            msg = "aiomysql is required for MySQL checker"
            raise ImportError(msg) from None

        if self._pool is not None:
            async with self._pool.acquire() as conn, conn.cursor() as cur:
                await cur.execute(self._query)
            return

        kwargs: dict[str, Any] = {
            "host": endpoint.host,
            "port": int(endpoint.port),
            "connect_timeout": self._timeout,
        }
        if self._dsn:
            parsed = urlparse(self._dsn)
            if parsed.username:
                kwargs["user"] = parsed.username
            if parsed.password:
                kwargs["password"] = parsed.password
            if parsed.path and parsed.path != "/":
                kwargs["db"] = parsed.path.lstrip("/")

        try:
            conn = await aiomysql.connect(**kwargs)
            try:
                async with conn.cursor() as cur:
                    await cur.execute(self._query)
            finally:
                conn.close()
        except TimeoutError as exc:
            msg = f"MySQL connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"MySQL connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        """Return the checker type."""
        return "mysql"
