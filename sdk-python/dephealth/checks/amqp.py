"""AMQP (RabbitMQ) health checker."""

from __future__ import annotations

from dephealth.checker import CheckAuthError, CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class AMQPChecker:
    """Health check for an AMQP broker via connection."""

    def __init__(
        self,
        timeout: float = 5.0,
        url: str = "",
    ) -> None:
        self._timeout = timeout
        self._url = url

    async def check(self, endpoint: Endpoint) -> None:
        """Establish an AMQP connection and close it."""
        try:
            import aio_pika
        except ImportError:
            msg = "aio-pika is required for AMQP checker"
            raise ImportError(msg) from None

        url = self._url or f"amqp://{endpoint.host}:{endpoint.port}/"
        try:
            connection = await aio_pika.connect_robust(url, timeout=self._timeout)
            await connection.close()
        except TimeoutError as exc:
            msg = f"AMQP connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"AMQP connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e
        except Exception as e:
            _classify_amqp_error(e, endpoint)
            raise

    def checker_type(self) -> str:
        """Return the checker type."""
        return "amqp"


def _classify_amqp_error(err: Exception, endpoint: Endpoint) -> None:
    """Re-raise AMQP auth errors as CheckAuthError."""
    msg = str(err)
    if "403" in msg or "ACCESS_REFUSED" in msg:
        raise CheckAuthError(f"AMQP auth error at {endpoint.host}:{endpoint.port}: {msg}") from err
