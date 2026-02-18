#!/usr/bin/env python3
"""Conformance runner — проверка Prometheus-метрик тестового сервиса.

Запускает сценарий из YAML-файла и проверяет, что метрики
соответствуют спецификации dephealth.

Использование:
    python verify.py --scenario scenarios/basic-health.yml --metrics-url http://localhost:8080/metrics
    python verify.py --scenario scenarios/full-failure.yml --metrics-url http://svc.ns.svc:8080/metrics
"""

import argparse
import logging
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from urllib.parse import urlparse, urlunparse

import requests
import yaml
from prometheus_client.parser import text_string_to_metric_families

logger = logging.getLogger("conformance.verify")

# Ожидаемые имена метрик из спецификации
HEALTH_METRIC = "app_dependency_health"
LATENCY_METRIC = "app_dependency_latency_seconds"
REQUIRED_LABELS = {"name", "group", "dependency", "type", "host", "port", "critical"}
EXPECTED_BUCKETS = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]
VALID_TYPES = {"http", "grpc", "tcp", "postgres", "mysql", "redis", "amqp", "kafka"}

HELP_HEALTH = "Health status of a dependency (1 = healthy, 0 = unhealthy)"
HELP_LATENCY = "Latency of dependency health check in seconds"

STATUS_METRIC = "app_dependency_status"
DETAIL_METRIC = "app_dependency_status_detail"
VALID_STATUSES = {"ok", "timeout", "connection_error", "dns_error",
                  "auth_error", "tls_error", "unhealthy", "error"}
HELP_STATUS = "Category of the last check result"
HELP_DETAIL = "Detailed reason of the last check result"

DETAIL_TO_STATUS = {
    "ok": "ok", "timeout": "timeout",
    "connection_refused": "connection_error", "network_unreachable": "connection_error",
    "host_unreachable": "connection_error", "dns_error": "dns_error",
    "auth_error": "auth_error", "tls_error": "tls_error",
    "unhealthy": "unhealthy", "no_brokers": "unhealthy",
    "grpc_not_serving": "unhealthy", "grpc_unknown": "unhealthy",
    "error": "error", "pool_exhausted": "error", "query_error": "error",
}

VALID_DETAILS_BY_TYPE = {
    "http": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "tls_error", "error"},
    "grpc": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "tls_error",
             "grpc_not_serving", "grpc_unknown", "error"},
    "tcp": {"ok", "timeout", "connection_refused", "dns_error", "error"},
    "postgres": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "tls_error", "error"},
    "mysql": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "tls_error", "error"},
    "redis": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "unhealthy", "error"},
    "amqp": {"ok", "timeout", "connection_refused", "dns_error", "auth_error", "tls_error", "unhealthy", "error"},
    "kafka": {"ok", "timeout", "connection_refused", "dns_error", "no_brokers", "error"},
}


@dataclass
class CheckResult:
    """Результат одной проверки."""

    name: str
    passed: bool
    message: str


def fetch_metrics(url: str, timeout: int = 10) -> str:
    """Получить текст метрик с /metrics."""
    resp = requests.get(url, timeout=timeout)
    resp.raise_for_status()
    return resp.text


def wait_for_metrics_ready(
    metrics_url: str,
    expected_healthy: dict[str, float],
    timeout: int = 60,
    poll_interval: int = 2,
) -> None:
    """Wait until app_dependency_health samples match expected values.

    Polls the metrics endpoint every poll_interval seconds.
    expected_healthy maps dependency name to expected health value (1.0 or 0.0).
    Raises TimeoutError if not all dependencies match within timeout.
    """
    if not expected_healthy:
        return

    deadline = time.monotonic() + timeout
    not_ready: dict[str, str] = {}
    while time.monotonic() < deadline:
        try:
            text = fetch_metrics(metrics_url)
            metrics = parse_metrics(text)
            if HEALTH_METRIC in metrics:
                actual: dict[str, float] = {}
                for sample in metrics[HEALTH_METRIC]["samples"]:
                    if sample["name"] == HEALTH_METRIC:
                        actual[sample["labels"].get("dependency", "")] = sample["value"]

                not_ready = {}
                for dep, expected_val in expected_healthy.items():
                    if dep not in actual:
                        not_ready[dep] = "missing"
                    elif actual[dep] != expected_val:
                        not_ready[dep] = f"health={actual[dep]}, want={expected_val}"

                if not not_ready:
                    logger.info(
                        "all %d expected dependencies ready in metrics",
                        len(expected_healthy),
                    )
                    return
                logger.debug("waiting for dependencies: %s", not_ready)
        except Exception as e:
            logger.debug("metrics not ready yet: %s", e)

        time.sleep(poll_interval)

    raise TimeoutError(
        f"timed out ({timeout}s) waiting for dependencies: {not_ready}"
    )


def parse_metrics(text: str) -> dict:
    """Парсинг Prometheus text format в структуру.

    Возвращает dict: metric_name -> list of {labels: dict, value: float}.
    """
    result = {}
    for family in text_string_to_metric_families(text):
        samples = []
        for sample in family.samples:
            samples.append({
                "name": sample.name,
                "labels": dict(sample.labels),
                "value": sample.value,
            })
        result[family.name] = {
            "type": family.type,
            "documentation": family.documentation,
            "samples": samples,
        }
    return result


