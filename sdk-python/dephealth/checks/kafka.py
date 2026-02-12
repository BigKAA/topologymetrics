"""Kafka health checker."""

from __future__ import annotations

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class KafkaChecker:
    """Health check for a Kafka broker via cluster metadata retrieval."""

    def __init__(self, timeout: float = 5.0) -> None:
        self._timeout = timeout

    async def check(self, endpoint: Endpoint) -> None:
        """Connect to Kafka and fetch cluster metadata."""
        try:
            from aiokafka import AIOKafkaClient
        except ImportError:
            msg = "aiokafka is required for Kafka checker"
            raise ImportError(msg) from None

        bootstrap = f"{endpoint.host}:{endpoint.port}"
        client = AIOKafkaClient(
            bootstrap_servers=bootstrap,
            request_timeout_ms=int(self._timeout * 1000),
        )
        try:
            await client.bootstrap()
            if not client.cluster.brokers():
                msg = f"Kafka broker {bootstrap}: no brokers in metadata"
                raise CheckConnectionRefusedError(msg)
        except TimeoutError as exc:
            msg = f"Kafka connection to {bootstrap} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"Kafka connection to {bootstrap} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e
        finally:
            await client.close()

    def checker_type(self) -> str:
        """Return the checker type."""
        return "kafka"
