#!/usr/bin/env bash
# Скрипт полного цикла conformance-тестирования dephealth SDK
#
# Использование:
#   ./run.sh [--no-cleanup] [--scenario SCENARIO]
#
# Опции:
#   --no-cleanup    Не удалять инфраструктуру после тестов
#   --scenario      Запустить только один сценарий (имя файла без расширения)
#   --metrics-url   URL метрик тестового сервиса (по умолчанию: через port-forward)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="dephealth-conformance"
CLEANUP=true
SINGLE_SCENARIO=""
METRICS_URL=""
PORT_FORWARD_PID=""

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

cleanup() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        log_info "Завершение port-forward (pid=$PORT_FORWARD_PID)"
        kill "$PORT_FORWARD_PID" 2>/dev/null || true
    fi

    if [ "$CLEANUP" = true ]; then
        log_info "Очистка namespace $NAMESPACE"
        kubectl delete namespace "$NAMESPACE" --ignore-not-found --timeout=60s || true
    else
        log_warn "Оставляем namespace $NAMESPACE (--no-cleanup)"
    fi
}

trap cleanup EXIT

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-cleanup) CLEANUP=false; shift ;;
        --scenario) SINGLE_SCENARIO="$2"; shift 2 ;;
        --metrics-url) METRICS_URL="$2"; shift 2 ;;
        *) log_error "Неизвестный аргумент: $1"; exit 1 ;;
    esac
done

# 1. Деплой инфраструктуры
log_info "Деплой инфраструктуры в namespace $NAMESPACE"
kubectl apply -f "$SCRIPT_DIR/k8s/namespace.yml"
kubectl apply -f "$SCRIPT_DIR/k8s/postgres/"
kubectl apply -f "$SCRIPT_DIR/k8s/redis/"
kubectl apply -f "$SCRIPT_DIR/k8s/rabbitmq/"
kubectl apply -f "$SCRIPT_DIR/k8s/kafka/"
kubectl apply -f "$SCRIPT_DIR/k8s/stubs/"
kubectl apply -f "$SCRIPT_DIR/k8s/test-service/"

# 2. Ожидание readiness всех подов
log_info "Ожидание readiness подов..."

for resource in \
    "statefulset/postgres-primary" \
    "statefulset/postgres-replica" \
    "deployment/redis" \
    "deployment/rabbitmq" \
    "statefulset/kafka" \
    "deployment/http-stub" \
    "deployment/grpc-stub" \
    "deployment/conformance-test-service"; do
    log_info "  Ожидание $resource..."
    kubectl -n "$NAMESPACE" rollout status "$resource" --timeout=120s
done

log_info "Все поды готовы"

# 3. Port-forward к тестовому сервису (если URL не задан)
if [ -z "$METRICS_URL" ]; then
    log_info "Запуск port-forward к conformance-test-service..."
    kubectl -n "$NAMESPACE" port-forward svc/conformance-test-service 8888:8080 &
    PORT_FORWARD_PID=$!
    sleep 3
    METRICS_URL="http://localhost:8888/metrics"
    log_info "METRICS_URL=$METRICS_URL (pid=$PORT_FORWARD_PID)"
fi

# 4. Запуск сценариев
SCENARIOS_DIR="$SCRIPT_DIR/scenarios"
PASSED=0
FAILED=0
TOTAL=0

run_scenario() {
    local scenario_file="$1"
    local name
    name="$(basename "$scenario_file" .yml)"

    log_info "Запуск сценария: $name"
    TOTAL=$((TOTAL + 1))

    if python3 "$SCRIPT_DIR/runner/verify.py" \
        --scenario "$scenario_file" \
        --metrics-url "$METRICS_URL"; then
        PASSED=$((PASSED + 1))
        log_info "Сценарий $name: ${GREEN}PASSED${NC}"
    else
        FAILED=$((FAILED + 1))
        log_error "Сценарий $name: ${RED}FAILED${NC}"
    fi
    echo ""
}

if [ -n "$SINGLE_SCENARIO" ]; then
    scenario_file="$SCENARIOS_DIR/${SINGLE_SCENARIO}.yml"
    if [ ! -f "$scenario_file" ]; then
        log_error "Сценарий не найден: $scenario_file"
        exit 1
    fi
    run_scenario "$scenario_file"
else
    for scenario_file in "$SCENARIOS_DIR"/*.yml; do
        run_scenario "$scenario_file"
    done
fi

# 5. Итоги
echo "========================================"
log_info "Итого: $PASSED passed, $FAILED failed из $TOTAL"
echo "========================================"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
