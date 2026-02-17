"""HTTP health checker."""

from __future__ import annotations

import base64
from typing import Any

import aiohttp

from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckTimeoutError,
    UnhealthyError,
)
from dephealth.dependency import Endpoint


def _validate_auth(
    headers: dict[str, str] | None,
    bearer_token: str | None,
    basic_auth: tuple[str, str] | None,
) -> None:
    """Validate that at most one auth method is configured."""
    methods = 0
    if bearer_token:
        methods += 1
    if basic_auth:
        methods += 1
    if headers:
        for key in headers:
            if key.lower() == "authorization":
                methods += 1
                break
    if methods > 1:
        msg = (
            "conflicting auth methods: specify only one of "
            "bearer_token, basic_auth, or Authorization header"
        )
        raise ValueError(msg)


def _build_headers(
    headers: dict[str, str] | None,
    bearer_token: str | None,
    basic_auth: tuple[str, str] | None,
) -> dict[str, str]:
    """Build the resolved headers map from auth parameters."""
    resolved: dict[str, str] = {}
    if headers:
        resolved.update(headers)
    if bearer_token:
        resolved["Authorization"] = f"Bearer {bearer_token}"
    if basic_auth:
        username, password = basic_auth
        credentials = base64.b64encode(
            f"{username}:{password}".encode(),
        ).decode("ascii")
        resolved["Authorization"] = f"Basic {credentials}"
    return resolved


class HTTPChecker:
    """Health check via HTTP GET to the health endpoint."""

    def __init__(
        self,
        health_path: str = "/health",
        timeout: float = 5.0,
        tls: bool = False,
        tls_skip_verify: bool = False,
        headers: dict[str, str] | None = None,
        bearer_token: str | None = None,
        basic_auth: tuple[str, str] | None = None,
    ) -> None:
        _validate_auth(headers, bearer_token, basic_auth)
        self._health_path = health_path
        self._timeout = timeout
        self._tls = tls
        self._tls_skip_verify = tls_skip_verify
        self._headers = _build_headers(headers, bearer_token, basic_auth)

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

        request_headers = {"User-Agent": "dephealth/0.4.2"}
        request_headers.update(self._headers)

        timeout = aiohttp.ClientTimeout(total=self._timeout)
        try:
            async with (
                aiohttp.ClientSession(
                    timeout=timeout,
                    connector=aiohttp.TCPConnector(**connector_kwargs),
                ) as session,
                session.get(url, headers=request_headers) as resp,
            ):
                if resp.status < 200 or resp.status >= 300:
                    # HTTP 401/403 → auth_error.
                    if resp.status in (401, 403):
                        msg = f"HTTP {resp.status} from {url}"
                        raise CheckAuthError(msg)
                    msg = f"HTTP {resp.status} from {url}"
                    raise UnhealthyError(msg, detail=f"http_{resp.status}")
        except (CheckAuthError, UnhealthyError):
            raise
        except TimeoutError as exc:
            msg = f"HTTP request to {url} timed out"
            raise CheckTimeoutError(msg) from exc
        except aiohttp.ClientConnectorError as e:
            msg = f"HTTP connection to {url} refused: {e}"
            raise CheckConnectionRefusedError(msg) from e

    def checker_type(self) -> str:
        """Return the checker type."""
        return "http"
