#!/usr/bin/env python3
"""Кросс-языковая верификация идентичности метрик dephealth SDK.

Снимает метрики со всех 4 SDK-сервисов и проверяет:
- Имена метрик (app_dependency_health, app_dependency_latency_seconds)
- Метки (name, dependency, type, host, port, critical + custom)
- HELP-строки
- Бакеты histogram
- Формат Prometheus text format
"""

import argparse
import json
import sys

import requests
from prometheus_client.parser import text_string_to_metric_families

HEALTH_METRIC = "app_dependency_health"
LATENCY_METRIC = "app_dependency_latency_seconds"
REQUIRED_LABELS = {"name", "dependency", "type", "host", "port", "critical"}
EXPECTED_BUCKETS = {0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}

EXPECTED_HELP = {
    HEALTH_METRIC: "Health status of a dependency (1 = healthy, 0 = unhealthy)",
    LATENCY_METRIC: "Latency of dependency health check in seconds",
}

EXPECTED_TYPES = {
    HEALTH_METRIC: "gauge",
    LATENCY_METRIC: "histogram",
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
    for name in [HEALTH_METRIC, LATENCY_METRIC]:
        if name not in metrics:
            info[name] = None
            continue

        family = metrics[name]
        labels_set = set()
        buckets = set()
        dep_names = set()
        dep_types = set()

        for sample in family["samples"]:
            sample_labels = set(sample["labels"].keys()) - {"le"}
            labels_set.update(sample_labels)

            if sample["labels"].get("dependency"):
                dep_names.add(sample["labels"]["dependency"])
            if sample["labels"].get("type"):
                dep_types.add(sample["labels"]["type"])

            if sample["name"] == f"{name}_bucket" and "le" in sample["labels"]:
                le_val = sample["labels"]["le"]
                if le_val != "+Inf":
                    buckets.add(float(le_val))

        info[name] = {
            "help": family["documentation"],
            "type": family["type"],
            "labels": sorted(labels_set),
            "buckets": sorted(buckets) if buckets else [],
            "dependencies": sorted(dep_names),
            "dep_types": sorted(dep_types),
        }

    return info


def compare_sdks(sdk_data: dict[str, dict]) -> list[str]:
    """Сравнить данные метрик между SDK. Возвращает список ошибок."""
    errors = []
    langs = list(sdk_data.keys())

    for metric_name in [HEALTH_METRIC, LATENCY_METRIC]:
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
        except Exception as e:
            print(f"  ОШИБКА: {e}")
            sdk_data[lang] = {}

    print("\n" + "=" * 60)
    print("Кросс-языковое сравнение")
    print("=" * 60)

    errors = compare_sdks(sdk_data)

    if errors:
        print(f"\nНайдено {len(errors)} расхождений:")
        for err in errors:
            print(f"  [-] {err}")
        sys.exit(1)
    else:
        print("\n  [+] Все метрики идентичны между SDK")
        print("  [+] Имена метрик совпадают")
        print("  [+] HELP-строки совпадают")
        print("  [+] Типы метрик совпадают")
        print("  [+] Обязательные метки присутствуют")
        print("  [+] Бакеты histogram совпадают")
        print("  [+] Зависимости совпадают")
        print("  [+] Типы зависимостей совпадают")

    if args.json:
        print("\n" + json.dumps(sdk_data, indent=2, default=str))


if __name__ == "__main__":
    main()
