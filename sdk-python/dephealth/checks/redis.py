"""Redis health checker."""

from __future__ import annotations

from typing import Any

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class RedisChecker:
    """Проверка доступности Redis через PING."""

    def __init__(
        self,
        timeout: float = 5.0,
        password: str = "",
        db: int = 0,
        client: Any = None,  # noqa: ANN401
    ) -> None:
        self._timeout = timeout
        self._password = password
        self._db = db
        self._client = client

    async def check(self, endpoint: Endpoint) -> None:
        """Выполняет PING на Redis."""
        try:
            from redis.asyncio import Redis
        except ImportError:
            msg = "redis is required for Redis checker"
            raise ImportError(msg) from None

        if self._client is not None:
            await self._client.ping()
            return

        url = f"redis://{endpoint.host}:{endpoint.port}/{self._db}"
        client = Redis.from_url(
            url,
            password=self._password or None,
            socket_timeout=self._timeout,
            socket_connect_timeout=self._timeout,
        )
        try:
            await client.ping()
        except TimeoutError:
            msg = f"Redis connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from None
        except OSError as e:
            msg = f"Redis connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e
        finally:
            await client.aclose()

    def checker_type(self) -> str:
        return "redis"
