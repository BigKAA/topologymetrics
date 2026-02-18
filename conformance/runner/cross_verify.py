#!/usr/bin/env python3
"""Кросс-языковая верификация идентичности метрик dephealth SDK.

Снимает метрики со всех 4 SDK-сервисов и проверяет:
- Имена метрик (health, latency, status, status_detail)
- Метки (name, dependency, type, host, port, critical + status/detail)
- HELP-строки
- Бакеты histogram
- Status enum полноту (8 серий)
- Формат Prometheus text format
"""

import argparse
import json
import sys
from urllib.parse import urlparse, urlunparse

import requests
from prometheus_client.parser import text_string_to_metric_families

HEALTH_METRIC = "app_dependency_health"
LATENCY_METRIC = "app_dependency_latency_seconds"
STATUS_METRIC = "app_dependency_status"
DETAIL_METRIC = "app_dependency_status_detail"

ALL_METRICS = [HEALTH_METRIC, LATENCY_METRIC, STATUS_METRIC, DETAIL_METRIC]

REQUIRED_LABELS = {"name", "group", "dependency", "type", "host", "port", "critical"}
EXPECTED_BUCKETS = {0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}

VALID_STATUSES = {"ok", "timeout", "connection_error", "dns_error",
                  "auth_error", "tls_error", "unhealthy", "error"}

EXPECTED_HELP = {
    HEALTH_METRIC: "Health status of a dependency (1 = healthy, 0 = unhealthy)",
    LATENCY_METRIC: "Latency of dependency health check in seconds",
    STATUS_METRIC: "Category of the last check result",
    DETAIL_METRIC: "Detailed reason of the last check result",
}

EXPECTED_TYPES = {
    HEALTH_METRIC: "gauge",
    LATENCY_METRIC: "histogram",
    STATUS_METRIC: "gauge",
    DETAIL_METRIC: "gauge",
}


def fetch_and_parse(url: str) -> dict:
    """Получить и распарсить метрики."""
    resp = requests.get(url, timeout=10)
    resp.raise_for_status()
    result = {}
    for family in text_string_to_metric_families(resp.text):
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


def extract_dephealth_info(metrics: dict) -> dict:
    """Извлечь информацию о dephealth-метриках."""
    info = {}
    for name in ALL_METRICS:
        if name not in metrics:
            info[name] = None
            continue

        family = metrics[name]
        labels_set = set()
        buckets = set()
        dep_names = set()
        dep_types = set()
        status_values = set()
        detail_values = set()

        for sample in family["samples"]:
            sample_labels = set(sample["labels"].keys()) - {"le", "status", "detail"}
            labels_set.update(sample_labels)

            if sample["labels"].get("dependency"):
                dep_names.add(sample["labels"]["dependency"])
            if sample["labels"].get("type"):
                dep_types.add(sample["labels"]["type"])

            if sample["name"] == f"{name}_bucket" and "le" in sample["labels"]:
                le_val = sample["labels"]["le"]
                if le_val != "+Inf":
                    buckets.add(float(le_val))

            if "status" in sample["labels"]:
                status_values.add(sample["labels"]["status"])
            if "detail" in sample["labels"]:
                detail_values.add(sample["labels"]["detail"])

        entry = {
            "help": family["documentation"],
            "type": family["type"],
            "labels": sorted(labels_set),
            "buckets": sorted(buckets) if buckets else [],
            "dependencies": sorted(dep_names),
            "dep_types": sorted(dep_types),
        }
        if status_values:
            entry["status_values"] = sorted(status_values)
        if detail_values:
            entry["detail_values"] = sorted(detail_values)

        info[name] = entry

    return info


