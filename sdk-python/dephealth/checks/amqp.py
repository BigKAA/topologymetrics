"""AMQP (RabbitMQ) health checker."""

from __future__ import annotations

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class AMQPChecker:
    """Проверка доступности AMQP-брокера через подключение."""

    def __init__(
        self,
        timeout: float = 5.0,
        url: str = "",
    ) -> None:
        self._timeout = timeout
        self._url = url

    async def check(self, endpoint: Endpoint) -> None:
        """Устанавливает AMQP-соединение и закрывает."""
        try:
            import aio_pika
        except ImportError:
            msg = "aio-pika is required for AMQP checker"
            raise ImportError(msg) from None

        url = self._url or f"amqp://{endpoint.host}:{endpoint.port}/"
        try:
            connection = await aio_pika.connect_robust(url, timeout=self._timeout)
            await connection.close()
        except TimeoutError:
            msg = f"AMQP connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from None
        except OSError as e:
            msg = f"AMQP connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        return "amqp"
