"""HealthChecker interface and check error types."""

from __future__ import annotations

from typing import Protocol

from dephealth.dependency import Endpoint


class CheckError(Exception):
    """Base dependency check error."""


class CheckTimeoutError(CheckError):
    """Check timed out."""


class CheckConnectionRefusedError(CheckError):
    """Connection refused."""


class UnhealthyError(CheckError):
    """Dependency is reachable but unhealthy."""


class HealthChecker(Protocol):
    """Protocol for dependency health checking."""

    async def check(self, endpoint: Endpoint) -> None:
        """Check the dependency. Raises CheckError on failure."""
        ...

    def checker_type(self) -> str:
        """Return the checker type (http, grpc, tcp, ...)."""
        ...
