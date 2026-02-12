"""TCP health checker."""

from __future__ import annotations

import asyncio

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError
from dephealth.dependency import Endpoint


class TCPChecker:
    """Health check via TCP connection."""

    def __init__(self, timeout: float = 5.0) -> None:
        self._timeout = timeout

    async def check(self, endpoint: Endpoint) -> None:
        """Establish a TCP connection and close it immediately."""
        try:
            reader, writer = await asyncio.wait_for(
                asyncio.open_connection(endpoint.host, int(endpoint.port)),
                timeout=self._timeout,
            )
            writer.close()
            await writer.wait_closed()
        except TimeoutError as exc:
            msg = f"TCP connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc
        except OSError as e:
            msg = f"TCP connection to {endpoint.host}:{endpoint.port} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        """Return the checker type."""
        return "tcp"
