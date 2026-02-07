"""dephealth — SDK for monitoring microservice dependencies via Prometheus metrics."""

from dephealth.checker import (
    CheckConnectionRefusedError,
    CheckError,
    CheckTimeoutError,
    HealthChecker,
    UnhealthyError,
)
from dephealth.dependency import (
    CheckConfig,
    Dependency,
    DependencyType,
    Endpoint,
    default_check_config,
)
from dephealth.parser import ParsedConnection, parse_jdbc, parse_params, parse_url

__all__ = [
    "CheckConfig",
    "CheckError",
    "CheckTimeoutError",
    "CheckConnectionRefusedError",
    "Dependency",
    "DependencyType",
    "Endpoint",
    "HealthChecker",
    "ParsedConnection",
    "UnhealthyError",
    "default_check_config",
    "parse_jdbc",
    "parse_params",
    "parse_url",
]
