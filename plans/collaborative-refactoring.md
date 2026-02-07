# План реорганизации проекта для коллективной работы

> Цель: убрать привязку к домашнему окружению (harbor.kryukov.lan, nfs-client, *.kryukov.lan),
> параметризовать всю инфраструктуру, обеспечить два режима работы — docker-compose (локально)
> и Kubernetes (Helm Charts).

---

## Текущее состояние

**85 вхождений** `harbor.kryukov.lan` в 42 файлах. Привязки:

| Привязка | Файлов | Где |
| --- | --- | --- |
| Registry `harbor.kryukov.lan/docker/` | 28 | Dockerfiles, Makefiles, K8s manifests |
| Registry `harbor.kryukov.lan/mcr/` | 3 | C# SDK (MCR proxy) |
| Registry `harbor.kryukov.lan/library` | 5 | Push собственных образов |
| Домены `*.kryukov.lan` | 6 | HTTPRoute (test1-4, grafana) |
| Gateway `eg` / `envoy-gateway-system` | 5 | HTTPRoute parentRefs |
| StorageClass `nfs-client` | 5 | PVC в StatefulSet'ах |
| IP `192.168.218.180` | ~2 | Документация |

---

## Обзор фаз

| # | Фаза | Результат | Зависит от |
| --- | --- | --- | --- |
| 17 | Dockerfiles: параметризация registry | `ARG REGISTRY` во всех Dockerfiles | — |
| 18 | Makefiles: переменные окружения | Единый `.env.example`, настраиваемые переменные | Фаза 17 |
| 19 | docker-compose для локальной разработки | `docker-compose.yml` — полный стек зависимостей | Фаза 18 |
| 20 | Helm: инфраструктурные зависимости | `deploy/helm/dephealth-infra/` — postgres, redis, kafka, rabbitmq, stubs | Фаза 17 |
| 21 | Helm: тестовые сервисы | `deploy/helm/dephealth-services/` — go, python, java, csharp + HTTPRoutes | Фазы 18, 20 |
| 22 | Helm: мониторинг | `deploy/helm/dephealth-monitoring/` — VM, Grafana, Alertmanager, VMAlert | Фаза 20 |
| 23 | Helm: conformance | `deploy/helm/dephealth-conformance/` — 7 зависимостей + 4 тестовых сервиса | Фазы 20, 21 |
| 24 | Conformance runner: параметризация | `run.sh` / `verify.py` — configurable через env | Фаза 23 |
| 25 | Документация: CONTRIBUTING.md | Как развернуть dev-окружение (compose / k8s) | Фазы 19, 23 |
| 26 | Очистка: удаление raw-манифестов | Удалить `test-services/k8s/`, `conformance/k8s/`, `deploy/monitoring/` | Фазы 21, 22, 23 |

---

## Фаза 17: Dockerfiles — параметризация registry

**Цель**: все Dockerfiles принимают registry как build-arg с дефолтом `docker.io`.

**Статус**: [x] Завершена

### Принцип

```dockerfile
# До:
FROM harbor.kryukov.lan/docker/golang:1.25-alpine AS builder

# После:
ARG REGISTRY=docker.io
FROM ${REGISTRY}/golang:1.25-alpine AS builder
```

Для C# SDK отдельный аргумент `MCR_REGISTRY` (другой прокси):

```dockerfile
ARG MCR_REGISTRY=mcr.microsoft.com
FROM ${MCR_REGISTRY}/dotnet/sdk:8.0 AS builder
```

### Затрагиваемые файлы (13 Dockerfiles)

**Тестовые сервисы (4):**

- `test-services/go-service/Dockerfile` — 2 вхождения (builder + runtime)
- `test-services/python-service/Dockerfile` — 2 вхождения
- `test-services/java-service/Dockerfile` — 2 вхождения
- `test-services/csharp-service/Dockerfile` — 2 вхождения (MCR)

**Conformance-сервисы (5):**

- `conformance/test-service/Dockerfile` — 2 вхождения (Go, legacy)
- `conformance/test-service-python/Dockerfile` — 2 вхождения
- `conformance/test-service-java/Dockerfile` — 2 вхождения
- `conformance/test-service-csharp/Dockerfile` — 2 вхождения (MCR)
- `conformance/runner/Dockerfile` — 1 вхождение

**Stubs (2):**