def compare_sdks(sdk_data: dict[str, dict]) -> list[str]:
    """Сравнить данные метрик между SDK. Возвращает список ошибок."""
    errors = []
    langs = list(sdk_data.keys())

    for metric_name in ALL_METRICS:
        # Проверить наличие метрики
        for lang in langs:
            if sdk_data[lang].get(metric_name) is None:
                errors.append(f"[{lang}] метрика {metric_name} отсутствует")

        present = [l for l in langs if sdk_data[l].get(metric_name) is not None]
        if len(present) < 2:
            continue

        ref_lang = present[0]
        ref = sdk_data[ref_lang][metric_name]

        # HELP-строки
        expected_help = EXPECTED_HELP[metric_name]
        for lang in present:
            actual_help = sdk_data[lang][metric_name]["help"]
            if actual_help != expected_help:
                errors.append(
                    f"[{lang}] {metric_name} HELP: '{actual_help}' != '{expected_help}'"
                )

        # Тип метрики
        expected_type = EXPECTED_TYPES[metric_name]
        for lang in present:
            actual_type = sdk_data[lang][metric_name]["type"]
            if actual_type != expected_type:
                errors.append(
                    f"[{lang}] {metric_name} type: '{actual_type}' != '{expected_type}'"
                )

        # Метки
        for lang in present:
            actual_labels = set(sdk_data[lang][metric_name]["labels"])
            if not REQUIRED_LABELS.issubset(actual_labels):
                missing = REQUIRED_LABELS - actual_labels
                errors.append(
                    f"[{lang}] {metric_name} отсутствуют метки: {missing}"
                )

        # Бакеты (только для histogram)
        if metric_name == LATENCY_METRIC:
            for lang in present:
                actual_buckets = set(sdk_data[lang][metric_name]["buckets"])
                if not EXPECTED_BUCKETS.issubset(actual_buckets):
                    missing = EXPECTED_BUCKETS - actual_buckets
                    errors.append(
                        f"[{lang}] {metric_name} отсутствуют бакеты: {sorted(missing)}"
                    )

        # Status enum completeness (only for STATUS_METRIC)
        if metric_name == STATUS_METRIC:
            for lang in present:
                status_vals = set(sdk_data[lang][metric_name].get("status_values", []))
                if status_vals != VALID_STATUSES:
                    missing = VALID_STATUSES - status_vals
                    extra = status_vals - VALID_STATUSES
                    errors.append(
                        f"[{lang}] {metric_name} status набор неполный: "
                        f"missing={missing}, extra={extra}"
                    )

        # Зависимости (должны совпадать)
        for lang in present[1:]:
            ref_deps = set(ref["dependencies"])
            lang_deps = set(sdk_data[lang][metric_name]["dependencies"])
            if ref_deps != lang_deps:
                errors.append(
                    f"[{lang}] {metric_name} зависимости отличаются от {ref_lang}: "
                    f"extra={lang_deps - ref_deps}, missing={ref_deps - lang_deps}"
                )

        # Типы зависимостей
        for lang in present[1:]:
            ref_types = set(ref["dep_types"])
            lang_types = set(sdk_data[lang][metric_name]["dep_types"])
            if ref_types != lang_types:
                errors.append(
                    f"[{lang}] {metric_name} типы отличаются от {ref_lang}: "
                    f"extra={lang_types - ref_types}, missing={ref_types - lang_types}"
                )

    return errors


HEALTH_DETAILS_REQUIRED_FIELDS = {
    "healthy", "status", "detail", "latency_ms", "type",
    "name", "host", "port", "critical", "last_checked_at", "labels",
}


def derive_service_url(metrics_url: str) -> str:
    """Derive service base URL from metrics URL (strip path)."""
    parsed = urlparse(metrics_url)
    return urlunparse((parsed.scheme, parsed.netloc, "", "", "", ""))


def fetch_health_details(metrics_url: str) -> dict:
    """Fetch /health-details JSON from the test service."""
    base_url = derive_service_url(metrics_url)
    url = f"{base_url}/health-details"
    resp = requests.get(url, timeout=10)
    resp.raise_for_status()
    return resp.json()


