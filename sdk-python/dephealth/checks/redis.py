"""Redis health checker."""

from __future__ import annotations

from typing import Any

from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckTimeoutError,
    UnhealthyError,
)
from dephealth.dependency import Endpoint


class RedisChecker:
    """Health check for Redis via PING."""

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
        """Execute PING on Redis."""
        try:
            from redis.asyncio import Redis
        except ImportError:
            msg = "redis is required for Redis checker"
            raise ImportError(msg) from None

        if self._client is not None:
            try:
                await self._client.ping()
            except Exception as e:
                _classify_redis_error(e, endpoint)
                raise
            return

        conn_url = self._url or f"redis://{endpoint.host}:{endpoint.port}/{self._db}"
        kwargs: dict[str, Any] = {
            "socket_timeout": self._timeout,
            "socket_connect_timeout": self._timeout,
        }
        # Explicit password takes priority over URL.
        if self._password:
            kwargs["password"] = self._password
        elif not self._url:
            kwargs["password"] = None
        client = Redis.from_url(conn_url, **kwargs)
        try:
            result = await client.ping()
            if result is not True:
                msg = f"Redis PING to {endpoint.host}:{endpoint.port} returned {result!r}"
                raise UnhealthyError(msg)
        except (CheckAuthError, CheckTimeoutError, CheckConnectionRefusedError, UnhealthyError):
            raise
        except TimeoutError as exc:
            msg = f"Redis connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"Redis connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e
        except Exception as e:
            _classify_redis_error(e, endpoint)
            raise
        finally:
            await client.aclose()

    def checker_type(self) -> str:
        """Return the checker type."""
        return "redis"


def _classify_redis_error(err: Exception, endpoint: Endpoint) -> None:
    """Re-raise Redis auth errors as CheckAuthError."""
    msg = str(err)
    if "NOAUTH" in msg or "WRONGPASS" in msg or "AUTH" in msg:
        raise CheckAuthError(f"Redis auth error at {endpoint.host}:{endpoint.port}: {msg}") from err