- `conformance/stubs/http-stub/Dockerfile` — 2 вхождения
- `conformance/stubs/grpc-stub/Dockerfile` — 2 вхождения

### Задачи

- [ ] 17.1. Добавить `ARG REGISTRY=docker.io` перед каждым FROM в 11 Dockerfiles
- [ ] 17.2. Добавить `ARG MCR_REGISTRY=mcr.microsoft.com` в 2 C# Dockerfiles
- [ ] 17.3. Заменить хардкод `harbor.kryukov.lan/docker/` на `${REGISTRY}/` во всех FROM
- [ ] 17.4. Заменить хардкод `harbor.kryukov.lan/mcr/` на `${MCR_REGISTRY}/` в C# Dockerfiles
- [ ] 17.5. Проверить сборку всех образов с дефолтным registry (docker.io)
- [ ] 17.6. Проверить сборку с `--build-arg REGISTRY=harbor.kryukov.lan/docker`

### Важные нюансы

- `ARG` перед `FROM` действует только на выбор базового образа
- Для multi-stage: `ARG REGISTRY` нужно повторить перед каждым `FROM` (или один раз до первого FROM — работает для всех стадий при использовании `${REGISTRY}`)
- Docker BuildKit кеширует слои по-разному при разных ARG — это ОК

---

## Фаза 18: Makefiles — переменные окружения

**Цель**: все Makefiles используют настраиваемые переменные с разумными дефолтами.
Единый `.env.example` для документирования.

**Статус**: [x] Завершена

### Текущие переменные (хардкод)

| Makefile | Переменная | Текущее значение |
| --- | --- | --- |
| sdk-go | `GO_IMAGE` | `harbor.kryukov.lan/docker/golang:$(GO_VERSION)` |
| sdk-go | `LINT_IMAGE` | `harbor.kryukov.lan/docker/golangci/golangci-lint:$(LINT_VERSION)` |
| sdk-go | `ALPINE_IMAGE` | `harbor.kryukov.lan/docker/alpine:3.21` |
| sdk-go | `REGISTRY` | `harbor.kryukov.lan/library` |
| sdk-python | `PY_IMAGE` | `harbor.kryukov.lan/docker/python:$(PYTHON_VERSION)-slim` |
| sdk-python | `REGISTRY` | `harbor.kryukov.lan/library` |
| sdk-java | `MAVEN_IMAGE` | `harbor.kryukov.lan/docker/maven:3.9-eclipse-temurin-$(JAVA_VERSION)` |
| sdk-java | `RUNTIME_IMAGE` | `harbor.kryukov.lan/docker/eclipse-temurin:$(JAVA_VERSION)-jre-alpine` |
| sdk-java | `REGISTRY` | `harbor.kryukov.lan/library` |
| sdk-csharp | `SDK_IMAGE` | `harbor.kryukov.lan/mcr/dotnet/sdk:$(DOTNET_VERSION)` |
| sdk-csharp | `RUNTIME_IMAGE` | `harbor.kryukov.lan/mcr/dotnet/aspnet:$(DOTNET_VERSION)-alpine` |
| sdk-csharp | `REGISTRY` | `harbor.kryukov.lan/library` |

### Целевая структура переменных

```makefile
# Общие переменные (одинаковы для всех Makefiles):
IMAGE_REGISTRY    ?= docker.io              # откуда тянуть базовые образы
MCR_REGISTRY      ?= mcr.microsoft.com      # MCR-образы (.NET)
PUSH_REGISTRY     ?=                         # куда пушить собранные образы (пусто = не пушить)

# Пример для sdk-go/Makefile:
GO_IMAGE      = $(IMAGE_REGISTRY)/golang:$(GO_VERSION)
GO_ALPINE     = $(IMAGE_REGISTRY)/golang:$(GO_VERSION)-alpine
ALPINE_IMAGE  = $(IMAGE_REGISTRY)/alpine:3.21
LINT_IMAGE    = $(IMAGE_REGISTRY)/golangci/golangci-lint:$(LINT_VERSION)
```

### `.env.example` (корень проекта)

```bash
# === Container Registry ===
# Registry для базовых Docker Hub образов (golang, python, redis и т.д.)
IMAGE_REGISTRY=docker.io

# Registry для MCR-образов (.NET SDK/Runtime)
MCR_REGISTRY=mcr.microsoft.com

# Registry для push собранных образов (оставить пустым, если push не нужен)
PUSH_REGISTRY=

# === Kubernetes (только при использовании Helm) ===
# StorageClass для PVC (пусто = default кластера)
STORAGE_CLASS=

# Базовый домен для HTTPRoute
DOMAIN=dephealth.local

# Gateway API: имя и namespace gateway
GATEWAY_NAME=gateway
GATEWAY_NAMESPACE=default
```