def extract_health_details_info(data: dict) -> dict:
    """Extract structural information from HealthDetails response."""
    keys = sorted(data.keys())
    fields_per_entry = set()
    types_per_entry: dict[str, set] = {}
    statuses = set()
    dep_names = set()
    dep_types = set()

    for key, entry in data.items():
        if not isinstance(entry, dict):
            continue
        fields_per_entry.update(entry.keys())
        for field, val in entry.items():
            types_per_entry.setdefault(field, set()).add(type(val).__name__)
        if "status" in entry:
            statuses.add(entry["status"])
        if "name" in entry:
            dep_names.add(entry["name"])
        if "type" in entry:
            dep_types.add(entry["type"])

    return {
        "keys": keys,
        "fields": sorted(fields_per_entry),
        "statuses": sorted(statuses),
        "dependencies": sorted(dep_names),
        "dep_types": sorted(dep_types),
        "endpoint_count": len(data),
    }


def compare_health_details(sdk_hd: dict[str, dict]) -> list[str]:
    """Compare HealthDetails responses across SDKs."""
    errors = []
    langs = list(sdk_hd.keys())
    if len(langs) < 2:
        return errors

    ref_lang = langs[0]
    ref_info = extract_health_details_info(sdk_hd[ref_lang])

    # Check each SDK has /health-details data
    for lang in langs:
        if not sdk_hd[lang]:
            errors.append(f"[{lang}] /health-details returned empty or failed")

    present = [l for l in langs if sdk_hd[l]]
    if len(present) < 2:
        return errors

    ref_lang = present[0]
    ref_data = sdk_hd[ref_lang]
    ref_info = extract_health_details_info(ref_data)

    # Keys must match
    for lang in present[1:]:
        lang_info = extract_health_details_info(sdk_hd[lang])
        if lang_info["keys"] != ref_info["keys"]:
            ref_keys = set(ref_info["keys"])
            lang_keys = set(lang_info["keys"])
            errors.append(
                f"[{lang}] /health-details keys differ from {ref_lang}: "
                f"extra={lang_keys - ref_keys}, missing={ref_keys - lang_keys}"
            )

    # Fields must match (all 11 required)
    for lang in present:
        lang_info = extract_health_details_info(sdk_hd[lang])
        missing = HEALTH_DETAILS_REQUIRED_FIELDS - set(lang_info["fields"])
        if missing:
            errors.append(f"[{lang}] /health-details missing fields: {missing}")

    # Dependencies and types must match
    for lang in present[1:]:
        lang_info = extract_health_details_info(sdk_hd[lang])
        if lang_info["dependencies"] != ref_info["dependencies"]:
            errors.append(
                f"[{lang}] /health-details dependencies differ from {ref_lang}: "
                f"{lang_info['dependencies']} != {ref_info['dependencies']}"
            )
        if lang_info["dep_types"] != ref_info["dep_types"]:
            errors.append(
                f"[{lang}] /health-details dep_types differ from {ref_lang}: "
                f"{lang_info['dep_types']} != {ref_info['dep_types']}"
            )

    # For each common key, compare static fields (name, host, port, type, critical)
    common_keys = set(ref_info["keys"])
    for lang in present[1:]:
        lang_keys = set(extract_health_details_info(sdk_hd[lang])["keys"])
        common_keys &= lang_keys

    for key in sorted(common_keys):
        ref_entry = ref_data[key]
        for lang in present[1:]:
            lang_entry = sdk_hd[lang].get(key, {})
            for field in ("name", "host", "port", "type", "critical"):
                ref_val = ref_entry.get(field)
                lang_val = lang_entry.get(field)
                if ref_val != lang_val:
                    errors.append(
                        f"[{lang}] /health-details key='{key}' "
                        f"field '{field}': '{lang_val}' != '{ref_val}' ({ref_lang})"
                    )

    return errors


