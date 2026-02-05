"""Утилиты для управления инфраструктурой conformance-тестов в Kubernetes."""

import json
import logging
import subprocess
import time

logger = logging.getLogger("conformance.utils")

NAMESPACE = "dephealth-conformance"


def kubectl(*args: str, namespace: str = NAMESPACE) -> str:
    """Выполнить kubectl-команду и вернуть stdout."""
    cmd = ["kubectl", "-n", namespace, *args]
    logger.debug("exec: %s", " ".join(cmd))
    result = subprocess.run(cmd, capture_output=True, text=True, check=True)
    return result.stdout.strip()


def scale_resource(kind: str, name: str, replicas: int) -> None:
    """Масштабировать Deployment/StatefulSet."""
    kubectl("scale", kind, name, f"--replicas={replicas}")
    logger.info("scaled %s/%s to %d replicas", kind, name, replicas)


def wait_for_ready(kind: str, name: str, timeout: int = 120) -> None:
    """Ожидать readiness ресурса."""
    logger.info("waiting for %s/%s to be ready (timeout=%ds)", kind, name, timeout)
    kubectl("rollout", "status", f"{kind}/{name}", f"--timeout={timeout}s")
    logger.info("%s/%s is ready", kind, name)


def wait_for_pods_ready(label_selector: str, timeout: int = 120) -> None:
    """Ожидать readiness подов по label selector."""
    logger.info("waiting for pods with selector '%s'", label_selector)
    kubectl(
        "wait", "pod",
        f"--selector={label_selector}",
        "--for=condition=Ready",
        f"--timeout={timeout}s",
    )
    logger.info("pods with selector '%s' are ready", label_selector)


def get_pod_name(label_selector: str) -> str:
    """Получить имя первого пода по label selector."""
    output = kubectl(
        "get", "pods",
        f"--selector={label_selector}",
        "-o", "jsonpath={.items[0].metadata.name}",
    )
    return output


def port_forward_start(
    service: str, local_port: int, remote_port: int
) -> subprocess.Popen:
    """Запустить port-forward к сервису (фоновый процесс)."""
    cmd = [
        "kubectl", "-n", NAMESPACE,
        "port-forward", f"svc/{service}",
        f"{local_port}:{remote_port}",
    ]
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    time.sleep(2)  # Дать port-forward время на установку
    logger.info("port-forward %s %d:%d (pid=%d)", service, local_port, remote_port, proc.pid)
    return proc


def wait_for_metric_stabilization(
    metrics_url: str,
    metric_name: str,
    expected_value: float,
    cycles: int = 3,
    interval: float = 5.0,
) -> bool:
    """Ожидать стабилизации значения метрики на протяжении N циклов.

    Возвращает True, если метрика стабилизировалась с ожидаемым значением.
    """
    import requests
    from prometheus_client.parser import text_string_to_metric_families

    stable_count = 0
    for i in range(cycles * 3):  # Максимум 3x попыток
        try:
            resp = requests.get(metrics_url, timeout=10)
            resp.raise_for_status()

            found = False
            for family in text_string_to_metric_families(resp.text):
                if family.name == metric_name:
                    for sample in family.samples:
                        if sample.value == expected_value:
                            found = True
                            break
                    break

            if found:
                stable_count += 1
                logger.debug(
                    "metric %s = %s (stable %d/%d)",
                    metric_name, expected_value, stable_count, cycles,
                )
                if stable_count >= cycles:
                    return True
            else:
                stable_count = 0

        except Exception as e:
            logger.warning("failed to check metric: %s", e)
            stable_count = 0

        time.sleep(interval)

    return False


def get_all_pods_status() -> list[dict]:
    """Получить статус всех подов в namespace."""
    output = kubectl("get", "pods", "-o", "json")
    data = json.loads(output)
    result = []
    for pod in data.get("items", []):
        name = pod["metadata"]["name"]
        phase = pod["status"].get("phase", "Unknown")
        ready = all(
            cs.get("ready", False)
            for cs in pod["status"].get("containerStatuses", [])
        )
        result.append({"name": name, "phase": phase, "ready": ready})
    return result