### Задачи

- [ ] 18.1. Создать `.env.example` в корне проекта
- [ ] 18.2. Добавить поддержку `-include .env` во все 4 Makefile (опциональная загрузка)
- [ ] 18.3. Рефакторинг `sdk-go/Makefile` — использовать `IMAGE_REGISTRY`, `PUSH_REGISTRY`
- [ ] 18.4. Рефакторинг `sdk-python/Makefile` — аналогично
- [ ] 18.5. Рефакторинг `sdk-java/Makefile` — аналогично
- [ ] 18.6. Рефакторинг `sdk-csharp/Makefile` — аналогично (+ `MCR_REGISTRY`)
- [ ] 18.7. Передача `--build-arg REGISTRY=$(IMAGE_REGISTRY)` в `make image` targets
- [ ] 18.8. Обновить `plans/makefile-conventions.md` — отразить новые переменные
- [ ] 18.9. Добавить `.env` в `.gitignore`
- [ ] 18.10. Проверить `make test` / `make lint` с дефолтами (docker.io)

---

## Фаза 19: docker-compose для локальной разработки

**Цель**: разработчик может поднять все зависимости одной командой
без Kubernetes. Два профиля: `dev` (базовые 4 зависимости) и `full` (все 7 + stubs).

**Статус**: [x] Завершена

### Файлы

- `docker-compose.yml` — основной файл
- `docker-compose.override.yml.example` — пример переопределений (registry, порты)

### Содержимое `docker-compose.yml`

```yaml
# Профиль "dev": базовые зависимости для разработки SDK
# Профиль "full": все 7 зависимостей (как в conformance)
#
# Использование:
#   docker compose up -d                       # только базовые (postgres, redis)
#   docker compose --profile full up -d        # все зависимости
#   docker compose down -v                     # остановить и удалить volumes

x-registry: &registry ${IMAGE_REGISTRY:-docker.io}

services:
  postgres:
    image: ${IMAGE_REGISTRY:-docker.io}/postgres:17-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_USER: dephealth
      POSTGRES_PASSWORD: dephealth-test-pass
      POSTGRES_DB: dephealth
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dephealth"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: ${IMAGE_REGISTRY:-docker.io}/redis:7-alpine
    ports: ["6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  kafka:
    image: ${IMAGE_REGISTRY:-docker.io}/apache/kafka:3.8.1
    profiles: ["full"]
    ports: ["9092:9092"]
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_PROCESS_ROLES: broker,controller
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092
      KAFKA_CONTROLLER_QUORUM_VOTERS: 1@localhost:9093
      KAFKA_CONTROLLER_LISTENER_NAMES: CONTROLLER
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,CONTROLLER:PLAINTEXT
    volumes:
      - kafka-data:/var/lib/kafka/data

  rabbitmq:
    image: ${IMAGE_REGISTRY:-docker.io}/rabbitmq:3-management-alpine
    profiles: ["full"]
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: dephealth
      RABBITMQ_DEFAULT_PASS: dephealth-test-pass
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "check_port_connectivity"]
      interval: 10s
      timeout: 10s
      retries: 5

  http-stub:
    image: ${PUSH_REGISTRY:-local}/dephealth-http-stub:latest
    profiles: ["full"]
    build:
      context: .
      dockerfile: conformance/stubs/http-stub/Dockerfile
      args:
        REGISTRY: ${IMAGE_REGISTRY:-docker.io}
    ports: ["8081:8080"]

  grpc-stub:
    image: ${PUSH_REGISTRY:-local}/dephealth-grpc-stub:latest
    profiles: ["full"]
    build:
      context: .
      dockerfile: conformance/stubs/grpc-stub/Dockerfile
      args:
        REGISTRY: ${IMAGE_REGISTRY:-docker.io}
    ports:
      - "9090:9090"
      - "8082:8080"

volumes:
  postgres-data:
  kafka-data:
```

### Задачи

