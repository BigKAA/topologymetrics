#!/usr/bin/env bash
# Скрипт полного цикла conformance-тестирования dephealth SDK
#
# Использование:
#   ./run.sh [--lang LANG] [--no-cleanup] [--scenario SCENARIO]
#
# Опции:
#   --lang          Язык SDK: go|python|java|csharp|all (по умолчанию: go)
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
LANG_SDK="go"

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
        --lang) LANG_SDK="$2"; shift 2 ;;
        --no-cleanup) CLEANUP=false; shift ;;
        --scenario) SINGLE_SCENARIO="$2"; shift 2 ;;
        --metrics-url) METRICS_URL="$2"; shift 2 ;;
        *) log_error "Неизвестный аргумент: $1"; exit 1 ;;
    esac
done

# Валидация --lang
SUPPORTED_LANGS="go python java csharp all"
if ! echo "$SUPPORTED_LANGS" | grep -qw "$LANG_SDK"; then
    log_error "Неизвестный язык: $LANG_SDK (допустимые: $SUPPORTED_LANGS)"
    exit 1
fi

# --- Функции для конкретных языков ---

# Определение каталога тестового сервиса и deployment-имени для языка
get_test_service_dir() {
    local lang="$1"
    case "$lang" in
        go) echo "$SCRIPT_DIR/test-service" ;;
        *)  echo "$SCRIPT_DIR/test-service-${lang}" ;;
    esac
}

get_deployment_name() {
    local lang="$1"
    case "$lang" in
        go) echo "conformance-test-service" ;;
        *)  echo "conformance-test-service-${lang}" ;;
    esac
}

get_service_name() {
    local lang="$1"
    case "$lang" in
        go) echo "conformance-test-service" ;;
        *)  echo "conformance-test-service-${lang}" ;;
    esac
}

# Деплой общей инфраструктуры (БД, кэши, стабы)
deploy_infra() {
    log_info "Деплой инфраструктуры в namespace $NAMESPACE"
    kubectl apply -f "$SCRIPT_DIR/k8s/namespace.yml"
    kubectl apply -f "$SCRIPT_DIR/k8s/postgres/"
    kubectl apply -f "$SCRIPT_DIR/k8s/redis/"
    kubectl apply -f "$SCRIPT_DIR/k8s/rabbitmq/"
    kubectl apply -f "$SCRIPT_DIR/k8s/kafka/"
    kubectl apply -f "$SCRIPT_DIR/k8s/stubs/"

    log_info "Ожидание readiness инфраструктуры..."
    for resource in \
        "statefulset/postgres-primary" \
        "statefulset/postgres-replica" \
        "deployment/redis" \
        "deployment/rabbitmq" \
        "statefulset/kafka" \
        "deployment/http-stub" \
        "deployment/grpc-stub"; do
        log_info "  Ожидание $resource..."
        kubectl -n "$NAMESPACE" rollout status "$resource" --timeout=120s
    done
    log_info "Инфраструктура готова"
}

# Деплой тестового сервиса для конкретного языка
deploy_test_service() {
    local lang="$1"
    local svc_dir
    svc_dir="$(get_test_service_dir "$lang")"

    if [ ! -d "$svc_dir/k8s" ]; then
        log_error "Каталог тестового сервиса не найден: $svc_dir/k8s"
        return 1
    fi

    log_info "Деплой тестового сервиса ($lang): $svc_dir/k8s/"
    kubectl apply -f "$svc_dir/k8s/"

    local deployment
    deployment="$(get_deployment_name "$lang")"
    log_info "  Ожидание deployment/$deployment..."
    kubectl -n "$NAMESPACE" rollout status "deployment/$deployment" --timeout=120s
}

# Запуск тестовых сценариев для конкретного языка
run_lang_tests() {
    local lang="$1"
    local metrics_url="$2"
    local svc_name
    svc_name="$(get_service_name "$lang")"

    local local_url="$metrics_url"

    # Port-forward, если URL не задан
    if [ -z "$local_url" ]; then
        local port=$((8888 + RANDOM % 1000))
        log_info "Запуск port-forward к $svc_name (порт $port)..."
        kubectl -n "$NAMESPACE" port-forward "svc/$svc_name" "$port:8080" &
        PORT_FORWARD_PID=$!
        sleep 3
        local_url="http://localhost:$port/metrics"
        log_info "METRICS_URL=$local_url (pid=$PORT_FORWARD_PID)"
    fi

    # Запуск сценариев
    local scenarios_dir="$SCRIPT_DIR/scenarios"
    local passed=0 failed=0 total=0

    run_scenario() {
        local scenario_file="$1"
        local name
        name="$(basename "$scenario_file" .yml)"

        log_info "[$lang] Запуск сценария: $name"
        total=$((total + 1))

        if python3 "$SCRIPT_DIR/runner/verify.py" \
            --scenario "$scenario_file" \
            --metrics-url "$local_url"; then
            passed=$((passed + 1))
            log_info "[$lang] Сценарий $name: ${GREEN}PASSED${NC}"
        else
            failed=$((failed + 1))
            log_error "[$lang] Сценарий $name: ${RED}FAILED${NC}"
        fi
        echo ""
    }

    if [ -n "$SINGLE_SCENARIO" ]; then
        local scenario_file="$scenarios_dir/${SINGLE_SCENARIO}.yml"
        if [ ! -f "$scenario_file" ]; then
            log_error "Сценарий не найден: $scenario_file"
            return 1
        fi
        run_scenario "$scenario_file"
    else
        for scenario_file in "$scenarios_dir"/*.yml; do
            run_scenario "$scenario_file"
        done
    fi

    # Завершение port-forward
    if [ -n "$PORT_FORWARD_PID" ]; then
        kill "$PORT_FORWARD_PID" 2>/dev/null || true
        PORT_FORWARD_PID=""
    fi

    echo "========================================"
    log_info "[$lang] Итого: $passed passed, $failed failed из $total"
    echo "========================================"

    return "$failed"
}

# --- Основная логика ---

# Определяем список языков для прогона
if [ "$LANG_SDK" = "all" ]; then
    LANGS="go python java csharp"
else
    LANGS="$LANG_SDK"
fi

# 1. Деплой инфраструктуры (общая для всех языков)
deploy_infra

# 2. Для каждого языка: деплой сервиса + тесты
TOTAL_FAILED=0

for lang in $LANGS; do
    log_info "========== Тестирование SDK: $lang =========="

    # Деплой тестового сервиса (для go используем существующий k8s/test-service/)
    if [ "$lang" = "go" ]; then
        kubectl apply -f "$SCRIPT_DIR/k8s/test-service/"
        log_info "  Ожидание deployment/conformance-test-service..."
        kubectl -n "$NAMESPACE" rollout status "deployment/conformance-test-service" --timeout=120s
    else
        if ! deploy_test_service "$lang"; then
            log_warn "Пропуск $lang: тестовый сервис не найден"
            continue
        fi
    fi

    # Запуск тестов
    if ! run_lang_tests "$lang" "$METRICS_URL"; then
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
    fi

    echo ""
done

# 3. Итоги
if [ "$TOTAL_FAILED" -gt 0 ]; then
    log_error "Есть неудачные языки: $TOTAL_FAILED"
    exit 1
fi

log_info "Все тесты пройдены успешно"
