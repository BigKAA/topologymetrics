[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Developer Guide

## Prerequisites

Required:

- **Docker** 24+ (builds, tests, and linting run in containers)
- **Make** (wrappers for Docker commands in each SDK)

Optional (for Kubernetes testing):

- **kubectl** with cluster access
- **Helm** 3+ (for deploying via Helm charts)
- Local SDKs (for development without Docker): Go 1.25+, Python 3.12+, Java 21, .NET 8

## Quick Start

### Docker Compose (Local Development)

```bash
# 1. Configure environment variables
cp .env.example .env
# Edit .env if needed (registry, versions)

# 2. Start infrastructure (PostgreSQL, Redis, RabbitMQ, Kafka, stubs)
docker compose up -d

# 3. Run tests for the desired SDK
cd sdk-go && make test
cd sdk-python && make test
cd sdk-java && make test
cd sdk-csharp && make test
```

### Make Commands

Each SDK (`sdk-go/`, `sdk-python/`, `sdk-java/`, `sdk-csharp/`) provides
a uniform set of Make targets:

| Command | Description |
| --- | --- |
| `make pull` | Download Docker images (first run) |
| `make test` | Run unit tests |
| `make test-coverage` | Tests with coverage |
| `make lint` | Linting |
| `make fmt` | Auto-formatting |
| `make build` | Compilation / build check |
| `make image` | Build Docker image for the test service |
| `make push` | Upload image to registry |
| `make clean` | Clear caches |

All commands run inside Docker containers — local SDK installation is not required.

## Kubernetes Environment (Helm)

For full testing, 4 Helm charts are available in `deploy/helm/`:

| Chart | Description |
| --- | --- |
| `dephealth-infra` | Infrastructure: PostgreSQL, Redis, RabbitMQ, Kafka, stubs |
| `dephealth-services` | Test services (Go, Python, Java, C#) |
| `dephealth-monitoring` | VictoriaMetrics + Alertmanager + Grafana |
| `dephealth-conformance` | Conformance tests (infra + 4 services) |

Deployment example:

```bash
# Infrastructure + test services
helm install dephealth-infra deploy/helm/dephealth-infra/
helm install dephealth-services deploy/helm/dephealth-services/

# With custom values (e.g. for a private registry)
helm install dephealth-infra deploy/helm/dephealth-infra/ \
    -f deploy/helm/dephealth-infra/values-homelab.yaml
```

## Project Structure

```text
spec/                       # Metric, behavior, and config specification
conformance/                # Conformance tests (8 scenarios × 4 languages)
sdk-go/                     # Go SDK (dephealth/)
sdk-python/                 # Python SDK (dephealth/, dephealth_fastapi/)
sdk-java/                   # Java SDK (Maven multi-module)
sdk-csharp/                 # C# SDK (.NET 8)
test-services/              # Test microservices + K8s manifests
deploy/
├── helm/                   # Helm charts
├── grafana/dashboards/     # Grafana dashboards
├── alerting/               # Alerting rules
└── monitoring/             # Monitoring stack (VM, Alertmanager, Grafana)
docs/                       # Documentation (quickstart, migration, comparison)
plans/                      # Development plans
```

## Development Workflow

Git workflow is described in [GIT-WORKFLOW.md](GIT-WORKFLOW.md):

- Main branch: `master`
- Feature branches: `feature/<scope>-<description>`
- Commit format: `<type>(<scope>): <subject>`
  - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
  - Scope: `sdk-go`, `sdk-python`, `sdk-java`, `sdk-csharp`, `conformance`, `helm`, `docs`
- Commit language: Russian
- Quick fixes (typos) — directly to `master`

## Running Tests

### Unit Tests

```bash
# Single SDK
cd sdk-go && make test

# All SDKs (from root)
for sdk in sdk-go sdk-python sdk-java sdk-csharp; do
    (cd "$sdk" && make test)
done
```

### Linting

```bash
cd sdk-go && make lint       # golangci-lint v2
cd sdk-python && make lint   # ruff + mypy
cd sdk-java && make lint     # Checkstyle + SpotBugs
cd sdk-csharp && make lint   # dotnet format --verify-no-changes
```

### Conformance Tests

Require a Kubernetes cluster. More details: [conformance/README.md](conformance/README.md).

```bash
# All languages via Helm
./conformance/run.sh --lang all --deploy-mode helm

# Single language, single scenario
./conformance/run.sh --lang go --scenario basic-health
```

## SDK-specific Notes

### Go

- Module: `github.com/BigKAA/topologymetrics/sdk-go`
- Package: `dephealth` (directory `sdk-go/dephealth/`)
- `checks` are registered via `init()` in `checks/factories.go`
- Linter: golangci-lint v2 (config `.golangci.yml` in `sdk-go/`)

### Python

- Packages: `dephealth` (core) + `dephealth_fastapi` (FastAPI integration)
- Linter: ruff + mypy
- Minimum check interval: 1 second
- Checker dependencies (asyncpg, aioredis, etc.) — optional extras

### Java

- Maven multi-module: `dephealth-core` + `dephealth-spring-boot-starter`
- Java 21 LTS, Micrometer for metrics
- Linter: Checkstyle (Google-based) + SpotBugs
- Spring Boot endpoint: `/actuator/prometheus`

### C\#

- .NET 8 LTS, prometheus-net 8.2.1
- Projects: `DepHealth.Core` + `DepHealth.AspNetCore`
- Linter: `dotnet format --verify-no-changes`

## Environment Variables

The `.env.example` file contains all supported variables.
Copy it to `.env` — Makefile will automatically pick it up:

| Variable | Description | Default |
| --- | --- | --- |
| `IMAGE_REGISTRY` | Registry for base images | `docker.io` |
| `MCR_REGISTRY` | Registry for .NET images | `mcr.microsoft.com` |
| `PUSH_REGISTRY` | Registry for pushing built images | (local tag) |

## Troubleshooting

### Docker: `permission denied` error

Make sure the user is in the `docker` group:

```bash
sudo usermod -aG docker "$USER"
# Re-login
```

### Make: `No rule to make target`

Make sure you run `make` from the SDK directory (`sdk-go/`, `sdk-python/`, etc.),
not from the repository root.

### Helm: `Error: chart not found`

Build chart dependencies:

```bash
helm dependency build deploy/helm/dephealth-conformance/
```

### golangci-lint: Go version incompatibility

golangci-lint v2 requires a compatible Go version. Makefile uses a Docker image
with the correct version — make sure `make pull` has been run.

### Python: mypy errors after config changes

Delete the mypy cache:

```bash
rm -rf sdk-python/.mypy_cache
```

### Conformance: RabbitMQ failing check

RabbitMQ probe timeout must be >= 10 seconds.
This is accounted for in the `timeout.yml` conformance scenario.

---

<a id="russian"></a>

# Руководство для разработчиков

## Предварительные требования

Обязательно:

- **Docker** 24+ (сборка, тесты и линтинг выполняются в контейнерах)
- **Make** (обёртки для Docker-команд в каждом SDK)

Опционально (для Kubernetes-тестирования):

- **kubectl** с доступом к кластеру
- **Helm** 3+ (для деплоя через Helm-чарты)
- Локальные SDK (для разработки без Docker): Go 1.25+, Python 3.12+, Java 21, .NET 8

## Быстрый старт

### Docker Compose (локальная разработка)

```bash
# 1. Настроить переменные окружения
cp .env.example .env
# Отредактировать .env при необходимости (registry, версии)

# 2. Запустить инфраструктуру (PostgreSQL, Redis, RabbitMQ, Kafka, заглушки)
docker compose up -d

# 3. Запустить тесты для нужного SDK
cd sdk-go && make test
cd sdk-python && make test
cd sdk-java && make test
cd sdk-csharp && make test
```

### Команды Make

Каждый SDK (`sdk-go/`, `sdk-python/`, `sdk-java/`, `sdk-csharp/`) предоставляет
единообразный набор Make-целей:

| Команда | Описание |
| --- | --- |
| `make pull` | Скачать Docker-образы (первый запуск) |
| `make test` | Запуск unit-тестов |
| `make test-coverage` | Тесты с покрытием |
| `make lint` | Линтинг |
| `make fmt` | Автоформатирование |
| `make build` | Компиляция / проверка сборки |
| `make image` | Сборка Docker-образа тестового сервиса |
| `make push` | Загрузка образа в registry |
| `make clean` | Очистка кэшей |

Все команды выполняются в Docker-контейнерах — локальная установка SDK не требуется.

## Kubernetes-окружение (Helm)

Для полного тестирования используются 4 Helm-чарта в `deploy/helm/`:

| Чарт | Описание |
| --- | --- |
| `dephealth-infra` | Инфраструктура: PostgreSQL, Redis, RabbitMQ, Kafka, заглушки |
| `dephealth-services` | Тестовые сервисы (Go, Python, Java, C#) |
| `dephealth-monitoring` | VictoriaMetrics + Alertmanager + Grafana |
| `dephealth-conformance` | Conformance-тесты (infra + 4 сервиса) |

Пример деплоя:

```bash
# Инфраструктура + тестовые сервисы
helm install dephealth-infra deploy/helm/dephealth-infra/
helm install dephealth-services deploy/helm/dephealth-services/

# С кастомными values (например, для приватного registry)
helm install dephealth-infra deploy/helm/dephealth-infra/ \
    -f deploy/helm/dephealth-infra/values-homelab.yaml
```

## Структура проекта

```text
spec/                       # Спецификация метрик, поведения, конфигурации
conformance/                # Conformance-тесты (8 сценариев × 4 языка)
sdk-go/                     # Go SDK (dephealth/)
sdk-python/                 # Python SDK (dephealth/, dephealth_fastapi/)
sdk-java/                   # Java SDK (Maven multi-module)
sdk-csharp/                 # C# SDK (.NET 8)
test-services/              # Тестовые микросервисы + K8s-манифесты
deploy/
├── helm/                   # Helm-чарты
├── grafana/dashboards/     # Grafana-дашборды
├── alerting/               # Правила алертинга
└── monitoring/             # Стек мониторинга (VM, Alertmanager, Grafana)
docs/                       # Документация (quickstart, migration, comparison)
plans/                      # Планы разработки
```

## Workflow разработки

Git workflow описан в [GIT-WORKFLOW.md](GIT-WORKFLOW.md):

- Основная ветка: `master`
- Ветки фич: `feature/<scope>-<description>`
- Формат коммитов: `<type>(<scope>): <subject>`
  - Типы: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
  - Scope: `sdk-go`, `sdk-python`, `sdk-java`, `sdk-csharp`, `conformance`, `helm`, `docs`
- Язык коммитов: русский
- Быстрые правки (опечатки) — напрямую в `master`

## Запуск тестов

### Unit-тесты

```bash
# Один SDK
cd sdk-go && make test

# Все SDK (из корня)
for sdk in sdk-go sdk-python sdk-java sdk-csharp; do
    (cd "$sdk" && make test)
done
```

### Линтинг

```bash
cd sdk-go && make lint       # golangci-lint v2
cd sdk-python && make lint   # ruff + mypy
cd sdk-java && make lint     # Checkstyle + SpotBugs
cd sdk-csharp && make lint   # dotnet format --verify-no-changes
```

### Conformance-тесты

Требуют Kubernetes-кластер. Подробнее: [conformance/README.md](conformance/README.md).

```bash
# Все языки через Helm
./conformance/run.sh --lang all --deploy-mode helm

# Один язык, один сценарий
./conformance/run.sh --lang go --scenario basic-health
```

## SDK-специфичные заметки

### Go

- Модуль: `github.com/BigKAA/topologymetrics/sdk-go`
- Пакет: `dephealth` (каталог `sdk-go/dephealth/`)
- `checks` регистрируются через `init()` в `checks/factories.go`
- Линтер: golangci-lint v2 (конфиг `.golangci.yml` в `sdk-go/`)

### Python

- Пакеты: `dephealth` (core) + `dephealth_fastapi` (FastAPI-интеграция)
- Линтер: ruff + mypy
- Минимальный интервал проверки: 1 секунда
- Зависимости для проверок (asyncpg, aioredis и т.д.) — optional extras

### Java

- Maven multi-module: `dephealth-core` + `dephealth-spring-boot-starter`
- Java 21 LTS, Micrometer для метрик
- Линтер: Checkstyle (Google-based) + SpotBugs
- Spring Boot endpoint: `/actuator/prometheus`

### C\#

- .NET 8 LTS, prometheus-net 8.2.1
- Проекты: `DepHealth.Core` + `DepHealth.AspNetCore`
- Линтер: `dotnet format --verify-no-changes`

## Переменные окружения

Файл `.env.example` содержит все поддерживаемые переменные.
Скопируйте его в `.env` — Makefile автоматически подключит:

| Переменная | Описание | По умолчанию |
| --- | --- | --- |
| `IMAGE_REGISTRY` | Registry для базовых образов | `docker.io` |
| `MCR_REGISTRY` | Registry для .NET-образов | `mcr.microsoft.com` |
| `PUSH_REGISTRY` | Registry для push собранных образов | (локальный тег) |

## Troubleshooting

### Docker: ошибка `permission denied`

Убедитесь, что пользователь в группе `docker`:

```bash
sudo usermod -aG docker "$USER"
# Перелогиниться
```

### Make: `No rule to make target`

Убедитесь, что запускаете `make` из каталога SDK (`sdk-go/`, `sdk-python/` и т.д.),
а не из корня репозитория.

### Helm: `Error: chart not found`

Собрать зависимости чарта:

```bash
helm dependency build deploy/helm/dephealth-conformance/
```

### golangci-lint: несовместимость с версией Go

golangci-lint v2 требует совместимую версию Go. Makefile использует Docker-образ
с правильной версией — убедитесь, что `make pull` выполнен.

### Python: ошибки mypy после изменения конфигурации

Удалите кэш mypy:

```bash
rm -rf sdk-python/.mypy_cache
```

### Conformance: RabbitMQ не проходит проверку

RabbitMQ probe timeout должен быть >= 10 секунд.
В conformance-сценарии `timeout.yml` это учтено.
