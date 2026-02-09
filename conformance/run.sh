#!/usr/bin/env bash
# Скрипт полного цикла conformance-тестирования dephealth SDK
#
# Использование:
#   ./run.sh [--lang LANG] [--scenario SCENARIO] [--deploy-mode MODE]
#
# Опции:
#   --lang          Язык SDK: go|python|java|csharp|all (по умолчанию: go)
#   --scenario      Запустить только один сценарий (имя файла без расширения)
#   --metrics-url   URL метрик тестового сервиса (по умолчанию: через port-forward)
#   --deploy-mode   Режим деплоя: helm|kubectl (по умолчанию: helm)
#   --namespace     Kubernetes namespace (по умолчанию: dephealth-conformance)
#   --helm-values   Дополнительный values-файл для Helm

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${NAMESPACE:-dephealth-conformance}"
SINGLE_SCENARIO=""
METRICS_URL=""
PORT_FORWARD_PID=""
LANG_SDK="go"

# Флаг инициализации (cleanup не выполняется до начала деплоя)
DEPLOYED=false

# Режим деплоя: helm (по умолчанию) или kubectl (legacy)
DEPLOY_MODE="${DEPLOY_MODE:-helm}"
HELM_CHART_DIR="${HELM_CHART_DIR:-$(cd "$SCRIPT_DIR/../deploy/helm/dephealth-conformance" 2>/dev/null && pwd || echo "")}"
HELM_VALUES="${HELM_VALUES:-}"
HELM_RELEASE="dephealth-conformance"

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

    # Не выполнять cleanup, если деплой ещё не начался (например, --help)
    if [ "$DEPLOYED" != true ]; then
        return
    fi

    if [ "$DEPLOY_MODE" = "helm" ]; then
        log_info "Удаление Helm-релиза $HELM_RELEASE"
        helm uninstall "$HELM_RELEASE" -n "$NAMESPACE" --wait 2>/dev/null || true
    fi
    log_info "Удаление namespace $NAMESPACE"
    kubectl delete namespace "$NAMESPACE" --ignore-not-found --timeout=60s || true
}

trap cleanup EXIT

# Парсинг аргументов
while [[ $# -gt 0 ]]; do
    case $1 in
        --lang) LANG_SDK="$2"; shift 2 ;;
        --scenario) SINGLE_SCENARIO="$2"; shift 2 ;;
        --metrics-url) METRICS_URL="$2"; shift 2 ;;
        --deploy-mode) DEPLOY_MODE="$2"; shift 2 ;;
        --namespace) NAMESPACE="$2"; shift 2 ;;
        --helm-values) HELM_VALUES="$2"; shift 2 ;;
        --help|-h)
            echo "Использование: $0 [опции]"
            echo ""
            echo "Опции:"
            echo "  --lang LANG          Язык SDK: go|python|java|csharp|all (по умолчанию: go)"
            echo "  --scenario NAME      Запустить только один сценарий (имя без .yml)"
            echo "  --metrics-url URL    URL метрик тестового сервиса"
            echo "  --deploy-mode MODE   Режим деплоя: helm|kubectl (по умолчанию: helm)"
            echo "  --namespace NS       Kubernetes namespace (по умолчанию: dephealth-conformance)"
            echo "  --helm-values FILE   Дополнительный values-файл для Helm"
            echo "  --help, -h           Показать справку"
            exit 0
            ;;
        *) log_error "Неизвестный аргумент: $1"; exit 1 ;;
    esac
done

# Валидация --lang
SUPPORTED_LANGS="go python java csharp all"
if ! echo "$SUPPORTED_LANGS" | grep -qw "$LANG_SDK"; then
    log_error "Неизвестный язык: $LANG_SDK (допустимые: $SUPPORTED_LANGS)"
    exit 1
fi

# Валидация --deploy-mode
if [ "$DEPLOY_MODE" != "helm" ] && [ "$DEPLOY_MODE" != "kubectl" ]; then
    log_error "Неизвестный режим деплоя: $DEPLOY_MODE (допустимые: helm, kubectl)"
    exit 1
fi