- [x] 19.1. Создать `docker-compose.yml` в корне проекта
- [x] 19.2. Создать `docker-compose.override.yml.example` с примером для harbor
- [x] 19.3. Проверить `docker compose up -d` с дефолтами (docker.io)
- [x] 19.4. Проверить `docker compose --profile full up -d`
- [x] 19.5. Проверить подключение SDK-тестов к compose-сервисам
- [x] 19.6. Добавить `docker-compose.override.yml` в `.gitignore`

---

## Фаза 20: Helm Chart — инфраструктурные зависимости

**Цель**: Helm chart для развёртывания postgres, redis, kafka, rabbitmq, http/grpc-stubs в Kubernetes.
Заменяет: `test-services/k8s/postgres/`, `test-services/k8s/redis/`,
`conformance/k8s/postgres/`, `conformance/k8s/redis/`, `conformance/k8s/kafka/`,
`conformance/k8s/rabbitmq/`, `conformance/k8s/stubs/`, `test-services/k8s/stubs/`.

**Статус**: [ ] Не начата

### Структура

```
deploy/helm/dephealth-infra/
├── Chart.yaml
├── values.yaml                 # дефолты (docker.io, без storageClass)
├── values-homelab.yaml         # домашнее окружение
├── templates/
│   ├── _helpers.tpl
│   ├── namespace.yml
│   ├── postgres-primary.yml    # StatefulSet + Service
│   ├── postgres-replica.yml    # StatefulSet + Service (опционально)
│   ├── redis.yml               # Deployment + Service
│   ├── kafka.yml               # StatefulSet + Service (опционально)
│   ├── rabbitmq.yml            # Deployment + Service (опционально)
│   ├── http-stub.yml           # Deployment + Service
│   └── grpc-stub.yml           # Deployment + Service
```

### `values.yaml` (ключевые параметры)

```yaml
global:
  imageRegistry: docker.io
  storageClass: ""
  namespace: dephealth-test

postgres:
  enabled: true
  image: postgres
  tag: "17-alpine"
  credentials:
    user: dephealth
    password: dephealth-test-pass
    database: dephealth
  primary:
    storage: 1Gi
  replica:
    enabled: false          # true для conformance
    storage: 1Gi

redis:
  enabled: true
  image: redis
  tag: "7-alpine"

kafka:
  enabled: false            # true для conformance
  image: apache/kafka
  tag: "3.8.1"
  storage: 1Gi

rabbitmq:
  enabled: false            # true для conformance
  image: rabbitmq
  tag: "3-management-alpine"
  credentials:
    user: dephealth
    password: dephealth-test-pass

stubs:
  httpStub:
    enabled: true
    image: ""               # custom registry / pre-built
    tag: latest
  grpcStub:
    enabled: true
    image: ""
    tag: latest
```

### `values-homelab.yaml`

```yaml
global:
  imageRegistry: harbor.kryukov.lan/docker
  storageClass: nfs-client
```

### Задачи

- [ ] 20.1. Создать `deploy/helm/dephealth-infra/Chart.yaml`
- [ ] 20.2. Создать `values.yaml` с дефолтами для docker.io
- [ ] 20.3. Создать `values-homelab.yaml` для домашнего окружения
- [ ] 20.4. Шаблоны: namespace, postgres-primary (StatefulSet + Service)
- [ ] 20.5. Шаблоны: postgres-replica (условный, `{{ if .Values.postgres.replica.enabled }}`)
- [ ] 20.6. Шаблоны: redis (Deployment + Service)
- [ ] 20.7. Шаблоны: kafka (условный StatefulSet + Service)
- [ ] 20.8. Шаблоны: rabbitmq (условный Deployment + Service)
- [ ] 20.9. Шаблоны: http-stub, grpc-stub (Deployment + Service)
- [ ] 20.10. Шаблон `_helpers.tpl` — helper для формирования image path
- [ ] 20.11. `helm template` — проверить генерацию манифестов
- [ ] 20.12. `helm install` — проверить деплой в кластер
- [ ] 20.13. Проверить с `values-homelab.yaml`

---

## Фаза 21: Helm Chart — тестовые сервисы

**Цель**: Helm chart для go/python/java/csharp тестовых сервисов + HTTPRoutes.
Заменяет: `test-services/k8s/go-service/`, `test-services/k8s/python-service/`,
`test-services/k8s/java-service/`, `test-services/k8s/csharp-service/`.

**Статус**: [ ] Не начата

### Структура