def check_metric_exists(metrics: dict, name: str) -> CheckResult:
    """Проверить, что метрика существует."""
    if name in metrics:
        return CheckResult(f"metric_{name}_exists", True, f"метрика {name} найдена")
    return CheckResult(f"metric_{name}_exists", False, f"метрика {name} НЕ найдена")


def check_help_text(metrics: dict, name: str, expected_help: str) -> CheckResult:
    """Проверить текст HELP метрики."""
    if name not in metrics:
        return CheckResult(f"metric_{name}_help", False, f"метрика {name} не найдена")

    actual = metrics[name]["documentation"]
    if actual == expected_help:
        return CheckResult(f"metric_{name}_help", True, "HELP текст корректен")
    return CheckResult(
        f"metric_{name}_help", False,
        f"HELP: ожидалось '{expected_help}', получено '{actual}'",
    )


def check_required_labels(metrics: dict, name: str) -> list[CheckResult]:
    """Проверить наличие обязательных меток у всех samples."""
    results = []
    if name not in metrics:
        results.append(
            CheckResult(f"metric_{name}_labels", False, f"метрика {name} не найдена")
        )
        return results

    # Extra labels depending on metric type
    extra_labels = set()
    if name == STATUS_METRIC:
        extra_labels = {"status"}
    elif name == DETAIL_METRIC:
        extra_labels = {"detail"}

    for sample in metrics[name]["samples"]:
        # Skip _bucket, _sum, _count — they have extra labels (le)
        base_name = name
        if sample["name"].endswith(("_bucket", "_sum", "_count")):
            base_name = sample["name"].rsplit("_", 1)[0]
            if base_name != name:
                continue

        labels = set(sample["labels"].keys()) - {"le"} - extra_labels
        missing = REQUIRED_LABELS - labels
        if missing:
            results.append(CheckResult(
                f"labels_{sample['labels'].get('dependency', '?')}",
                False,
                f"отсутствуют метки: {missing} в {sample['labels']}",
            ))

    if not results:
        results.append(CheckResult(f"metric_{name}_labels", True, "все обязательные метки присутствуют"))

    return results


def check_label_values(metrics: dict, name: str) -> list[CheckResult]:
    """Проверить корректность значений меток."""
    results = []
    if name not in metrics:
        return results

    for sample in metrics[name]["samples"]:
        if sample["name"] != name:
            continue

        labels = sample["labels"]
        instance_name = labels.get("name", "")
        dep = labels.get("dependency", "")
        dep_type = labels.get("type", "")
        port = labels.get("port", "")
        critical = labels.get("critical", "")

        # Проверить формат name ([a-z][a-z0-9-]*, 1-63 символа)
        if instance_name:
            if not re.fullmatch(r"[a-z][a-z0-9-]{0,62}", instance_name):
                results.append(CheckResult(
                    f"label_name_{dep}", False,
                    f"невалидный формат name: '{instance_name}' "
                    f"(ожидается [a-z][a-z0-9-]*, 1-63 символа)",
                ))

        # Проверить формат group ([a-z][a-z0-9-]*, 1-63 символа)
        group_val = labels.get("group", "")
        if group_val:
            if not re.fullmatch(r"[a-z][a-z0-9-]{0,62}", group_val):
                results.append(CheckResult(
                    f"label_group_{dep}", False,
                    f"невалидный формат group: '{group_val}' "
                    f"(ожидается [a-z][a-z0-9-]*, 1-63 символа)",
                ))

        # Проверить значение critical (yes/no)
        if critical and critical not in ("yes", "no"):
            results.append(CheckResult(
                f"label_critical_{dep}", False,
                f"невалидное значение critical: '{critical}' (ожидается 'yes' или 'no')",
            ))

        # Проверить формат dependency name
        if dep and not all(c.isalnum() or c == "-" for c in dep):
            results.append(CheckResult(
                f"label_dependency_{dep}", False,
                f"невалидное имя зависимости: '{dep}'",
            ))

        # Проверить тип
        if dep_type and dep_type not in VALID_TYPES:
            results.append(CheckResult(
                f"label_type_{dep}", False,
                f"неизвестный тип: '{dep_type}'",
            ))

        # Проверить порт
        if port:
            try:
                p = int(port)
                if not 1 <= p <= 65535:
                    raise ValueError
            except ValueError:
                results.append(CheckResult(
                    f"label_port_{dep}", False,
                    f"невалидный порт: '{port}'",
                ))

    if not results:
        results.append(CheckResult("label_values", True, "все значения меток корректны"))

    return results


def check_health_values(metrics: dict) -> list[CheckResult]:
    """Проверить, что значения health-метрики 0 или 1."""
    results = []
    if HEALTH_METRIC not in metrics:
        return results

    for sample in metrics[HEALTH_METRIC]["samples"]:
        if sample["name"] != HEALTH_METRIC:
            continue
        if sample["value"] not in (0.0, 1.0):
            dep = sample["labels"].get("dependency", "?")
            results.append(CheckResult(
                f"health_value_{dep}", False,
                f"значение {sample['value']} (ожидалось 0 или 1)",
            ))

    if not results:
        results.append(CheckResult("health_values", True, "все значения 0 или 1"))

    return results


