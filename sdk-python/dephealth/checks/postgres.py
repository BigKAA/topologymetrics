"""PostgreSQL health checker."""

from __future__ import annotations

from typing import Any

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class PostgresChecker:
    """Проверка доступности PostgreSQL через SELECT 1."""

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
        """Выполняет SELECT 1 на PostgreSQL."""
        try:
            import asyncpg
        except ImportError:
            msg = "asyncpg is required for Postgres checker"
            raise ImportError(msg) from None

        if self._pool is not None:
            # Pool-режим: используем существующий пул.
            async with self._pool.acquire() as conn:
                await conn.fetchval(self._query)
            return

        # Автономный режим: новое соединение.
        dsn = self._dsn or f"postgresql://{endpoint.host}:{endpoint.port}"
        try:
            conn = await asyncpg.connect(dsn, timeout=self._timeout)
            try:
                await conn.fetchval(self._query)
            finally:
                await conn.close()
        except TimeoutError:
            msg = f"Postgres connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from None
        except OSError as e:
            msg = f"Postgres connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        return "postgres"