```
deploy/helm/dephealth-services/
├── Chart.yaml
├── values.yaml
├── values-homelab.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── go-service.yml          # Deployment + Service + ConfigMap
│   ├── python-service.yml
│   ├── java-service.yml
│   ├── csharp-service.yml
│   └── httproutes.yml          # Все HTTPRoute (условные)
```

### `values.yaml`

```yaml
global:
  imageRegistry: docker.io
  pushRegistry: ""
  domain: dephealth.local
  gateway:
    name: gateway
    namespace: default
  namespace: dephealth-test

services:
  go:
    enabled: true
    image: dephealth-test-go
    tag: latest
    hostname: test1.dephealth.local
    metricsPath: /metrics
  python:
    enabled: true
    image: dephealth-test-python
    tag: latest
    hostname: test2.dephealth.local
    metricsPath: /metrics
  java:
    enabled: true
    image: dephealth-test-java
    tag: latest
    hostname: test3.dephealth.local
    metricsPath: /actuator/prometheus
  csharp:
    enabled: true
    image: dephealth-test-csharp
    tag: latest
    hostname: test4.dephealth.local
    metricsPath: /metrics

# Зависимости — ссылки на сервисы из dephealth-infra
dependencies:
  postgresHost: postgres-primary
  redisHost: redis
  httpStubHost: http-stub
  grpcStubHost: grpc-stub
  infraNamespace: ""           # если пусто — тот же namespace
```

### `values-homelab.yaml`

```yaml
global:
  pushRegistry: harbor.kryukov.lan/library
  domain: kryukov.lan
  gateway:
    name: eg
    namespace: envoy-gateway-system

services:
  go:
    hostname: test1.kryukov.lan
  python:
    hostname: test2.kryukov.lan
  java:
    hostname: test3.kryukov.lan
  csharp:
    hostname: test4.kryukov.lan
```

### Задачи

- [ ] 21.1. Создать `deploy/helm/dephealth-services/Chart.yaml`
- [ ] 21.2. Создать `values.yaml` и `values-homelab.yaml`
- [ ] 21.3. Шаблоны: 4 Deployment + Service + ConfigMap (по одному на язык)
- [ ] 21.4. Шаблон: HTTPRoutes (условные, по одному на сервис)
- [ ] 21.5. `_helpers.tpl` — helpers для image, labels, selectors
- [ ] 21.6. `helm template` — проверить генерацию
- [ ] 21.7. `helm install` — проверить деплой с dephealth-infra

---

## Фаза 22: Helm Chart — мониторинг

**Цель**: Helm chart для VictoriaMetrics + VMAlert + Alertmanager + Grafana.
Заменяет: `deploy/monitoring/`.

**Статус**: [ ] Не начата

### Структура

```
deploy/helm/dephealth-monitoring/
├── Chart.yaml
├── values.yaml
├── values-homelab.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── namespace.yml
│   ├── victoriametrics.yml     # StatefulSet + Service + scrape ConfigMap
│   ├── vmalert.yml             # Deployment + Service + rules ConfigMap
│   ├── alertmanager.yml        # Deployment + Service + config ConfigMap
│   ├── grafana.yml             # Deployment + Service + datasource/dashboard ConfigMaps
│   └── grafana-httproute.yml   # HTTPRoute (условный)
├── dashboards/                 # JSON-файлы дашбордов (монтируются как ConfigMap)
│   ├── overview.json
│   ├── service-detail.json
│   └── dependency-map.json
```

### `values.yaml`

```yaml
global:
  imageRegistry: docker.io
  storageClass: ""
  namespace: dephealth-monitoring
  domain: dephealth.local
  gateway:
    name: gateway
    namespace: default

victoriametrics:
  image: victoriametrics/victoria-metrics
  tag: v1.108.1
  storage: 2Gi
  retention: 7d

vmalert:
  image: victoriametrics/vmalert
  tag: v1.108.1

alertmanager:
  image: prom/alertmanager
  tag: v0.28.1

grafana:
  image: grafana/grafana
  tag: "11.6.0"
  hostname: grafana.dephealth.local
  adminPassword: dephealth
  httproute:
    enabled: true

scrapeTargets:
  namespace: dephealth-test
  services:
    - name: go-service
      port: 8080
      metricsPath: /metrics
    - name: python-service
      port: 8080
      metricsPath: /metrics
    - name: java-service
      port: 8080
      metricsPath: /actuator/prometheus
    - name: csharp-service
      port: 8080
      metricsPath: /metrics
```

