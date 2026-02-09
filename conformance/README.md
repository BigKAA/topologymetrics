# Conformance-тесты dephealth

Инструментарий для проверки соответствия SDK спецификации dephealth.
Проверяет Prometheus-метрики, экспортируемые тестовыми сервисами на каждом языке.

## Архитектура

```text
conformance/
├── run.sh                  # Оркестратор: деплой + прогон сценариев
├── runner/
│   ├── verify.py           # Проверка метрик по YAML-сценарию
│   ├── cross_verify.py     # Кросс-языковое сравнение метрик
│   ├── utils.py            # Утилиты kubectl
│   ├── Dockerfile          # Контейнер runner (для CI)
│   └── requirements.txt    # Python-зависимости
├── scenarios/              # YAML-сценарии (8 шт.)
│   ├── basic-health.yml    # Базовая проверка: все зависимости healthy
│   ├── partial-failure.yml # Частичный сбой: часть зависимостей unhealthy
│   ├── full-failure.yml    # Полный сбой: все зависимости unhealthy
│   ├── recovery.yml        # Восстановление: unhealthy → healthy
│   ├── latency.yml         # Проверка histogram латентности
│   ├── labels.yml          # Корректность меток (dependency, type, host, port)
│   ├── timeout.yml         # Таймаут проверки (сервис недоступен)
│   └── initial-state.yml   # Начальное состояние при запуске
├── test-service/           # Go conformance-сервис
├── test-service-python/    # Python conformance-сервис
├── test-service-java/      # Java conformance-сервис
└── test-service-csharp/    # C# conformance-сервис
```

## Режимы деплоя

### Helm (рекомендуется)

Helm-чарт `dephealth-conformance` разворачивает инфраструктуру (PostgreSQL, Redis,
RabbitMQ, Kafka, HTTP/gRPC-заглушки) и все 4 тестовых сервиса в одном релизе.

```bash
# Все языки
./run.sh --lang all --deploy-mode helm

# Только Go
./run.sh --lang go --deploy-mode helm

# С кастомными values
./run.sh --lang all --deploy-mode helm --helm-values ../deploy/helm/dephealth-conformance/values-homelab.yaml

# Кастомный namespace
./run.sh --lang all --deploy-mode helm --namespace my-conformance
```

### kubectl (legacy)

> **Примечание**: raw-манифесты (`conformance/k8s/`) удалены после миграции на Helm.
> Режим kubectl поддерживается в `run.sh`, но требует наличия манифестов.
> Рекомендуется использовать Helm.

## Полный список опций

```text
./run.sh [опции]

  --lang LANG          Язык SDK: go|python|java|csharp|all (по умолчанию: go)
  --no-cleanup         Не удалять инфраструктуру после тестов
  --scenario NAME      Запустить только один сценарий (имя без .yml)
  --metrics-url URL    URL метрик тестового сервиса (по умолчанию: через port-forward)
  --deploy-mode MODE   Режим деплоя: helm|kubectl (по умолчанию: helm)
  --namespace NS       Kubernetes namespace (по умолчанию: dephealth-conformance)
  --helm-values FILE   Дополнительный values-файл для Helm
  --help, -h           Показать справку
```

Переменные окружения (приоритет ниже аргументов):

| Переменная | Описание | По умолчанию |
| --- | --- | --- |
| `DEPLOY_MODE` | Режим деплоя | `helm` |
| `NAMESPACE` | Kubernetes namespace | `dephealth-conformance` |
| `HELM_CHART_DIR` | Путь к Helm-чарту | `deploy/helm/dephealth-conformance` |
| `HELM_VALUES` | Путь к values-файлу | (нет) |

## Сценарии

Каждый сценарий — YAML-файл с описанием:

- `pre_actions` — действия перед проверкой (ожидание, масштабирование, HTTP-запросы)
- `checks` — список проверок метрик
- `post_actions` — восстановление состояния после теста

| Сценарий | Что проверяет |
| --- | --- |
| `basic-health` | Все зависимости healthy, метрики существуют, значения = 1 |
| `partial-failure` | Часть зависимостей отключена, значения = 0 для них |
| `full-failure` | Все зависимости отключены, все значения = 0 |
| `recovery` | Восстановление после сбоя: 0 → 1 |
| `latency` | Histogram бакеты, `_sum`, `_count` присутствуют |
| `labels` | Обязательные метки (name, dependency, type, host, port, critical), корректные значения |
| `timeout` | Поведение при недоступности зависимости (таймаут) |
| `initial-state` | Начальное состояние при запуске сервиса |

## Runner (verify.py)

Отдельный запуск одного сценария:

```bash
python3 conformance/runner/verify.py \
    --scenario conformance/scenarios/basic-health.yml \
    --metrics-url http://localhost:8080/metrics \
    --namespace dephealth-conformance \
    --pod-label conformance-test-service \
    --verbose
```

## Кросс-языковая верификация

Сравнение метрик между всеми SDK:

```bash
python3 conformance/runner/cross_verify.py \
    --urls go=http://localhost:8081/metrics \
           python=http://localhost:8082/metrics \
           java=http://localhost:8083/actuator/prometheus \
           csharp=http://localhost:8084/metrics
```

## Требования

- Kubernetes-кластер с kubectl-доступом
- Helm 3+ (для helm-режима)
- Python 3.10+ с зависимостями: `requests`, `pyyaml`, `prometheus-client`

Установка Python-зависимостей:

```bash
pip install -r conformance/runner/requirements.txt
```
