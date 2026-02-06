"""MySQL health checker."""

from __future__ import annotations

from typing import Any

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class MySQLChecker:
    """Проверка доступности MySQL через SELECT 1."""

    def __init__(
        self,
        timeout: float = 5.0,
        query: str = "SELECT 1",
        pool: Any = None,  # noqa: ANN401
    ) -> None:
        self._timeout = timeout
        self._query = query
        self._pool = pool

    async def check(self, endpoint: Endpoint) -> None:
        """Выполняет SELECT 1 на MySQL."""
        try:
            import aiomysql
        except ImportError:
            msg = "aiomysql is required for MySQL checker"
            raise ImportError(msg) from None

        if self._pool is not None:
            async with self._pool.acquire() as conn, conn.cursor() as cur:
                await cur.execute(self._query)
            return

        try:
            conn = await aiomysql.connect(
                host=endpoint.host,
                port=int(endpoint.port),
                connect_timeout=self._timeout,
            )
            try:
                async with conn.cursor() as cur:
                    await cur.execute(self._query)
            finally:
                conn.close()
        except TimeoutError:
            msg = f"MySQL connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from None
        except OSError as e:
            msg = f"MySQL connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        return "mysql"