def check_histogram_buckets(metrics: dict) -> list[CheckResult]:
    """Проверить наличие histogram бакетов."""
    results = []
    if LATENCY_METRIC not in metrics:
        return [CheckResult("histogram_buckets", False, f"метрика {LATENCY_METRIC} не найдена")]

    bucket_name = f"{LATENCY_METRIC}_bucket"
    le_values = set()
    for sample in metrics[LATENCY_METRIC]["samples"]:
        if sample["name"] == bucket_name and "le" in sample["labels"]:
            le_val = sample["labels"]["le"]
            if le_val != "+Inf":
                le_values.add(float(le_val))

    expected_set = set(EXPECTED_BUCKETS)
    if expected_set.issubset(le_values):
        results.append(CheckResult("histogram_buckets", True, f"все бакеты присутствуют: {sorted(le_values)}"))
    else:
        missing = expected_set - le_values
        results.append(CheckResult("histogram_buckets", False, f"отсутствуют бакеты: {sorted(missing)}"))

    return results


def check_expected_dependencies(
    metrics: dict, expected: list[dict]
) -> list[CheckResult]:
    """Проверить ожидаемые зависимости из сценария."""
    results = []
    if HEALTH_METRIC not in metrics:
        results.append(CheckResult("expected_deps", False, f"метрика {HEALTH_METRIC} не найдена"))
        return results

    # Построить lookup: (dependency, host, port) -> {value, name, critical}
    actual = {}
    for sample in metrics[HEALTH_METRIC]["samples"]:
        if sample["name"] != HEALTH_METRIC:
            continue
        key = (
            sample["labels"].get("dependency"),
            sample["labels"].get("host"),
            sample["labels"].get("port"),
        )
        actual[key] = {
            "value": sample["value"],
            "name": sample["labels"].get("name", ""),
            "group": sample["labels"].get("group", ""),
            "critical": sample["labels"].get("critical", ""),
        }

    for exp in expected:
        key = (exp["dependency"], exp["host"], str(exp["port"]))
        exp_value = float(exp["value"])

        if key not in actual:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                False,
                f"метрика не найдена для {key}",
            ))
            continue

        entry = actual[key]

        if entry["value"] != exp_value:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                False,
                f"значение {entry['value']}, ожидалось {exp_value}",
            ))
        else:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                True,
                f"OK: {exp['dependency']} = {exp_value}",
            ))

        # Проверить name, если указано в сценарии
        if "name" in exp and entry["name"] != exp["name"]:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_name",
                False,
                f"name: '{entry['name']}', ожидалось '{exp['name']}'",
            ))

        # Проверить group, если указано в сценарии
        if "group" in exp and entry["group"] != exp["group"]:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_group",
                False,
                f"group: '{entry['group']}', ожидалось '{exp['group']}'",
            ))

        # Проверить critical, если указано в сценарии
        if "critical" in exp and entry["critical"] != exp["critical"]:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_critical",
                False,
                f"critical: '{entry['critical']}', ожидалось '{exp['critical']}'",
            ))

    return results


def _endpoint_key(labels: dict) -> tuple:
    """Build a unique key for an endpoint from labels."""
    return (
        labels.get("dependency", ""),
        labels.get("host", ""),
        labels.get("port", ""),
    )


def check_status_enum_completeness(metrics: dict) -> list[CheckResult]:
    """Check that each endpoint has exactly 8 status series, with exactly one = 1."""
    results = []
    if STATUS_METRIC not in metrics:
        return [CheckResult("status_enum_completeness", False,
                            f"метрика {STATUS_METRIC} не найдена")]

    # Group samples by endpoint
    endpoints: dict[tuple, dict[str, float]] = {}
    for sample in metrics[STATUS_METRIC]["samples"]:
        key = _endpoint_key(sample["labels"])
        status = sample["labels"].get("status", "")
        endpoints.setdefault(key, {})[status] = sample["value"]

    for key, statuses in endpoints.items():
        dep = key[0]
        status_set = set(statuses.keys())
        if status_set != VALID_STATUSES:
            missing = VALID_STATUSES - status_set
            extra = status_set - VALID_STATUSES
            results.append(CheckResult(
                f"status_enum_{dep}", False,
                f"неполный набор status: missing={missing}, extra={extra}",
            ))
            continue

        active = [s for s, v in statuses.items() if v == 1.0]
        if len(active) != 1:
            results.append(CheckResult(
                f"status_enum_{dep}", False,
                f"ожидался ровно 1 активный status, получено {len(active)}: {active}",
            ))
        else:
            results.append(CheckResult(
                f"status_enum_{dep}", True,
                f"8 серий, активный: {active[0]}",
            ))

    if not results:
        results.append(CheckResult("status_enum_completeness", True,
                                   "все endpoint-ы имеют 8 серий status"))
    return results


