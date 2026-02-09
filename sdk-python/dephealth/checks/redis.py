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
        url: str = "",
    ) -> None:
        self._timeout = timeout
        self._password = password
        self._db = db
        self._client = client
        self._url = url

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

        conn_url = self._url or f"redis://{endpoint.host}:{endpoint.port}/{self._db}"
        kwargs: dict[str, Any] = {
            "socket_timeout": self._timeout,
            "socket_connect_timeout": self._timeout,
        }
        # Явный password имеет приоритет над URL.
        if self._password:
            kwargs["password"] = self._password
        elif not self._url:
            kwargs["password"] = None
        client = Redis.from_url(conn_url, **kwargs)
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