def main():
    parser = argparse.ArgumentParser(description="Кросс-языковая верификация метрик")
    parser.add_argument(
        "--urls", required=True, nargs="+",
        help="URL-ы в формате lang=url, например go=http://localhost:8080/metrics",
    )
    parser.add_argument("--json", action="store_true", help="JSON-вывод")
    args = parser.parse_args()

    sdk_urls = {}
    for item in args.urls:
        lang, url = item.split("=", 1)
        sdk_urls[lang] = url

    print(f"Кросс-языковая верификация: {', '.join(sdk_urls.keys())}")
    print("=" * 60)

    sdk_data = {}
    for lang, url in sdk_urls.items():
        print(f"\n[{lang}] Получение метрик: {url}")
        try:
            metrics = fetch_and_parse(url)
            info = extract_dephealth_info(metrics)
            sdk_data[lang] = info
            for metric_name, data in info.items():
                if data is None:
                    print(f"  {metric_name}: ОТСУТСТВУЕТ")
                else:
                    print(f"  {metric_name}:")
                    print(f"    HELP: {data['help']}")
                    print(f"    type: {data['type']}")
                    print(f"    labels: {data['labels']}")
                    if data['buckets']:
                        print(f"    buckets: {data['buckets']}")
                    print(f"    dependencies: {data['dependencies']}")
                    print(f"    dep_types: {data['dep_types']}")
                    if "status_values" in data:
                        print(f"    status_values: {data['status_values']}")
                    if "detail_values" in data:
                        print(f"    detail_values: {data['detail_values']}")
        except Exception as e:
            print(f"  ОШИБКА: {e}")
            sdk_data[lang] = {}

    # --- HealthDetails API cross-verification ---
    print("\n" + "=" * 60)
    print("HealthDetails API — кросс-языковая верификация")
    print("=" * 60)

    sdk_hd: dict[str, dict] = {}
    for lang, url in sdk_urls.items():
        print(f"\n[{lang}] Получение /health-details")
        try:
            hd_data = fetch_health_details(url)
            sdk_hd[lang] = hd_data
            hd_info = extract_health_details_info(hd_data)
            print(f"  endpoints: {hd_info['endpoint_count']}")
            print(f"  keys: {hd_info['keys']}")
            print(f"  fields: {hd_info['fields']}")
            print(f"  dependencies: {hd_info['dependencies']}")
            print(f"  dep_types: {hd_info['dep_types']}")
            print(f"  statuses: {hd_info['statuses']}")
        except Exception as e:
            print(f"  ОШИБКА: {e}")
            sdk_hd[lang] = {}

    # --- Comparison ---
    print("\n" + "=" * 60)
    print("Кросс-языковое сравнение")
    print("=" * 60)

    errors = compare_sdks(sdk_data)
    hd_errors = compare_health_details(sdk_hd)
    errors.extend(hd_errors)

    if errors:
        print(f"\nНайдено {len(errors)} расхождений:")
        for err in errors:
            print(f"  [-] {err}")
        sys.exit(1)
    else:
        print("\n  [+] Все метрики идентичны между SDK")
        print("  [+] Имена метрик совпадают (4 метрики)")
        print("  [+] HELP-строки совпадают")
        print("  [+] Типы метрик совпадают")
        print("  [+] Обязательные метки присутствуют")
        print("  [+] Бакеты histogram совпадают")
        print("  [+] Status enum полнота (8 серий)")
        print("  [+] Зависимости совпадают")
        print("  [+] Типы зависимостей совпадают")
        print("  [+] HealthDetails JSON-структура идентична")
        print("  [+] HealthDetails ключи совпадают")
        print("  [+] HealthDetails статические поля совпадают")

    if args.json:
        output = {"metrics": sdk_data, "health_details": sdk_hd}
        print("\n" + json.dumps(output, indent=2, default=str))


if __name__ == "__main__":
    main()
