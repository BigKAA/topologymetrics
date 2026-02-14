"""PostgreSQL health checker."""

from __future__ import annotations

from typing import Any

from dephealth.checker import CheckAuthError, CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class PostgresChecker:
    """Health check for PostgreSQL via SELECT 1."""

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
        """Execute SELECT 1 on PostgreSQL."""
        try:
            import asyncpg
        except ImportError:
            msg = "asyncpg is required for Postgres checker"
            raise ImportError(msg) from None

        if self._pool is not None:
            # Pool mode: use the existing connection pool.
            async with self._pool.acquire() as conn:
                await conn.fetchval(self._query)
            return

        # Standalone mode: new connection.
        dsn = self._dsn or f"postgresql://{endpoint.host}:{endpoint.port}"
        try:
            conn = await asyncpg.connect(dsn, timeout=self._timeout)
            try:
                await conn.fetchval(self._query)
            finally:
                await conn.close()
        except TimeoutError as exc:
            msg = f"Postgres connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"Postgres connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e
        except Exception as e:
            _classify_postgres_error(e, endpoint)
            raise

    def checker_type(self) -> str:
        """Return the checker type."""
        return "postgres"


def _classify_postgres_error(err: Exception, endpoint: Endpoint) -> None:
    """Re-raise PostgreSQL auth errors as CheckAuthError."""
    msg = str(err)
    if "28000" in msg or "28P01" in msg or "password authentication failed" in msg:
        raise CheckAuthError(
            f"Postgres auth error at {endpoint.host}:{endpoint.port}: {msg}"
        ) from err
