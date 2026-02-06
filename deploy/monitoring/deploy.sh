#!/usr/bin/env bash
# Деплой стека мониторинга dephealth в Kubernetes.
#
# Использование:
#   ./deploy.sh          — полный деплой
#   ./deploy.sh teardown — удаление всего стека
#
# Предварительные требования:
#   - kubectl настроен на целевой кластер
#   - StorageClass nfs-client доступен
#   - Gateway eg в envoy-gateway-system настроен
#   - DNS: grafana.kryukov.lan → 192.168.218.180

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NAMESPACE="dephealth-monitoring"

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

wait_for_ready() {
    local resource="$1"
    local timeout="${2:-120}"
    info "Ожидание готовности: $resource (таймаут ${timeout}s)..."
    kubectl -n "$NAMESPACE" rollout status "$resource" --timeout="${timeout}s" 2>/dev/null || \
    kubectl -n "$NAMESPACE" wait --for=condition=ready pod -l "app=${resource##*/}" --timeout="${timeout}s" 2>/dev/null || \
    warn "Не удалось дождаться готовности $resource"
}

deploy() {
    info "=== Деплой стека мониторинга dephealth ==="

    # 1. Namespace
    info "Создание namespace $NAMESPACE..."
    kubectl apply -f "$SCRIPT_DIR/namespace.yml"

    # 2. ConfigMaps из JSON-дашбордов
    info "Создание ConfigMaps для Grafana-дашбордов..."
    kubectl create configmap grafana-dashboard-overview \
        --from-file=overview.json="$REPO_ROOT/deploy/grafana/dashboards/overview.json" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    kubectl create configmap grafana-dashboard-service-detail \
        --from-file=service-detail.json="$REPO_ROOT/deploy/grafana/dashboards/service-detail.json" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    kubectl create configmap grafana-dashboard-dependency-map \
        --from-file=dependency-map.json="$REPO_ROOT/deploy/grafana/dashboards/dependency-map.json" \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # 3. VictoriaMetrics
    info "Деплой VictoriaMetrics..."
    kubectl apply -f "$SCRIPT_DIR/victoriametrics/scrape-config.yml"
    kubectl apply -f "$SCRIPT_DIR/victoriametrics/statefulset.yml"
    kubectl apply -f "$SCRIPT_DIR/victoriametrics/service.yml"
    wait_for_ready "statefulset/victoriametrics" 180

    # 4. VMAlert
    info "Деплой VMAlert..."
    kubectl apply -f "$SCRIPT_DIR/vmalert/rules-configmap.yml"
    kubectl apply -f "$SCRIPT_DIR/vmalert/deployment.yml"
    kubectl apply -f "$SCRIPT_DIR/vmalert/service.yml"
    wait_for_ready "deployment/vmalert"

    # 5. Alertmanager
    info "Деплой Alertmanager..."
    kubectl apply -f "$SCRIPT_DIR/alertmanager/configmap.yml"
    kubectl apply -f "$SCRIPT_DIR/alertmanager/deployment.yml"
    kubectl apply -f "$SCRIPT_DIR/alertmanager/service.yml"
    wait_for_ready "deployment/alertmanager"

    # 6. Grafana
    info "Деплой Grafana..."
    kubectl apply -f "$SCRIPT_DIR/grafana/configmap-datasource.yml"
    kubectl apply -f "$SCRIPT_DIR/grafana/configmap-dashboards-provider.yml"
    kubectl apply -f "$SCRIPT_DIR/grafana/deployment.yml"
    kubectl apply -f "$SCRIPT_DIR/grafana/service.yml"
    kubectl apply -f "$SCRIPT_DIR/grafana/httproute.yml"
    wait_for_ready "deployment/grafana" 180

    info "=== Деплой завершён ==="
    echo ""
    info "Компоненты:"
    echo "  VictoriaMetrics : http://victoriametrics.$NAMESPACE.svc:8428"
    echo "  VMAlert         : http://vmalert.$NAMESPACE.svc:8880"
    echo "  Alertmanager    : http://alertmanager.$NAMESPACE.svc:9093"
    echo "  Grafana         : http://grafana.kryukov.lan (admin / dephealth)"
    echo ""
    info "Проверка:"
    echo "  kubectl -n $NAMESPACE get pods"
    echo "  kubectl -n $NAMESPACE port-forward svc/grafana 3000:3000"
    echo "  kubectl -n $NAMESPACE port-forward svc/victoriametrics 8428:8428"
    echo "  kubectl -n $NAMESPACE port-forward svc/alertmanager 9093:9093"
}

teardown() {
    warn "=== Удаление стека мониторинга dephealth ==="
    kubectl delete namespace "$NAMESPACE" --ignore-not-found
    info "Namespace $NAMESPACE удалён."
}

case "${1:-deploy}" in
    deploy)   deploy ;;
    teardown) teardown ;;
    *)
        error "Неизвестная команда: $1"
        echo "Использование: $0 [deploy|teardown]"
        exit 1
        ;;
esac
