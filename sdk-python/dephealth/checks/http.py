"""HTTP health checker."""

from __future__ import annotations

from typing import Any

import aiohttp

from dephealth.checker import CheckConnectionRefusedError, CheckTimeoutError, UnhealthyError
from dephealth.dependency import Endpoint


class HTTPChecker:
    """Health check via HTTP GET to the health endpoint."""

    def __init__(
        self,
        health_path: str = "/health",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
    ) -> None:
        self._health_path = health_path
        self._timeout = timeout
        self._tls = tls
        self._tls_skip_verify = tls_skip_verify

    async def check(self, endpoint: Endpoint) -> None:
        """Perform an HTTP GET and verify a 2xx response."""
        scheme = "https" if self._tls else "http"
        url = f"{scheme}://{endpoint.host}:{endpoint.port}{self._health_path}"

        connector_kwargs: dict[str, Any] = {}
        if self._tls_skip_verify:
            import ssl

            ctx = ssl.create_default_context()
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
            connector_kwargs["ssl"] = ctx

        timeout = aiohttp.ClientTimeout(total=self._timeout)
        try:
            async with (
                aiohttp.ClientSession(
                    timeout=timeout,
                    connector=aiohttp.TCPConnector(**connector_kwargs),
                ) as session,
                session.get(url) as resp,
            ):
                if resp.status < 200 or resp.status >= 300:
                    msg = f"HTTP {resp.status} from {url}"
                    raise UnhealthyError(msg)
        except TimeoutError as exc:
            msg = f"HTTP request to {url} timed out"
            raise CheckTimeoutError(msg) from exc
        except aiohttp.ClientConnectorError as e:
            msg = f"HTTP connection to {url} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        """Return the checker type."""
        return "http"
