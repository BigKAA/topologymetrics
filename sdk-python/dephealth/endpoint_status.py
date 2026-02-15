"""EndpointStatus: detailed health check state for a single endpoint."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime


@dataclass(frozen=True)
class EndpointStatus:
    """Detailed health check state for a single endpoint.

    Returned by health_details(). Contains all 11 fields defined in spec section 8.
    """

    healthy: bool | None
    status: str
    detail: str
    latency: float
    type: str
    name: str
    host: str
    port: str
    critical: bool
    last_checked_at: datetime | None
    labels: dict[str, str] = field(default_factory=dict)

    def latency_millis(self) -> float:
        """Return the latency in milliseconds."""
        return self.latency * 1000.0

    def to_dict(self) -> dict[str, object]:
        """Serialize to a JSON-compatible dictionary.

        Latency is serialized as ``latency_ms`` (milliseconds float).
        LastCheckedAt is serialized as ISO 8601 string or None.
        """
        return {
            "healthy": self.healthy,
            "status": self.status,
            "detail": self.detail,
            "latency_ms": self.latency_millis(),
            "type": self.type,
            "name": self.name,
            "host": self.host,
            "port": self.port,
            "critical": self.critical,
            "last_checked_at": (self.last_checked_at.isoformat() if self.last_checked_at else None),
            "labels": dict(self.labels),
        }


__all__ = [
    "EndpointStatus",
]
