"""Tests for authentication support in HTTP and gRPC checkers."""

from __future__ import annotations

import base64
from collections.abc import AsyncIterator

import pytest
from aiohttp import web

from dephealth.checker import CheckAuthError, UnhealthyError
from dephealth.checks.grpc import GRPCChecker, _validate_grpc_auth
from dephealth.checks.http import HTTPChecker, _validate_auth
from dephealth.dependency import Endpoint

# ---------------------------------------------------------------------------
# HTTP auth validation (sync)
# ---------------------------------------------------------------------------


class TestHTTPAuthValidation:
    """Test HTTP auth conflict validation."""

    def test_no_auth_no_error(self) -> None:
        _validate_auth(None, None, None)

    def test_single_bearer_ok(self) -> None:
        _validate_auth(None, "token", None)

    def test_single_basic_auth_ok(self) -> None:
        _validate_auth(None, None, ("user", "pass"))

    def test_single_authorization_header_ok(self) -> None:
        _validate_auth({"Authorization": "Custom"}, None, None)

    def test_bearer_with_non_auth_header_ok(self) -> None:
        _validate_auth({"X-Custom": "value"}, "token", None)

    def test_conflict_bearer_and_basic_auth(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_auth(None, "token", ("user", "pass"))

    def test_conflict_bearer_and_authorization_header(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_auth({"Authorization": "Custom"}, "token", None)

    def test_conflict_basic_auth_and_authorization_header(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_auth({"Authorization": "Custom"}, None, ("user", "pass"))

    def test_authorization_case_insensitive(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_auth({"authorization": "Custom"}, "token", None)


class TestHTTPCheckerConstructor:
    """Test HTTPChecker construction with auth params."""

    def test_bearer_token_builds_header(self) -> None:
        checker = HTTPChecker(bearer_token="my-token")
        assert checker._headers == {"Authorization": "Bearer my-token"}

    def test_basic_auth_builds_header(self) -> None:
        checker = HTTPChecker(basic_auth=("admin", "password"))
        expected = base64.b64encode(b"admin:password").decode("ascii")
        assert checker._headers == {"Authorization": f"Basic {expected}"}

    def test_custom_headers_passed(self) -> None:
        checker = HTTPChecker(headers={"X-API-Key": "my-key"})
        assert checker._headers == {"X-API-Key": "my-key"}

    def test_no_auth_empty_headers(self) -> None:
        checker = HTTPChecker()
        assert checker._headers == {}

    def test_conflict_raises_value_error(self) -> None:
        with pytest.raises(ValueError, match="conflicting"):
            HTTPChecker(bearer_token="token", basic_auth=("u", "p"))


# ---------------------------------------------------------------------------
# gRPC auth validation (sync)
# ---------------------------------------------------------------------------


class TestGRPCAuthValidation:
    """Test gRPC auth conflict validation."""

    def test_no_auth_no_error(self) -> None:
        _validate_grpc_auth(None, None, None)

    def test_single_bearer_ok(self) -> None:
        _validate_grpc_auth(None, "token", None)

    def test_single_basic_auth_ok(self) -> None:
        _validate_grpc_auth(None, None, ("user", "pass"))

    def test_single_authorization_metadata_ok(self) -> None:
        _validate_grpc_auth({"authorization": "Custom"}, None, None)

    def test_bearer_with_non_auth_metadata_ok(self) -> None:
        _validate_grpc_auth({"x-custom": "value"}, "token", None)

    def test_conflict_bearer_and_basic_auth(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_grpc_auth(None, "token", ("user", "pass"))

    def test_conflict_bearer_and_authorization_metadata(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_grpc_auth({"authorization": "Custom"}, "token", None)

    def test_conflict_basic_auth_and_authorization_metadata(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_grpc_auth({"authorization": "Custom"}, None, ("user", "pass"))

    def test_authorization_case_insensitive(self) -> None:
        with pytest.raises(ValueError, match="conflicting auth methods"):
            _validate_grpc_auth({"Authorization": "Custom"}, "token", None)


class TestGRPCCheckerConstructor:
    """Test GRPCChecker construction with auth params."""

    def test_bearer_token_builds_metadata(self) -> None:
        checker = GRPCChecker(bearer_token="my-token")
        assert ("authorization", "Bearer my-token") in checker._metadata

    def test_basic_auth_builds_metadata(self) -> None:
        checker = GRPCChecker(basic_auth=("admin", "password"))
        expected = base64.b64encode(b"admin:password").decode("ascii")
        assert ("authorization", f"Basic {expected}") in checker._metadata

    def test_custom_metadata_passed(self) -> None:
        checker = GRPCChecker(metadata={"x-api-key": "my-key"})
        assert ("x-api-key", "my-key") in checker._metadata

    def test_no_auth_empty_metadata(self) -> None:
        checker = GRPCChecker()
        assert checker._metadata == []

    def test_conflict_raises_value_error(self) -> None:
        with pytest.raises(ValueError, match="conflicting"):
            GRPCChecker(bearer_token="token", basic_auth=("u", "p"))


# ---------------------------------------------------------------------------
# HTTP auth integration tests (async, with aiohttp test server)
# ---------------------------------------------------------------------------


async def _start_http_server(
    handler: web.RequestHandler,
) -> tuple[web.AppRunner, int]:
    """Start an aiohttp server and return (runner, port)."""
    app = web.Application()
    app.router.add_get("/health", handler)
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "127.0.0.1", 0)
    await site.start()
    # Retrieve the bound port.
    port: int = site._server.sockets[0].getsockname()[1]  # type: ignore[union-attr]
    return runner, port


@pytest.fixture()
async def http_server_factory() -> AsyncIterator[object]:
    """Factory fixture that starts a server and cleans up after test."""
    runners: list[web.AppRunner] = []

    async def _factory(handler: web.RequestHandler) -> tuple[web.AppRunner, int]:
        runner, port = await _start_http_server(handler)
        runners.append(runner)
        return runner, port

    yield _factory

    for r in runners:
        await r.cleanup()


@pytest.mark.asyncio()
async def test_http_bearer_token_success(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """Bearer token is sent and server returns 200."""

    async def handler(request: web.Request) -> web.Response:
        auth = request.headers.get("Authorization", "")
        if auth == "Bearer test-token":
            return web.Response(status=200)
        return web.Response(status=401)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker(bearer_token="test-token")
    ep = Endpoint(host="127.0.0.1", port=str(port))
    await checker.check(ep)  # should not raise


@pytest.mark.asyncio()
async def test_http_basic_auth_success(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """Basic Auth is sent and server returns 200."""
    expected_cred = base64.b64encode(b"admin:password").decode("ascii")

    async def handler(request: web.Request) -> web.Response:
        auth = request.headers.get("Authorization", "")
        if auth == f"Basic {expected_cred}":
            return web.Response(status=200)
        return web.Response(status=401)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker(basic_auth=("admin", "password"))
    ep = Endpoint(host="127.0.0.1", port=str(port))
    await checker.check(ep)


@pytest.mark.asyncio()
async def test_http_custom_headers_success(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """Custom headers are sent and server returns 200."""

    async def handler(request: web.Request) -> web.Response:
        if request.headers.get("X-API-Key") == "my-key":
            return web.Response(status=200)
        return web.Response(status=403)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker(headers={"X-API-Key": "my-key"})
    ep = Endpoint(host="127.0.0.1", port=str(port))
    await checker.check(ep)


@pytest.mark.asyncio()
async def test_http_401_raises_auth_error(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """HTTP 401 raises CheckAuthError."""

    async def handler(_: web.Request) -> web.Response:
        return web.Response(status=401)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker()
    ep = Endpoint(host="127.0.0.1", port=str(port))
    with pytest.raises(CheckAuthError) as exc_info:
        await checker.check(ep)
    assert exc_info.value.status_category == "auth_error"
    assert exc_info.value.status_detail == "auth_error"


@pytest.mark.asyncio()
async def test_http_403_raises_auth_error(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """HTTP 403 raises CheckAuthError."""

    async def handler(_: web.Request) -> web.Response:
        return web.Response(status=403)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker()
    ep = Endpoint(host="127.0.0.1", port=str(port))
    with pytest.raises(CheckAuthError) as exc_info:
        await checker.check(ep)
    assert exc_info.value.status_category == "auth_error"


@pytest.mark.asyncio()
async def test_http_500_raises_unhealthy_not_auth(http_server_factory: ...) -> None:  # type: ignore[type-arg]
    """HTTP 500 raises UnhealthyError, not CheckAuthError."""

    async def handler(_: web.Request) -> web.Response:
        return web.Response(status=500)

    _, port = await http_server_factory(handler)  # type: ignore[misc]

    checker = HTTPChecker()
    ep = Endpoint(host="127.0.0.1", port=str(port))
    with pytest.raises(UnhealthyError) as exc_info:
        await checker.check(ep)
    assert exc_info.value.status_category == "unhealthy"
    assert exc_info.value.status_detail == "http_500"