### Задачи

- [ ] 22.1. Создать Chart.yaml, values.yaml, values-homelab.yaml
- [ ] 22.2. Перенести JSON-дашборды в `dashboards/`
- [ ] 22.3. Шаблоны: VictoriaMetrics (StatefulSet + Service + scrape ConfigMap)
- [ ] 22.4. Шаблоны: VMAlert (Deployment + Service + rules ConfigMap)
- [ ] 22.5. Шаблоны: Alertmanager (Deployment + Service + config ConfigMap)
- [ ] 22.6. Шаблоны: Grafana (Deployment + Service + datasource + dashboard provider + dashboards)
- [ ] 22.7. Шаблон: Grafana HTTPRoute (условный)
- [ ] 22.8. Удалить старый `deploy/monitoring/deploy.sh` (заменён Helm)
- [ ] 22.9. `helm template` + `helm install` — проверить

---

## Фаза 23: Helm Chart — conformance

**Цель**: Helm chart для полного conformance-окружения (7 зависимостей + 4 тестовых сервиса).
Переиспользует шаблоны из dephealth-infra (как зависимость или дублирование).

**Статус**: [ ] Не начата

### Подход

Два варианта:

**A. Зависимость от dephealth-infra** (через `Chart.yaml` dependencies):

```yaml
dependencies:
  - name: dephealth-infra
    version: "0.1.0"
    repository: "file://../dephealth-infra"
```

**B. Standalone chart** с собственными шаблонами (проще, но дублирование).

Рекомендация: **вариант A** — DRY, но требует `helm dependency update`.

### Структура

```
deploy/helm/dephealth-conformance/
├── Chart.yaml                  # зависимость: dephealth-infra
├── values.yaml
├── values-homelab.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── namespace.yml
│   ├── test-service-go.yml
│   ├── test-service-python.yml
│   ├── test-service-java.yml
│   └── test-service-csharp.yml
```

### `values.yaml` (ключевые отличия от test-services)

```yaml
global:
  imageRegistry: docker.io
  storageClass: ""
  namespace: dephealth-conformance

# Переопределение для dephealth-infra subchart
dephealth-infra:
  global:
    namespace: dephealth-conformance
  postgres:
    replica:
      enabled: true             # conformance нужна реплика
  kafka:
    enabled: true               # conformance нужен Kafka
  rabbitmq:
    enabled: true               # conformance нужен RabbitMQ

conformanceServices:
  go:
    enabled: true
    image: dephealth-conformance-test
    tag: latest
    metricsPath: /metrics
  python:
    enabled: true
    image: dephealth-conformance-python
    tag: latest
    metricsPath: /metrics
  java:
    enabled: true
    image: dephealth-conformance-java
    tag: latest
    metricsPath: /actuator/prometheus
  csharp:
    enabled: true
    image: dephealth-conformance-csharp
    tag: latest
    metricsPath: /metrics
```

### Задачи

- [ ] 23.1. Создать Chart.yaml с зависимостью от dephealth-infra
- [ ] 23.2. Создать values.yaml и values-homelab.yaml
- [ ] 23.3. Шаблоны: 4 conformance-сервиса (Deployment + Service + ConfigMap)
- [ ] 23.4. `helm dependency update` + `helm template` — проверить
- [ ] 23.5. `helm install` — проверить полный стек в кластере
- [ ] 23.6. Прогнать conformance-тесты через Helm-деплой

---

## Фаза 24: Conformance runner — параметризация

**Цель**: `run.sh` и `verify.py` принимают все настройки через env/аргументы.
Не привязаны к конкретным путям манифестов.

**Статус**: [ ] Не начата

### Изменения в `run.sh`

```bash
# Текущее: хардкод kubectl apply -f conformance/k8s/...
# Новое: два режима деплоя

DEPLOY_MODE=${DEPLOY_MODE:-helm}  # helm | kubectl
HELM_VALUES=${HELM_VALUES:-}      # путь к values-файлу

deploy_infra() {
  if [ "$DEPLOY_MODE" = "helm" ]; then
    helm upgrade --install dephealth-conformance \
      deploy/helm/dephealth-conformance/ \
      ${HELM_VALUES:+-f $HELM_VALUES} \
      --wait --timeout 5m
  else
    # fallback на raw kubectl (для CI без Helm)
    kubectl apply -f conformance/k8s/
  fi
}
```

