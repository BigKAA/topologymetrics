"""dephealth — SDK for monitoring microservice dependencies via Prometheus metrics."""

from __future__ import annotations

from dephealth.check_result import (
    ALL_STATUS_CATEGORIES,
    STATUS_AUTH_ERROR,
    STATUS_CONNECTION_ERROR,
    STATUS_DNS_ERROR,
    STATUS_ERROR,
    STATUS_OK,
    STATUS_TIMEOUT,
    STATUS_TLS_ERROR,
    STATUS_UNHEALTHY,
    STATUS_UNKNOWN,
    CheckResult,
    classify_error,
)
from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckDnsError,
    CheckError,
    CheckTimeoutError,
    CheckTlsError,
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
from dephealth.endpoint_status import EndpointStatus
from dephealth.parser import ParsedConnection, parse_jdbc, parse_params, parse_url
from dephealth.scheduler import EndpointNotFoundError

__all__ = [
    "ALL_STATUS_CATEGORIES",
    "CheckAuthError",
    "CheckConfig",
    "CheckConnectionRefusedError",
    "CheckDnsError",
    "CheckError",
    "CheckResult",
    "CheckTimeoutError",
    "CheckTlsError",
    "Dependency",
    "DependencyType",
    "Endpoint",
    "EndpointNotFoundError",
    "EndpointStatus",
    "HealthChecker",
    "LABEL_NAME_PATTERN",
    "ParsedConnection",
    "RESERVED_LABELS",
    "STATUS_AUTH_ERROR",
    "STATUS_CONNECTION_ERROR",
    "STATUS_DNS_ERROR",
    "STATUS_ERROR",
    "STATUS_OK",
    "STATUS_TIMEOUT",
    "STATUS_TLS_ERROR",
    "STATUS_UNHEALTHY",
    "STATUS_UNKNOWN",
    "UnhealthyError",
    "bool_to_yes_no",
    "classify_error",
    "default_check_config",
    "parse_jdbc",
    "parse_params",
    "parse_url",
    "validate_label_name",
    "validate_labels",
]