def check_status_health_consistency(metrics: dict) -> list[CheckResult]:
    """Check that health=1 ↔ status{ok}=1 and health=0 ↔ status{ok}=0."""
    results = []
    if HEALTH_METRIC not in metrics or STATUS_METRIC not in metrics:
        return [CheckResult("status_health_consistency", True,
                            "метрики health или status ещё не появились — пропуск")]

    # Build health lookup
    health_map: dict[tuple, float] = {}
    for sample in metrics[HEALTH_METRIC]["samples"]:
        if sample["name"] != HEALTH_METRIC:
            continue
        key = _endpoint_key(sample["labels"])
        health_map[key] = sample["value"]

    # Build status{ok} lookup
    status_ok_map: dict[tuple, float] = {}
    for sample in metrics[STATUS_METRIC]["samples"]:
        if sample["labels"].get("status") == "ok":
            key = _endpoint_key(sample["labels"])
            status_ok_map[key] = sample["value"]

    for key in health_map:
        dep = key[0]
        health_val = health_map[key]
        status_ok_val = status_ok_map.get(key)

        if status_ok_val is None:
            results.append(CheckResult(
                f"consistency_{dep}", False,
                f"status{{ok}} не найден для {dep}",
            ))
            continue

        if health_val == 1.0 and status_ok_val != 1.0:
            results.append(CheckResult(
                f"consistency_{dep}", False,
                f"health=1 но status{{ok}}={status_ok_val}",
            ))
        elif health_val == 0.0 and status_ok_val != 0.0:
            results.append(CheckResult(
                f"consistency_{dep}", False,
                f"health=0 но status{{ok}}={status_ok_val}",
            ))
        else:
            results.append(CheckResult(
                f"consistency_{dep}", True,
                f"health={health_val} ↔ status{{ok}}={status_ok_val}",
            ))

    if not results:
        results.append(CheckResult("status_health_consistency", True, "consistent"))
    return results


def check_detail_value_always_one(metrics: dict) -> list[CheckResult]:
    """Check that all detail metric values are exactly 1."""
    results = []
    if DETAIL_METRIC not in metrics:
        return [CheckResult("detail_value_always_one", False,
                            f"метрика {DETAIL_METRIC} не найдена")]

    for sample in metrics[DETAIL_METRIC]["samples"]:
        dep = sample["labels"].get("dependency", "?")
        detail = sample["labels"].get("detail", "?")
        if sample["value"] != 1.0:
            results.append(CheckResult(
                f"detail_value_{dep}", False,
                f"detail{{detail={detail}}} = {sample['value']}, ожидалось 1",
            ))

    if not results:
        results.append(CheckResult("detail_value_always_one", True,
                                   "все значения detail-метрики = 1"))
    return results


def check_detail_valid_values(metrics: dict) -> list[CheckResult]:
    """Check that detail values are valid for the checker type."""
    results = []
    if DETAIL_METRIC not in metrics:
        return [CheckResult("detail_valid_values", False,
                            f"метрика {DETAIL_METRIC} не найдена")]

    for sample in metrics[DETAIL_METRIC]["samples"]:
        dep = sample["labels"].get("dependency", "?")
        dep_type = sample["labels"].get("type", "")
        detail = sample["labels"].get("detail", "")

        # http_NNN pattern (e.g. http_503) is valid for http type
        if dep_type == "http" and re.fullmatch(r"http_\d{3}", detail):
            continue

        valid_set = VALID_DETAILS_BY_TYPE.get(dep_type)
        if valid_set is None:
            results.append(CheckResult(
                f"detail_valid_{dep}", False,
                f"неизвестный тип '{dep_type}' для {dep}",
            ))
            continue

        if detail not in valid_set:
            results.append(CheckResult(
                f"detail_valid_{dep}", False,
                f"detail='{detail}' невалидно для типа '{dep_type}'",
            ))

    if not results:
        results.append(CheckResult("detail_valid_values", True,
                                   "все detail-значения валидны"))
    return results


def check_detail_status_mapping(metrics: dict) -> list[CheckResult]:
    """Check that detail→status mapping matches specification."""
    results = []
    if STATUS_METRIC not in metrics or DETAIL_METRIC not in metrics:
        return [CheckResult("detail_status_mapping", False,
                            "метрики status или detail не найдены")]

    # Build active status per endpoint
    active_status: dict[tuple, str] = {}
    for sample in metrics[STATUS_METRIC]["samples"]:
        if sample["value"] == 1.0:
            key = _endpoint_key(sample["labels"])
            active_status[key] = sample["labels"].get("status", "")

    # Build detail per endpoint
    detail_map: dict[tuple, str] = {}
    for sample in metrics[DETAIL_METRIC]["samples"]:
        key = _endpoint_key(sample["labels"])
        detail_map[key] = sample["labels"].get("detail", "")

    for key in detail_map:
        dep = key[0]
        detail = detail_map[key]
        actual_status = active_status.get(key)

        if actual_status is None:
            results.append(CheckResult(
                f"mapping_{dep}", False,
                f"активный status не найден для {dep}",
            ))
            continue

        # http_NNN maps to "error"
        if re.fullmatch(r"http_\d{3}", detail):
            expected_status = "error"
        else:
            expected_status = DETAIL_TO_STATUS.get(detail)

        if expected_status is None:
            results.append(CheckResult(
                f"mapping_{dep}", False,
                f"нет маппинга для detail='{detail}'",
            ))
            continue

        if actual_status != expected_status:
            results.append(CheckResult(
                f"mapping_{dep}", False,
                f"detail='{detail}' → ожидался status='{expected_status}', "
                f"получен '{actual_status}'",
            ))
        else:
            results.append(CheckResult(
                f"mapping_{dep}", True,
                f"detail='{detail}' → status='{expected_status}'",
            ))

    if not results:
        results.append(CheckResult("detail_status_mapping", True,
                                   "все маппинги detail→status корректны"))
    return results