# Валидация Helm-чарта
if [ "$DEPLOY_MODE" = "helm" ] && [ -z "$HELM_CHART_DIR" ]; then
    log_error "Helm-чарт не найден. Укажите HELM_CHART_DIR или используйте --deploy-mode kubectl"
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

get_metrics_path() {
    local lang="$1"
    case "$lang" in
        java) echo "/actuator/prometheus" ;;
        *)    echo "/metrics" ;;
    esac
}

# --- Деплой ---

# Деплой через Helm (инфраструктура + тестовые сервисы в одном чарте)
deploy_helm() {
    log_info "Деплой через Helm: релиз=$HELM_RELEASE, чарт=$HELM_CHART_DIR"

    # Создать namespace, если не существует
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    local helm_args=("upgrade" "--install" "$HELM_RELEASE" "$HELM_CHART_DIR"
        "--namespace" "$NAMESPACE"
        "--set" "global.namespace=$NAMESPACE"
        "--wait" "--timeout" "5m")

    if [ -n "$HELM_VALUES" ]; then
        helm_args+=("--values" "$HELM_VALUES")
    fi

    helm "${helm_args[@]}"
    log_info "Helm-релиз $HELM_RELEASE установлен"
}

# Деплой инфраструктуры через kubectl (legacy)
deploy_infra_kubectl() {
    log_info "Деплой инфраструктуры через kubectl в namespace $NAMESPACE"
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

# Деплой тестового сервиса через kubectl (legacy)
deploy_test_service_kubectl() {
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

# Ожидание readiness тестового сервиса (для helm-режима сервис уже развёрнут)
wait_for_service() {
    local lang="$1"
    local deployment
    deployment="$(get_deployment_name "$lang")"

    log_info "Ожидание deployment/$deployment..."
    kubectl -n "$NAMESPACE" rollout status "deployment/$deployment" --timeout=120s
}

# Запуск тестовых сценариев для конкретного языка
run_lang_tests() {
    local lang="$1"
    local metrics_url="$2"
    local svc_name
    svc_name="$(get_service_name "$lang")"
    local deployment_name
    deployment_name="$(get_deployment_name "$lang")"

    local local_url="$metrics_url"

    # Port-forward, если URL не задан
    if [ -z "$local_url" ]; then
        local port=$((8888 + RANDOM % 1000))
        local metrics_path
        metrics_path="$(get_metrics_path "$lang")"
        log_info "Запуск port-forward к $svc_name (порт $port)..."
        kubectl -n "$NAMESPACE" port-forward "svc/$svc_name" "$port:8080" &
        PORT_FORWARD_PID=$!
        sleep 3
        local_url="http://localhost:$port${metrics_path}"
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
            --metrics-url "$local_url" \
            --namespace "$NAMESPACE" \
            --pod-label "$deployment_name"; then
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

log_info "Режим деплоя: $DEPLOY_MODE"
log_info "Namespace: $NAMESPACE"

# Определяем список языков для прогона
if [ "$LANG_SDK" = "all" ]; then
    LANGS="go python java csharp"
else
    LANGS="$LANG_SDK"
fi

# 1. Деплой инфраструктуры и сервисов
DEPLOYED=true
if [ "$DEPLOY_MODE" = "helm" ]; then
    deploy_helm
else
    deploy_infra_kubectl
fi

# 2. Для каждого языка: деплой сервиса (если kubectl) + тесты
TOTAL_FAILED=0

for lang in $LANGS; do
    log_info "========== Тестирование SDK: $lang =========="

    if [ "$DEPLOY_MODE" = "kubectl" ]; then
        # kubectl-режим: деплоим сервисы по одному
        if [ "$lang" = "go" ]; then
            kubectl apply -f "$SCRIPT_DIR/k8s/test-service/"
            log_info "  Ожидание deployment/conformance-test-service..."
            kubectl -n "$NAMESPACE" rollout status "deployment/conformance-test-service" --timeout=120s
        else
            if ! deploy_test_service_kubectl "$lang"; then
                log_warn "Пропуск $lang: тестовый сервис не найден"
                continue
            fi
        fi
    else
        # helm-режим: сервисы уже развёрнуты, ждём readiness
        wait_for_service "$lang"
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
