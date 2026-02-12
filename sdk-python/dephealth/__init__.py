"""dephealth — SDK for monitoring microservice dependencies via Prometheus metrics."""

from __future__ import annotations

from dephealth.checker import (
    CheckConnectionRefusedError,
    CheckError,
    CheckTimeoutError,
    HealthChecker,
    UnhealthyError,
)
from dephealth.dependency import (
    LABEL_NAME_PATTERN,
    RESERVED_LABELS,
    CheckConfig,
    Dependency,
    DependencyType,
    Endpoint,
    bool_to_yes_no,
    default_check_config,
    validate_label_name,
    validate_labels,
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
    "LABEL_NAME_PATTERN",
    "ParsedConnection",
    "RESERVED_LABELS",
    "UnhealthyError",
    "bool_to_yes_no",
    "default_check_config",
    "parse_jdbc",
    "parse_params",
    "parse_url",
    "validate_label_name",
    "validate_labels",
]
