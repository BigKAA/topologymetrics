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
import subprocess
import sys
import time
from dataclasses import dataclass

import requests
import yaml
from prometheus_client.parser import text_string_to_metric_families

logger = logging.getLogger("conformance.verify")

# Ожидаемые имена метрик из спецификации
HEALTH_METRIC = "app_dependency_health"
LATENCY_METRIC = "app_dependency_latency_seconds"
REQUIRED_LABELS = {"dependency", "type", "host", "port"}
EXPECTED_BUCKETS = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]
VALID_TYPES = {"http", "grpc", "tcp", "postgres", "mysql", "redis", "amqp", "kafka"}

HELP_HEALTH = "Health status of a dependency (1 = healthy, 0 = unhealthy)"
HELP_LATENCY = "Latency of dependency health check in seconds"


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

    for sample in metrics[name]["samples"]:
        # Пропускаем _bucket, _sum, _count — у них есть доп. метки (le)
        base_name = name
        if sample["name"].endswith(("_bucket", "_sum", "_count")):
            base_name = sample["name"].rsplit("_", 1)[0]
            if base_name != name:
                continue

        labels = set(sample["labels"].keys()) - {"le"}
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
        dep = labels.get("dependency", "")
        dep_type = labels.get("type", "")
        port = labels.get("port", "")

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

    # Построить lookup: (dependency, host, port) -> value
    actual = {}
    for sample in metrics[HEALTH_METRIC]["samples"]:
        if sample["name"] != HEALTH_METRIC:
            continue
        key = (
            sample["labels"].get("dependency"),
            sample["labels"].get("host"),
            sample["labels"].get("port"),
        )
        actual[key] = sample["value"]

    for exp in expected:
        key = (exp["dependency"], exp["host"], str(exp["port"]))
        exp_value = float(exp["value"])

        if key not in actual:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                False,
                f"метрика не найдена для {key}",
            ))
        elif actual[key] != exp_value:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                False,
                f"значение {actual[key]}, ожидалось {exp_value}",
            ))
        else:
            results.append(CheckResult(
                f"dep_{exp['dependency']}_{exp['host']}",
                True,
                f"OK: {exp['dependency']} = {exp_value}",
            ))

    return results


def execute_action(action: dict, namespace: str = "dephealth-conformance") -> None:
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
             "-l", "app=conformance-test-service",
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


def execute_actions(actions: list[dict], label: str) -> None:
    """Выполнить список действий (pre_actions или post_actions)."""
    if not actions:
        return
    logger.info("выполнение %s (%d действий)", label, len(actions))
    for i, action in enumerate(actions, 1):
        logger.info("  [%d/%d] %s", i, len(actions), action.get("type", "?"))
        execute_action(action)


def load_scenario(path: str) -> dict:
    """Загрузить сценарий из YAML-файла."""
    with open(path) as f:
        return yaml.safe_load(f)


def run_scenario(scenario: dict, metrics_url: str) -> list[CheckResult]:
    """Выполнить все проверки из сценария."""
    results = []

    # Выполнить pre_actions
    pre_actions = scenario.get("pre_actions", [])
    if pre_actions:
        try:
            execute_actions(pre_actions, "pre_actions")
        except Exception as e:
            return [CheckResult("pre_actions", False, f"ошибка pre_actions: {e}")]

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

    results = run_scenario(scenario, args.metrics_url)

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
            execute_actions(post_actions, "post_actions")
        except Exception as e:
            logger.error("ошибка post_actions: %s", e)

    if failed > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
