*[English version](CONTRIBUTING.md)*

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

## Code Style

Подробные руководства по стилю кода доступны в [`docs/code-style/`](docs/code-style/overview.ru.md):

- [Общие принципы](docs/code-style/overview.ru.md) — общие для всех SDK соглашения, архитектурные слои, философия обработки ошибок
- [Java](docs/code-style/java.ru.md) — именование, JavaDoc, builder pattern, Checkstyle + SpotBugs
- [Go](docs/code-style/go.ru.md) — именование, GoDoc, functional options, golangci-lint v2
- [Python](docs/code-style/python.ru.md) — именование, docstrings, type hints, async/await, ruff + mypy
- [C#](docs/code-style/csharp.ru.md) — именование, XML-doc, async/await, ConfigureAwait, dotnet format
- [Тестирование](docs/code-style/testing.ru.md) — именование тестов, AAA-паттерн, мокирование, покрытие

## Workflow разработки

Git workflow описан в [GIT-WORKFLOW.md](GIT-WORKFLOW.ru.md):

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

Требуют Kubernetes-кластер. Подробнее: [conformance/README.md](conformance/README.ru.md).

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