### Изменения в `verify.py`

- Параметр `--namespace` (вместо хардкода `dephealth-conformance`)
- Параметр `--pod-label` уже есть — ОК
- Параметр `--metrics-url` уже есть — ОК

### Задачи

- [ ] 24.1. Рефакторинг `run.sh` — поддержка Helm-деплоя
- [ ] 24.2. Параметр `--namespace` в `verify.py` и `utils.py`
- [ ] 24.3. Обновить `conformance/runner/Dockerfile` — включить Helm CLI (опционально)
- [ ] 24.4. Обновить README в `conformance/` — описать оба режима
- [ ] 24.5. Прогнать conformance с Helm-деплоем

---

## Фаза 25: Документация — CONTRIBUTING.md

**Цель**: исчерпывающее руководство для нового разработчика.

**Статус**: [ ] Не начата

### Структура CONTRIBUTING.md

1. **Предварительные требования** — Docker, kubectl (опционально), Helm (опционально)
2. **Быстрый старт (docker-compose)** — `docker compose up -d && make test`
3. **Kubernetes-окружение (Helm)** — для полного стека
4. **Структура проекта** — что где лежит
5. **Workflow разработки** — git branches, commit format, тесты
6. **Настройка своего окружения** — `.env`, `values-*.yaml`
7. **Запуск тестов** — unit, integration, conformance
8. **Troubleshooting** — частые проблемы

### Задачи

- [ ] 25.1. Создать `CONTRIBUTING.md` в корне проекта
- [ ] 25.2. Обновить корневой `README.md` — ссылка на CONTRIBUTING
- [ ] 25.3. Обновить `CLAUDE.md` — отразить новую структуру
- [ ] 25.4. Обновить `docs/quickstart/*.md` — убрать хардкод доменов

---

## Фаза 26: Очистка — удаление raw-манифестов

**Цель**: удалить старые raw K8s-манифесты после полной миграции на Helm.

**Статус**: [ ] Не начата

### Удаляемые директории

```
test-services/k8s/              # → deploy/helm/dephealth-infra + dephealth-services
conformance/k8s/                # → deploy/helm/dephealth-conformance
deploy/monitoring/              # → deploy/helm/dephealth-monitoring
```

### Сохраняемые файлы (перенести в Helm)

- `deploy/monitoring/deploy.sh` → удалить (Helm заменяет)
- `deploy/grafana/dashboards/*.json` → `deploy/helm/dephealth-monitoring/dashboards/`
- `deploy/alerting/rules.yml` → шаблон в dephealth-monitoring
- `deploy/alerting/inhibition-rules.yml` → шаблон в dephealth-monitoring

### Задачи

- [ ] 26.1. Проверить, что все Helm-чарты полностью заменяют raw-манифесты
- [ ] 26.2. Удалить `test-services/k8s/`
- [ ] 26.3. Удалить `conformance/k8s/`
- [ ] 26.4. Удалить `deploy/monitoring/` (кроме Helm)
- [ ] 26.5. Обновить все ссылки в документации
- [ ] 26.6. Финальный conformance-прогон через Helm

---

## Порядок выполнения и зависимости

```
Фаза 17 (Dockerfiles)
  ↓
Фаза 18 (Makefiles + .env)
  ↓         ↓
Фаза 19   Фаза 20 (Helm infra)
(compose)    ↓         ↓
  ↓       Фаза 21   Фаза 22
  ↓      (services) (monitoring)
  ↓         ↓
  ↓       Фаза 23 (conformance)
  ↓         ↓
  ↓       Фаза 24 (runner)
  ↓         ↓
  ↓─────────↓
      ↓
Фаза 25 (CONTRIBUTING.md)
      ↓
Фаза 26 (очистка)
```

---

## Оценка объёма

| Фаза | Файлов создать/изменить | Сложность |
| --- | --- | --- |
| 17 | 13 Dockerfiles | Низкая |
| 18 | 4 Makefiles + .env + .gitignore | Низкая |
| 19 | 2 файла compose | Средняя |
| 20 | ~12 файлов (chart + templates) | Высокая |
| 21 | ~8 файлов | Средняя |
| 22 | ~12 файлов + dashboards | Высокая |
| 23 | ~8 файлов | Средняя |
| 24 | 3 файла | Низкая |
| 25 | 4 файла docs | Низкая |
| 26 | Удаление ~40 файлов | Низкая |