def check_expected_status(
    metrics: dict, expected: list[dict],
) -> list[CheckResult]:
    """Check that specific endpoints have the expected active status."""
    results = []
    if STATUS_METRIC not in metrics:
        return [CheckResult("expected_status", False,
                            f"метрика {STATUS_METRIC} не найдена")]

    # Build active status per (dependency, host, port)
    active_status: dict[tuple, str] = {}
    for sample in metrics[STATUS_METRIC]["samples"]:
        if sample["value"] == 1.0:
            key = _endpoint_key(sample["labels"])
            active_status[key] = sample["labels"].get("status", "")

    for exp in expected:
        key = (exp["dependency"], exp["host"], str(exp["port"]))
        dep = exp["dependency"]
        exp_status = exp["status"]

        actual = active_status.get(key)
        if actual is None:
            results.append(CheckResult(
                f"expected_status_{dep}", False,
                f"активный status не найден для {dep}",
            ))
        elif actual != exp_status:
            results.append(CheckResult(
                f"expected_status_{dep}", False,
                f"status='{actual}', ожидалось '{exp_status}'",
            ))
        else:
            results.append(CheckResult(
                f"expected_status_{dep}", True,
                f"status='{actual}' OK",
            ))

    return results


def check_expected_detail(
    metrics: dict, expected: list[dict],
) -> list[CheckResult]:
    """Check that specific endpoints have the expected detail value."""
    results = []
    if DETAIL_METRIC not in metrics:
        return [CheckResult("expected_detail", False,
                            f"метрика {DETAIL_METRIC} не найдена")]

    # Build detail per (dependency, host, port)
    detail_map: dict[tuple, str] = {}
    for sample in metrics[DETAIL_METRIC]["samples"]:
        key = _endpoint_key(sample["labels"])
        detail_map[key] = sample["labels"].get("detail", "")

    for exp in expected:
        key = (exp["dependency"], exp["host"], str(exp["port"]))
        dep = exp["dependency"]
        exp_detail = exp["detail"]

        actual = detail_map.get(key)
        if actual is None:
            results.append(CheckResult(
                f"expected_detail_{dep}", False,
                f"detail не найден для {dep}",
            ))
        elif actual != exp_detail:
            results.append(CheckResult(
                f"expected_detail_{dep}", False,
                f"detail='{actual}', ожидалось '{exp_detail}'",
            ))
        else:
            results.append(CheckResult(
                f"expected_detail_{dep}", True,
                f"detail='{actual}' OK",
            ))

    return results


# --- HealthDetails API checks ---

HEALTH_DETAILS_REQUIRED_FIELDS = {
    "healthy", "status", "detail", "latency_ms", "type",
    "name", "host", "port", "critical", "last_checked_at", "labels",
}
VALID_STATUS_CATEGORIES = {"ok", "timeout", "connection_error", "dns_error",
                           "auth_error", "tls_error", "unhealthy", "error", "unknown"}


def derive_service_url(metrics_url: str) -> str:
    """Derive service base URL from metrics URL (strip path)."""
    parsed = urlparse(metrics_url)
    return urlunparse((parsed.scheme, parsed.netloc, "", "", "", ""))


def fetch_health_details(metrics_url: str, timeout: int = 10) -> dict:
    """Fetch /health-details JSON from the test service."""
    base_url = derive_service_url(metrics_url)
    url = f"{base_url}/health-details"
    resp = requests.get(url, timeout=timeout)
    resp.raise_for_status()
    return resp.json()


def check_health_details_endpoint_exists(metrics_url: str) -> CheckResult:
    """Check that /health-details returns HTTP 200 with valid JSON."""
    try:
        data = fetch_health_details(metrics_url)
        if not isinstance(data, dict):
            return CheckResult(
                "health_details_endpoint_exists", False,
                f"response is not a JSON object: {type(data).__name__}",
            )
        return CheckResult(
            "health_details_endpoint_exists", True,
            f"/health-details returned {len(data)} endpoints",
        )
    except Exception as e:
        return CheckResult("health_details_endpoint_exists", False, f"error: {e}")


def check_health_details_structure(data: dict) -> list[CheckResult]:
    """Check that each entry has all 11 required fields."""
    results = []
    for key, entry in data.items():
        if not isinstance(entry, dict):
            results.append(CheckResult(
                f"structure_{key}", False,
                f"entry is not an object: {type(entry).__name__}",
            ))
            continue
        missing = HEALTH_DETAILS_REQUIRED_FIELDS - set(entry.keys())
        if missing:
            results.append(CheckResult(
                f"structure_{key}", False,
                f"missing fields: {missing}",
            ))
        else:
            results.append(CheckResult(
                f"structure_{key}", True,
                "all 11 fields present",
            ))
    if not results:
        results.append(CheckResult("health_details_structure", False,
                                   "no endpoints in response"))
    return results


