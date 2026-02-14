"""HealthChecker interface and check error types."""

from __future__ import annotations

from typing import Protocol

from dephealth.dependency import Endpoint


class CheckError(Exception):
    """Base dependency check error with optional status classification."""

    def __init__(
        self,
        *args: object,
        status_category: str = "error",
        status_detail: str = "error",
    ) -> None:
        super().__init__(*args)
        self._status_category = status_category
        self._status_detail = status_detail

    @property
    def status_category(self) -> str:
        """Return the status category for this error."""
        return self._status_category

    @property
    def status_detail(self) -> str:
        """Return the detail value for this error."""
        return self._status_detail


class CheckTimeoutError(CheckError):
    """Check timed out."""

    def __init__(self, *args: object) -> None:
        super().__init__(*args, status_category="timeout", status_detail="timeout")


class CheckConnectionRefusedError(CheckError):
    """Connection refused."""

    def __init__(self, *args: object) -> None:
        super().__init__(
            *args,
            status_category="connection_error",
            status_detail="connection_refused",
        )


class CheckDnsError(CheckError):
    """DNS resolution failure."""

    def __init__(self, *args: object) -> None:
        super().__init__(*args, status_category="dns_error", status_detail="dns_error")


class CheckAuthError(CheckError):
    """Authentication/authorization failure."""

    def __init__(self, *args: object) -> None:
        super().__init__(*args, status_category="auth_error", status_detail="auth_error")


class CheckTlsError(CheckError):
    """TLS/SSL error."""

    def __init__(self, *args: object) -> None:
        super().__init__(*args, status_category="tls_error", status_detail="tls_error")


class UnhealthyError(CheckError):
    """Dependency is reachable but unhealthy."""

    def __init__(self, *args: object, detail: str = "unhealthy") -> None:
        super().__init__(*args, status_category="unhealthy", status_detail=detail)


class HealthChecker(Protocol):
    """Protocol for dependency health checking."""

    async def check(self, endpoint: Endpoint) -> None:
        """Check the dependency. Raises CheckError on failure."""
        ...

    def checker_type(self) -> str:
        """Return the checker type (http, grpc, tcp, ...)."""
        ...
