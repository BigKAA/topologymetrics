"""Error classification for health check results."""

from __future__ import annotations

import socket
import ssl
from dataclasses import dataclass

# Status category constants.
STATUS_OK = "ok"
STATUS_TIMEOUT = "timeout"
STATUS_CONNECTION_ERROR = "connection_error"
STATUS_DNS_ERROR = "dns_error"
STATUS_AUTH_ERROR = "auth_error"
STATUS_TLS_ERROR = "tls_error"
STATUS_UNHEALTHY = "unhealthy"
STATUS_ERROR = "error"
STATUS_UNKNOWN = "unknown"

ALL_STATUS_CATEGORIES = (
    STATUS_OK,
    STATUS_TIMEOUT,
    STATUS_CONNECTION_ERROR,
    STATUS_DNS_ERROR,
    STATUS_AUTH_ERROR,
    STATUS_TLS_ERROR,
    STATUS_UNHEALTHY,
    STATUS_ERROR,
)


@dataclass(frozen=True)
class CheckResult:
    """Classification of a health check outcome."""

    category: str
    detail: str


def classify_error(err: BaseException | None) -> CheckResult:
    """Classify a check error into a status category and detail.

    Classification chain:
    1. CheckError with status_category/status_detail properties
    2. Standard library error types (TimeoutError, socket errors, SSL errors)
    3. Fallback → error/error
    """
    if err is None:
        return CheckResult(category=STATUS_OK, detail=STATUS_OK)

    # 1. Classified check errors (imports deferred to avoid circular import).
    from dephealth.checker import (
        CheckAuthError,
        CheckConnectionRefusedError,
        CheckDnsError,
        CheckError,
        CheckTimeoutError,
        CheckTlsError,
        UnhealthyError,
    )

    if isinstance(err, CheckError) and err.status_category != STATUS_ERROR:
        return CheckResult(category=err.status_category, detail=err.status_detail)

    # Check typed error subclasses.
    if isinstance(err, CheckTimeoutError):
        return CheckResult(category=STATUS_TIMEOUT, detail="timeout")
    if isinstance(err, CheckConnectionRefusedError):
        return CheckResult(category=STATUS_CONNECTION_ERROR, detail="connection_refused")
    if isinstance(err, CheckDnsError):
        return CheckResult(category=STATUS_DNS_ERROR, detail="dns_error")
    if isinstance(err, CheckAuthError):
        return CheckResult(category=STATUS_AUTH_ERROR, detail="auth_error")
    if isinstance(err, CheckTlsError):
        return CheckResult(category=STATUS_TLS_ERROR, detail="tls_error")
    if isinstance(err, UnhealthyError):
        return CheckResult(category=STATUS_UNHEALTHY, detail=err.status_detail)

    # 2. Platform error detection.
    if isinstance(err, TimeoutError):
        return CheckResult(category=STATUS_TIMEOUT, detail="timeout")
    if isinstance(err, asyncio_timeout_error()):
        return CheckResult(category=STATUS_TIMEOUT, detail="timeout")
    if isinstance(err, socket.gaierror):
        return CheckResult(category=STATUS_DNS_ERROR, detail="dns_error")
    if isinstance(err, ConnectionRefusedError):
        return CheckResult(category=STATUS_CONNECTION_ERROR, detail="connection_refused")
    if isinstance(err, ConnectionError):
        return CheckResult(category=STATUS_CONNECTION_ERROR, detail="connection_refused")
    if isinstance(err, ssl.SSLError):
        return CheckResult(category=STATUS_TLS_ERROR, detail="tls_error")
    if isinstance(err, ssl.SSLCertVerificationError):
        return CheckResult(category=STATUS_TLS_ERROR, detail="tls_error")

    # Check wrapped exceptions.
    cause = getattr(err, "__cause__", None) or getattr(err, "__context__", None)
    if cause is not None and cause is not err:
        inner = classify_error(cause)
        if inner.category != STATUS_ERROR:
            return inner

    # 3. Fallback.
    return CheckResult(category=STATUS_ERROR, detail="error")


def asyncio_timeout_error() -> type:
    """Return asyncio.TimeoutError (same as TimeoutError in 3.11+)."""
    import asyncio

    return asyncio.TimeoutError


__all__ = [
    "ALL_STATUS_CATEGORIES",
    "STATUS_AUTH_ERROR",
    "STATUS_CONNECTION_ERROR",
    "STATUS_DNS_ERROR",
    "STATUS_ERROR",
    "STATUS_OK",
    "STATUS_TIMEOUT",
    "STATUS_TLS_ERROR",
    "STATUS_UNHEALTHY",
    "STATUS_UNKNOWN",
    "CheckResult",
    "classify_error",
]