def check_health_details_types(data: dict) -> list[CheckResult]:
    """Check that field types are correct for each entry."""
    results = []
    for key, entry in data.items():
        if not isinstance(entry, dict):
            continue
        errors = []
        # healthy: bool or null
        h = entry.get("healthy")
        if h is not None and not isinstance(h, bool):
            errors.append(f"healthy: expected bool|null, got {type(h).__name__}")
        # status: string
        if not isinstance(entry.get("status"), str):
            errors.append(f"status: expected string, got {type(entry.get('status')).__name__}")
        # detail: string
        if not isinstance(entry.get("detail"), str):
            errors.append(f"detail: expected string, got {type(entry.get('detail')).__name__}")
        # latency_ms: number
        lat = entry.get("latency_ms")
        if not isinstance(lat, (int, float)):
            errors.append(f"latency_ms: expected number, got {type(lat).__name__}")
        # type: string
        if not isinstance(entry.get("type"), str):
            errors.append(f"type: expected string, got {type(entry.get('type')).__name__}")
        # name: string
        if not isinstance(entry.get("name"), str):
            errors.append(f"name: expected string, got {type(entry.get('name')).__name__}")
        # host: string
        if not isinstance(entry.get("host"), str):
            errors.append(f"host: expected string, got {type(entry.get('host')).__name__}")
        # port: string
        if not isinstance(entry.get("port"), str):
            errors.append(f"port: expected string, got {type(entry.get('port')).__name__}")
        # critical: bool
        if not isinstance(entry.get("critical"), bool):
            errors.append(f"critical: expected bool, got {type(entry.get('critical')).__name__}")
        # last_checked_at: string or null
        lca = entry.get("last_checked_at")
        if lca is not None and not isinstance(lca, str):
            errors.append(f"last_checked_at: expected string|null, got {type(lca).__name__}")
        # labels: object
        if not isinstance(entry.get("labels"), dict):
            errors.append(f"labels: expected object, got {type(entry.get('labels')).__name__}")

        if errors:
            results.append(CheckResult(
                f"types_{key}", False,
                "; ".join(errors),
            ))
        else:
            results.append(CheckResult(f"types_{key}", True, "all field types correct"))

    if not results:
        results.append(CheckResult("health_details_types", False,
                                   "no endpoints in response"))
    return results


def check_health_details_consistency(
    data: dict, metrics: dict,
) -> list[CheckResult]:
    """Check that HealthDetails keys match metrics endpoints."""
    results = []

    # Build set of (dependency, host, port) from health metric
    metrics_keys = set()
    if HEALTH_METRIC in metrics:
        for sample in metrics[HEALTH_METRIC]["samples"]:
            if sample["name"] != HEALTH_METRIC:
                continue
            metrics_keys.add((
                sample["labels"].get("dependency", ""),
                sample["labels"].get("host", ""),
                sample["labels"].get("port", ""),
            ))

    # Build set from health-details keys ("dep:host:port")
    details_keys = set()
    for key in data:
        parts = key.rsplit(":", 2)
        if len(parts) == 3:
            details_keys.add(tuple(parts))
        else:
            results.append(CheckResult(
                f"consistency_key_{key}", False,
                f"invalid key format (expected dependency:host:port): '{key}'",
            ))

    # Compare
    missing_in_details = metrics_keys - details_keys
    extra_in_details = details_keys - metrics_keys

    if missing_in_details:
        for mk in missing_in_details:
            results.append(CheckResult(
                f"consistency_{mk[0]}", False,
                f"endpoint {mk} present in metrics but missing in /health-details",
            ))
    if extra_in_details:
        for dk in extra_in_details:
            results.append(CheckResult(
                f"consistency_{dk[0]}", False,
                f"endpoint {dk} present in /health-details but missing in metrics",
            ))

    if not missing_in_details and not extra_in_details and not results:
        results.append(CheckResult(
            "health_details_consistency", True,
            f"all {len(metrics_keys)} endpoints match between metrics and /health-details",
        ))

    # Also check field consistency: health value ↔ healthy bool, status ↔ active status
    if HEALTH_METRIC in metrics:
        health_map: dict[tuple, float] = {}
        for sample in metrics[HEALTH_METRIC]["samples"]:
            if sample["name"] != HEALTH_METRIC:
                continue
            key = _endpoint_key(sample["labels"])
            health_map[key] = sample["value"]

        for hd_key, entry in data.items():
            parts = hd_key.rsplit(":", 2)
            if len(parts) != 3:
                continue
            mk = tuple(parts)
            if mk not in health_map:
                continue
            metric_health = health_map[mk]
            api_healthy = entry.get("healthy")
            if metric_health == 1.0 and api_healthy is not True:
                results.append(CheckResult(
                    f"consistency_health_{mk[0]}", False,
                    f"metric health=1 but /health-details healthy={api_healthy}",
                ))
            elif metric_health == 0.0 and api_healthy is not False:
                results.append(CheckResult(
                    f"consistency_health_{mk[0]}", False,
                    f"metric health=0 but /health-details healthy={api_healthy}",
                ))

    return results


def check_health_details_status_values(data: dict) -> list[CheckResult]:
    """Check that status values match allowed StatusCategory values."""
    results = []
    for key, entry in data.items():
        if not isinstance(entry, dict):
            continue
        status = entry.get("status", "")
        if status not in VALID_STATUS_CATEGORIES:
            results.append(CheckResult(
                f"status_value_{key}", False,
                f"invalid status '{status}' (expected one of {VALID_STATUS_CATEGORIES})",
            ))
        else:
            results.append(CheckResult(
                f"status_value_{key}", True,
                f"status='{status}' OK",
            ))
    if not results:
        results.append(CheckResult("health_details_status_values", False,
                                   "no endpoints in response"))
    return results


def check_health_details_expected(
    data: dict, expected: list[dict],
) -> list[CheckResult]:
    """Check that specific endpoints have expected values."""
    results = []

    for exp in expected:
        dep = exp["dependency"]
        host = exp["host"]
        port = str(exp["port"])
        hd_key = f"{dep}:{host}:{port}"

        if hd_key not in data:
            results.append(CheckResult(
                f"expected_hd_{dep}", False,
                f"endpoint '{hd_key}' not found in /health-details",
            ))
            continue

        entry = data[hd_key]
        entry_errors = []

        # Check each expected field
        if "healthy" in exp:
            exp_healthy = exp["healthy"]
            actual_healthy = entry.get("healthy")
            if actual_healthy != exp_healthy:
                entry_errors.append(
                    f"healthy: {actual_healthy}, expected {exp_healthy}")

        if "status" in exp:
            if entry.get("status") != exp["status"]:
                entry_errors.append(
                    f"status: '{entry.get('status')}', expected '{exp['status']}'")

        if "detail" in exp:
            if entry.get("detail") != exp["detail"]:
                entry_errors.append(
                    f"detail: '{entry.get('detail')}', expected '{exp['detail']}'")

        if "type" in exp:
            if entry.get("type") != exp["type"]:
                entry_errors.append(
                    f"type: '{entry.get('type')}', expected '{exp['type']}'")

        if "critical" in exp:
            if entry.get("critical") != exp["critical"]:
                entry_errors.append(
                    f"critical: {entry.get('critical')}, expected {exp['critical']}")

        if "name" in exp:
            if entry.get("name") != exp["name"]:
                entry_errors.append(
                    f"name: '{entry.get('name')}', expected '{exp['name']}'")

        if entry_errors:
            results.append(CheckResult(
                f"expected_hd_{dep}", False,
                "; ".join(entry_errors),
            ))
        else:
            results.append(CheckResult(
                f"expected_hd_{dep}", True,
                f"{dep} OK",
            ))

    return results


def execute_action(
    action: dict,
    namespace: str = "dephealth-conformance",
    pod_label: str = "conformance-test-service",
) -> None:
    """Выполнить одно действие из pre_actions/post_actions."""
    action_type = action["type"]

    if action_type == "wait":
        seconds = action["seconds"]
        logger.info("ожидание %d секунд...", seconds)
        time.sleep(seconds)

    elif action_type == "scale":
        kind = action["kind"]
        name = action["name"]
        replicas = action["replicas"]
        logger.info("масштабирование %s/%s → %d реплик", kind, name, replicas)
        subprocess.run(
            ["kubectl", "-n", namespace, "scale", kind, name, f"--replicas={replicas}"],
            check=True, capture_output=True, text=True,
        )

    elif action_type == "wait_ready":
        kind = action["kind"]
        name = action["name"]
        timeout = action.get("timeout", 120)
        logger.info("ожидание readiness %s/%s (timeout=%ds)", kind, name, timeout)
        subprocess.run(
            ["kubectl", "-n", namespace, "rollout", "status",
             f"{kind}/{name}", f"--timeout={timeout}s"],
            check=True, capture_output=True, text=True,
        )

    elif action_type == "http_request":
        method = action.get("method", "GET")
        url = action["url"]
        logger.info("HTTP %s %s (через kubectl exec)", method, url)
        # Находим pod test-service
        result = subprocess.run(
            ["kubectl", "-n", namespace, "get", "pods",
             "-l", f"app={pod_label}",
             "-o", "jsonpath={.items[0].metadata.name}"],
            check=True, capture_output=True, text=True,
        )
        pod_name = result.stdout.strip()
        subprocess.run(
            ["kubectl", "-n", namespace, "exec", pod_name, "--",
             "curl", "-s", "-X", method, url],
            check=True, capture_output=True, text=True,
        )

    else:
        logger.warning("неизвестный тип действия: %s", action_type)


def execute_actions(
    actions: list[dict],
    label: str,
    namespace: str = "dephealth-conformance",
    pod_label: str = "conformance-test-service",
) -> None:
    """Выполнить список действий (pre_actions или post_actions)."""
    if not actions:
        return
    logger.info("выполнение %s (%d действий)", label, len(actions))
    for i, action in enumerate(actions, 1):
        logger.info("  [%d/%d] %s", i, len(actions), action.get("type", "?"))
        execute_action(action, namespace=namespace, pod_label=pod_label)


def load_scenario(path: str) -> dict:
    """Загрузить сценарий из YAML-файла."""
    with open(path) as f:
        return yaml.safe_load(f)


def run_scenario(
    scenario: dict,
    metrics_url: str,
    namespace: str = "dephealth-conformance",
    pod_label: str = "conformance-test-service",
) -> list[CheckResult]:
    """Выполнить все проверки из сценария."""
    results = []

    # Determine expected dependency health values from scenario checks
    expected_healthy: dict[str, float] = {}
    for check in scenario.get("checks", []):
        if check.get("type") == "expected_dependencies":
            for dep in check.get("dependencies", []):
                expected_healthy[dep["dependency"]] = float(dep["value"])

    # Выполнить pre_actions
    pre_actions = scenario.get("pre_actions", [])
    if pre_actions:
        try:
            execute_actions(pre_actions, "pre_actions", namespace=namespace, pod_label=pod_label)
        except Exception as e:
            return [CheckResult("pre_actions", False, f"ошибка pre_actions: {e}")]

    # Wait for all dependencies to reach expected health values before running checks
    if expected_healthy:
        try:
            wait_for_metrics_ready(metrics_url, expected_healthy)
        except TimeoutError as e:
            return [CheckResult("wait_for_metrics", False, str(e))]

    # Получить метрики
    try:
        text = fetch_metrics(metrics_url)
    except Exception as e:
        return [CheckResult("fetch_metrics", False, f"не удалось получить метрики: {e}")]

    metrics = parse_metrics(text)

    checks = scenario.get("checks", [])
    for check in checks:
        check_type = check["type"]

        if check_type == "metric_exists":
            results.append(check_metric_exists(metrics, check["metric"]))

        elif check_type == "help_text":
            results.append(check_help_text(metrics, check["metric"], check["expected"]))

        elif check_type == "required_labels":
            results.extend(check_required_labels(metrics, check["metric"]))

        elif check_type == "label_values":
            results.extend(check_label_values(metrics, check["metric"]))

        elif check_type == "health_values":
            results.extend(check_health_values(metrics))

        elif check_type == "histogram_buckets":
            results.extend(check_histogram_buckets(metrics))

        elif check_type == "expected_dependencies":
            results.extend(check_expected_dependencies(metrics, check["dependencies"]))

        elif check_type == "status_enum_completeness":
            results.extend(check_status_enum_completeness(metrics))

        elif check_type == "status_health_consistency":
            results.extend(check_status_health_consistency(metrics))

        elif check_type == "detail_value_always_one":
            results.extend(check_detail_value_always_one(metrics))

        elif check_type == "detail_valid_values":
            results.extend(check_detail_valid_values(metrics))

        elif check_type == "detail_status_mapping":
            results.extend(check_detail_status_mapping(metrics))

        elif check_type == "expected_status":
            results.extend(check_expected_status(metrics, check["endpoints"]))

        elif check_type == "expected_detail":
            results.extend(check_expected_detail(metrics, check["endpoints"]))

        # --- HealthDetails API checks ---

        elif check_type == "health_details_endpoint_exists":
            results.append(check_health_details_endpoint_exists(metrics_url))

        elif check_type == "health_details_structure":
            hd_data = fetch_health_details(metrics_url)
            results.extend(check_health_details_structure(hd_data))

        elif check_type == "health_details_types":
            hd_data = fetch_health_details(metrics_url)
            results.extend(check_health_details_types(hd_data))

        elif check_type == "health_details_consistency":
            hd_data = fetch_health_details(metrics_url)
            results.extend(check_health_details_consistency(hd_data, metrics))

        elif check_type == "health_details_status_values":
            hd_data = fetch_health_details(metrics_url)
            results.extend(check_health_details_status_values(hd_data))

        elif check_type == "health_details_expected":
            hd_data = fetch_health_details(metrics_url)
            results.extend(check_health_details_expected(hd_data, check["endpoints"]))

        else:
            results.append(CheckResult(
                f"unknown_check_{check_type}", False,
                f"неизвестный тип проверки: {check_type}",
            ))

    return results


def main():
    parser = argparse.ArgumentParser(description="dephealth conformance runner")
    parser.add_argument(
        "--scenario", required=True,
        help="Путь к YAML-файлу сценария",
    )
    parser.add_argument(
        "--metrics-url", required=True,
        help="URL endpoint-а /metrics тестового сервиса",
    )
    parser.add_argument(
        "--namespace", default="dephealth-conformance",
        help="Kubernetes namespace (по умолчанию: dephealth-conformance)",
    )
    parser.add_argument(
        "--pod-label", default="conformance-test-service",
        help="Значение лейбла app= для пода, через который выполняются HTTP-запросы (kubectl exec)",
    )
    parser.add_argument(
        "--verbose", "-v", action="store_true",
        help="Подробный вывод",
    )
    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)-5s %(name)s: %(message)s",
    )

    logger.info("загрузка сценария: %s", args.scenario)
    scenario = load_scenario(args.scenario)
    logger.info("сценарий: %s", scenario.get("name", "без имени"))

    results = run_scenario(scenario, args.metrics_url, namespace=args.namespace, pod_label=args.pod_label)

    # Вывод результатов
    passed = 0
    failed = 0
    for r in results:
        symbol = "+" if r.passed else "-"
        print(f"  [{symbol}] {r.name}: {r.message}")
        if r.passed:
            passed += 1
        else:
            failed += 1

    print(f"\nИтого: {passed} passed, {failed} failed из {len(results)}")

    # Выполнить post_actions (всегда, даже при ошибках)
    post_actions = scenario.get("post_actions", [])
    if post_actions:
        try:
            execute_actions(post_actions, "post_actions", namespace=args.namespace, pod_label=args.pod_label)
        except Exception as e:
            logger.error("ошибка post_actions: %s", e)

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
