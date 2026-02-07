# План разработки dephealth

> Подробный план разработки SDK для мониторинга зависимостей микросервисов.
> Пилотный язык: **Go**. Тестовая среда: **Docker + Kubernetes**.

---

## Обзор фаз

| # | Фаза | Результат | Зависит от |
| --- | --- | --- | --- |
| 1 | Спецификация | `spec/` — контракты метрик, поведения, конфигурации | — |
| 2 | Инфраструктура conformance-тестов | `conformance/` — docker-compose, runner, сценарии | Фаза 1 |
| 3 | Go SDK: ядро и парсер | `sdk-go/dephealth/` — абстракции, парсер конфигов | Фаза 1 |
| 4 | Go SDK: чекеры | `sdk-go/dephealth/checks/` — все 8 типов проверок | Фаза 3 |
| 5 | Go SDK: метрики и планировщик | `sdk-go/dephealth/` — Prometheus exporter, scheduler | Фазы 3, 4 |
| 6 | Go SDK: публичный API и contrib | `sdk-go/dephealth/` — Option pattern, contrib/ | Фаза 5 |
| 7 | Тестовый сервис на Go | `test-services/go-service/` — пилотный микросервис | Фаза 6 |
| 8 | Conformance-прогон Go SDK | Прохождение всех сценариев | Фазы 2, 7 |
| 9 | Документация и CI/CD | `docs/`, GitHub Actions, Makefile | Фаза 8 |
| 10 | Grafana дашборды и алерты | `deploy/grafana/`, `deploy/alerting/` | Фаза 8 |
| 11 | Инфраструктура контейнерной разработки | Makefile Go SDK, конвенции, Docker-инфраструктура | Фаза 10 |
| 12 | Python SDK — Core + Checkers | `sdk-python/` — ядро, парсер, 8 чекеров, метрики, планировщик | Фаза 11 |
| 13 | Python SDK — FastAPI + Conformance | `dephealth-fastapi`, тестовый сервис, conformance-прогон | Фаза 12 |
| 14 | Java SDK — Core + Spring Boot + Conformance | `sdk-java/` — ядро, чекеры, Spring Boot, conformance-прогон | Фаза 11 |
| 15 | C# SDK — Core + ASP.NET + Conformance | `sdk-csharp/` — ядро, чекеры, ASP.NET, conformance-прогон | Фаза 11 |
| 16 | Кросс-языковая документация + Релиз v1.0 | docs для всех языков, финальный прогон, публикация | Фазы 13, 14, 15 |

---

## Фаза 1: Спецификация

**Цель**: зафиксировать единый источник правды для всех SDK.

**Статус**: [x] Завершена

### Задачи фазы 1

#### 1.1. Контракт метрик (`spec/metric-contract.md`)

- [x] Определить формат метрики здоровья `app_dependency_health`
  - Тип: Gauge
  - Значения: `1` (доступен), `0` (недоступен)
  - Обязательные метки: `dependency`, `type`, `host`, `port`
  - Опциональные метки: `role`, `shard`, `vhost`
  - Правила формирования значений меток (допустимые символы, длина)
- [x] Определить формат метрики латентности `app_dependency_latency_seconds`
  - Тип: Histogram
  - Бакеты: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
  - Метки: идентичны `app_dependency_health`
- [x] Описать формат endpoint `/metrics` (OpenMetrics / Prometheus text format)
- [x] Привести примеры вывода `/metrics` для типовых конфигураций
- [x] Описать поведение при множественных endpoint-ах одной зависимости
  (одна метрика на endpoint vs агрегация)

#### 1.2. Контракт поведения (`spec/check-behavior.md`)

- [x] Зафиксировать параметры по умолчанию:
  - `checkInterval`: 15s
  - `timeout`: 5s
  - `initialDelay`: 5s
  - `failureThreshold`: 1
  - `successThreshold`: 1
- [x] Описать жизненный цикл проверки:
  - Инициализация → `initialDelay` → первая проверка → периодические проверки
  - Поведение при старте (начальное значение метрики до первой проверки)
  - Поведение при останове (graceful shutdown)
- [x] Описать логику порогов (failureThreshold, successThreshold)
  - Переход healthy → unhealthy
  - Переход unhealthy → healthy
  - Начальное состояние (unknown → первая проверка)
- [x] Описать каждый тип проверки:
  - `http`: GET к `healthPath` (default `/health`), ожидание 2xx, поддержка TLS
  - `grpc`: gRPC Health Check Protocol (grpc.health.v1.Health/Check)
  - `tcp`: установка TCP-соединения и немедленное закрытие
  - `postgres`: `SELECT 1` через connection pool или новое соединение
  - `mysql`: `SELECT 1` через connection pool или новое соединение
  - `redis`: команда `PING`, ожидание `PONG`
  - `amqp`: проверка соединения с брокером (connection open/close)
  - `kafka`: Metadata request к брокеру
- [x] Описать два режима работы:
  - Автономный: SDK создаёт временное соединение
  - Интеграция с pool: SDK использует существующий connection pool
- [x] Описать обработку ошибок:
  - Таймаут → unhealthy
  - DNS resolution failure → unhealthy
  - Connection refused → unhealthy
  - TLS handshake failure → unhealthy

#### 1.3. Контракт конфигурации (`spec/config-contract.md`)

- [x] Описать поддерживаемые форматы ввода:
  - Полный URL: `postgres://user:pass@host:port/db`
  - Отдельные параметры: `host`, `port` (env vars или прямые значения)
  - Connection string: `Host=...;Port=...;Database=...`
  - JDBC URL: `jdbc:postgresql://host:port/db`
- [x] Описать правила парсинга каждого формата:
  - Извлечение `host`, `port`, `type`
  - Автоопределение `type` из URL-схемы
  - Обработка IPv6 адресов
  - Обработка URL без порта (default ports)
  - Обработка URL с несколькими хостами (кластер)
- [x] Таблица портов по умолчанию для каждого типа
- [x] Описать программный API конфигурации:
  - Builder pattern / Option pattern
  - Обязательные параметры: `name` (логическое имя)
  - Опциональные: `type`, `critical`, `checkInterval`, `timeout`
- [x] Описать конфигурацию через environment variables:
  - Формат: `DEPHEALTH_<DEPENDENCY_NAME>_<PARAM>`
  - Пример: `DEPHEALTH_POSTGRES_MAIN_CHECK_INTERVAL=30s`

### Артефакты фазы 1

```text
spec/
├── metric-contract.md
├── check-behavior.md
└── config-contract.md
```

### Критерии завершения фазы 1

- Все три документа написаны и прошли markdownlint
- Примеры покрывают все типовые сценарии
- Нет противоречий между документами

---

## Фаза 2: Инфраструктура conformance-тестов

**Цель**: создать тестовую среду в Kubernetes и сценарии для проверки соответствия SDK спецификации.

**Статус**: [x] Завершена

### Задачи фазы 2

#### 2.1. Namespace и базовая инфраструктура

- [x] Создать namespace `dephealth-conformance`
- [x] Определить структуру каталогов k8s-манифестов

#### 2.2. Kubernetes-манифесты зависимостей

- [x] **PostgreSQL** (StatefulSet + Service):
  - Primary экземпляр
  - Replica (отдельный StatefulSet для теста partial failure)
  - Service для каждого экземпляра
- [x] **Redis** (Deployment + Service)
- [x] **RabbitMQ** (Deployment + Service, с management plugin)
- [x] **Kafka** (StatefulSet + Service, KRaft mode без Zookeeper)
- [x] Readiness/liveness probes для каждого сервиса
- [x] ConfigMaps и Secrets для конфигурации

#### 2.3. Управляемые заглушки

- [x] **HTTP-заглушка** (Deployment + Service):
  - Dockerfile, образ в `harbor.kryukov.lan/library/dephealth-http-stub`
  - `/health` — возвращает 200 (управляемо через `/admin/toggle`)
  - `/admin/toggle` — включить/выключить ответ health
  - `/admin/delay?ms=1000` — добавить задержку
- [x] **gRPC-заглушка** (Deployment + Service):
  - Dockerfile, образ в `harbor.kryukov.lan/library/dephealth-grpc-stub`
  - grpc.health.v1.Health/Check — управляемый статус
  - Admin HTTP API для переключения состояния

#### 2.4. Conformance runner (`conformance/runner/`)

- [x] Скрипт `verify.py` (Python):
  - Запрос `GET /metrics` к тестовому сервису (через Service DNS)
  - Парсинг Prometheus text format
  - Проверка наличия метрик с правильными именами
  - Проверка значений меток
  - Проверка значений метрик (0 или 1)
  - Проверка наличия histogram бакетов
- [x] Утилиты для управления инфраструктурой (`utils.py`):
  - Масштабирование Deployments/StatefulSets через kubectl
  - Ожидание стабилизации метрик (несколько циклов проверок)
  - Проверка readiness подов перед запуском тестов

#### 2.5. Тестовые сценарии (`conformance/scenarios/`)

- [x] `basic-health.yml` — все endpoint-ы доступны → все метрики = 1
- [x] `partial-failure.yml` — масштабировать реплику PG в 0 → правильные метки
- [x] `full-failure.yml` — масштабировать Redis в 0 → метрика = 0
- [x] `recovery.yml` — масштабировать Redis обратно в 1 → метрика возвращается в 1
- [x] `latency.yml` — проверить наличие histogram с правильными бакетами
- [x] `labels.yml` — проверить правильность всех меток (host, port, type, dependency)
- [x] `timeout.yml` — включить задержку в заглушке > timeout → unhealthy
- [x] `initial-state.yml` — поведение до первой проверки (после initialDelay)

#### 2.6. Скрипт оркестрации

- [x] `conformance/run.sh` — скрипт для полного цикла conformance:
  - `kubectl apply` — деплой инфраструктуры
  - Ожидание readiness всех подов
  - Деплой тестового сервиса (SDK под тестом)
  - Запуск runner для каждого сценария
  - Сбор результатов
  - Опциональная очистка

### Артефакты фазы 2

```text
conformance/
├── k8s/
│   ├── namespace.yml
│   ├── postgres/
│   │   ├── primary-statefulset.yml
│   │   ├── replica-statefulset.yml
│   │   └── service.yml
│   ├── redis/
│   │   ├── deployment.yml
│   │   └── service.yml
│   ├── rabbitmq/
│   │   ├── deployment.yml
│   │   └── service.yml
│   ├── kafka/
│   │   ├── statefulset.yml
│   │   └── service.yml
│   └── stubs/
│       ├── http-stub-deployment.yml
│       ├── http-stub-service.yml
│       ├── grpc-stub-deployment.yml
│       └── grpc-stub-service.yml
├── stubs/
│   ├── http-stub/
│   │   ├── Dockerfile
│   │   └── main.go
│   └── grpc-stub/
│       ├── Dockerfile
│       └── main.go
├── runner/
│   ├── Dockerfile
│   ├── requirements.txt
│   ├── verify.py
│   └── utils.py
├── scenarios/
│   ├── basic-health.yml
│   ├── partial-failure.yml
│   ├── full-failure.yml
│   ├── recovery.yml
│   ├── latency.yml
│   ├── labels.yml
│   ├── timeout.yml
│   └── initial-state.yml
└── run.sh
```

### Критерии завершения фазы 2

- `kubectl apply` разворачивает всю инфраструктуру в namespace `dephealth-conformance`
- Все поды переходят в Ready
- Заглушки управляемы через HTTP API (внутри кластера)
- Runner корректно парсит Prometheus-метрики
- Сценарии описаны в YAML с ожидаемыми результатами
- `run.sh` выполняет полный цикл: деплой → тесты → результаты

---

## Фаза 3: Go SDK — ядро и парсер

**Цель**: реализовать core-абстракции и парсер конфигураций.

**Статус**: [x] Завершена

### Задачи фазы 3

#### 3.1. Инициализация Go-модуля

- [x] Создать `sdk-go/` с `go.mod`
  - Имя модуля: `github.com/BigKAA/topologymetrics`
- [x] Настроить `.golangci.yml` (линтер)
- [x] Создать базовую структуру каталогов

#### 3.2. Core-абстракции (`sdk-go/dephealth/`)

- [x] `dependency.go`:
  - Структура `Dependency`: `Name`, `Type`, `Critical`, `Endpoints`, `CheckConfig`
  - Структура `Endpoint`: `Host`, `Port`, `Metadata` (map[string]string)
  - Структура `CheckConfig`: `Interval`, `Timeout`, `InitialDelay`,
    `FailureThreshold`, `SuccessThreshold`
  - Значения по умолчанию из спецификации
- [x] `checker.go`:
  - Интерфейс `HealthChecker`:

    ```go
    type HealthChecker interface {
        Check(ctx context.Context, endpoint Endpoint) error
        Type() string
    }
    ```

  - Ошибки: `ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`

#### 3.3. Парсер конфигураций (`sdk-go/dephealth/parser.go`)

- [x] Функция `ParseURL(rawURL string) ([]ParsedConnection, error)`
  - Поддержка схем: `postgres://`, `postgresql://`, `redis://`, `rediss://`,
    `amqp://`, `amqps://`, `http://`, `https://`, `grpc://`, `kafka://`
  - Извлечение host, port из URL
  - Автоопределение type из схемы
  - Обработка default ports (postgres:5432, redis:6379, etc.)
  - Обработка IPv6: `[::1]:5432`
- [x] Функция `ParseConnectionString(connStr string) (host, port string, err error)`
  - Формат `Key=Value;Key=Value`
  - Поиск ключей: `Host`, `Server`, `Data Source`, `Address`
  - Поиск ключей порта: `Port`
- [x] Функция `ParseJDBC(jdbcURL string) ([]ParsedConnection, error)`
  - Формат `jdbc:postgresql://host:port/db`
- [x] Функция `ParseParams(host, port string) (Endpoint, error)`
  - Прямые параметры (host + port)

#### 3.4. Unit-тесты для парсера

- [x] `parser_test.go`:
  - Тесты для каждого формата URL (все схемы)
  - Тесты для connection string (разные форматы ключей)
  - Тесты для JDBC URL
  - Тесты для IPv6 адресов
  - Тесты для URL без порта (default ports)
  - Тесты для URL с несколькими хостами
  - Тесты для невалидных входных данных

### Артефакты фазы 3

```text
sdk-go/
├── go.mod
├── go.sum
├── .golangci.yml
└── dephealth/
    ├── dependency.go
    ├── dependency_test.go
    ├── checker.go
    ├── parser.go
    └── parser_test.go
```

### Критерии завершения фазы 3

- `go build ./...` проходит без ошибок
- `go test ./...` — все тесты парсера проходят
- `golangci-lint run` — без предупреждений
- Покрытие парсера тестами > 90%

---

## Фаза 4: Go SDK — чекеры

**Цель**: реализовать все 8 типов проверок здоровья.

**Статус**: [x] Завершена

### Задачи фазы 4

#### 4.1. TCP Checker (`checks/tcp.go`)

- [x] Реализация: `net.DialTimeout` → закрытие
- [x] Таймаут из CheckConfig
- [x] Unit-тест с тестовым TCP-сервером

#### 4.2. HTTP Checker (`checks/http.go`)

- [x] Реализация: `http.Get` к `healthPath`
- [x] Настраиваемый `healthPath` (default `/health`)
- [x] Ожидание 2xx статуса
- [x] Поддержка TLS (InsecureSkipVerify — опционально)
- [x] Таймаут из CheckConfig
- [x] Unit-тест с `httptest.Server`

#### 4.3. gRPC Checker (`checks/grpc.go`)

- [x] Реализация: grpc.health.v1.Health/Check
- [x] Поддержка TLS и insecure
- [x] Таймаут из CheckConfig
- [x] Unit-тест с тестовым gRPC-сервером

#### 4.4. PostgreSQL Checker (`checks/postgres.go`)

- [x] Автономный режим: создание нового соединения, `SELECT 1`
- [x] Интеграция с pool: принимает `*sql.DB`, `pool.QueryContext(ctx, "SELECT 1")`
- [x] Таймаут через context
- [x] Integration-тест с Docker PostgreSQL (build tag `integration`)

#### 4.5. MySQL Checker (`checks/mysql.go`)

- [x] Аналогично PostgreSQL: автономный режим + интеграция с pool
- [x] `SELECT 1`
- [x] Integration-тест с Docker MySQL

#### 4.6. Redis Checker (`checks/redis.go`)

- [x] Автономный режим: `redis.Dial` → `PING` → `Close`
- [x] Интеграция с pool: принимает `*redis.Client` (go-redis)
- [x] Таймаут через context
- [x] Integration-тест с Docker Redis

#### 4.7. AMQP Checker (`checks/amqp.go`)

- [x] Автономный режим: `amqp.Dial` → проверка → `Close`
- [x] Поддержка vhost
- [x] Таймаут
- [x] Integration-тест с Docker RabbitMQ

#### 4.8. Kafka Checker (`checks/kafka.go`)

- [x] Автономный режим: создание клиента → `Metadata` request → закрытие
- [x] Поддержка нескольких брокеров
- [x] Таймаут
- [x] Integration-тест с Docker Kafka

### Артефакты фазы 4

```text
sdk-go/dephealth/checks/
├── tcp.go
├── tcp_test.go
├── http.go
├── http_test.go
├── grpc.go
├── grpc_test.go
├── postgres.go
├── postgres_test.go
├── mysql.go
├── mysql_test.go
├── redis.go
├── redis_test.go
├── amqp.go
├── amqp_test.go
├── kafka.go
└── kafka_test.go
```

### Критерии завершения фазы 4

- Все unit-тесты проходят: `go test ./... -short`
- Integration-тесты проходят с Docker: `go test ./... -tags integration`
- Каждый чекер корректно обрабатывает таймауты и ошибки соединения
- golangci-lint без предупреждений

---

## Фаза 5: Go SDK — метрики и планировщик

**Цель**: реализовать Prometheus exporter и периодический запуск проверок.

**Статус**: [x] Завершена

### Задачи фазы 5

#### 5.1. Prometheus Exporter (`metrics.go`)

- [x] Создание Gauge `app_dependency_health` с метками
- [x] Создание Histogram `app_dependency_latency_seconds` с метками и бакетами
- [x] Регистрация в `prometheus.DefaultRegisterer` (с возможностью указать кастомный)
- [x] Метод `SetHealth(dep, endpoint, value)` — обновить значение gauge
- [x] Метод `ObserveLatency(dep, endpoint, duration)` — записать в histogram
- [x] Корректное формирование label values из Dependency + Endpoint
- [x] Unit-тесты: проверка значений метрик через `prometheus/testutil`

#### 5.2. Check Scheduler (`scheduler.go`)

- [x] Запуск горутины для каждой зависимости
- [x] Соблюдение `initialDelay` перед первой проверкой
- [x] Периодический запуск с интервалом `checkInterval`
- [x] Передача context с таймаутом `timeout` в каждый вызов Check
- [x] Логика порогов:
  - Счётчик последовательных failures/successes
  - Переключение состояния при достижении порога
- [x] Обновление метрик после каждой проверки
- [x] Graceful shutdown: отмена context → ожидание завершения горутин
- [x] Unit-тесты с mock-чекерами

#### 5.3. Интеграция метрик и планировщика

- [x] Scheduler вызывает Exporter для обновления метрик
- [x] Замер латентности (time.Since) при каждой проверке
- [x] Логирование (slog) — результаты проверок, ошибки
- [x] Тест: полный цикл scheduler → checker → metrics

### Артефакты фазы 5

```text
sdk-go/dephealth/
├── metrics.go
├── metrics_test.go
├── scheduler.go
└── scheduler_test.go
```

### Критерии завершения фазы 5

- Метрики корректно регистрируются и обновляются
- Scheduler соблюдает интервалы и таймауты
- Логика порогов работает корректно
- Graceful shutdown без утечек горутин
- Все тесты проходят

---

## Фаза 6: Go SDK — публичный API и contrib

**Цель**: создать удобный публичный API (Option pattern) и contrib-интеграции.

**Статус**: [x] Завершена

### Задачи фазы 6

#### 6.1. Публичный API (`dephealth.go`, `options.go`)

- [x] Функция `New(opts ...Option) (*DepHealth, error)`
- [x] Option pattern для конфигурации:

  ```go
  dephealth.HTTP("payment-service", dephealth.FromURL(url), dephealth.Critical(true))
  dephealth.Postgres("postgres-main", dephealth.FromParams(host, port))
  dephealth.Redis("redis-cache", dephealth.FromURL(url))
  dephealth.GRPC("auth-service", dephealth.FromURL(url))
  dephealth.TCP("custom-service", dephealth.FromParams(host, port))
  dephealth.AMQP("rabbitmq", dephealth.FromURL(url))
  dephealth.Kafka("kafka", dephealth.FromURL(url))
  dephealth.MySQL("mysql-main", dephealth.FromURL(url))
  ```

- [x] Глобальные опции:

  ```go
  dephealth.WithCheckInterval(30 * time.Second)
  dephealth.WithTimeout(10 * time.Second)
  dephealth.WithRegisterer(customRegisterer)
  dephealth.WithLogger(slog.Default())
  ```

- [x] Методы `DepHealth`:
  - `Start(ctx context.Context) error` — запуск scheduler
  - `Stop()` — graceful shutdown
  - `Health() map[string]bool` — текущее состояние (для readiness)
- [x] Checker-обёртки (DependencyOption) для всех типов
- [x] AddDependency — хелпер для contrib-модулей
- [x] RegisterCheckerFactory — реестр фабрик чекеров (без import cycle)

#### 6.2. Contrib: database/sql (`contrib/sqldb/`)

- [x] `FromDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option`
- [x] `FromMySQLDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option`
- [x] Использует `checks.NewPostgresChecker(checks.WithPostgresDB(db))`
- [x] Unit-тесты с go-sqlmock

#### 6.3. Contrib: go-redis (`contrib/redispool/`)

- [x] `FromClient(name string, client *redis.Client, opts ...dephealth.DependencyOption) dephealth.Option`
- [x] Автоматическое извлечение host/port из `client.Options().Addr`
- [x] Использует `checks.NewRedisChecker(checks.WithRedisClient(client))`
- [x] Unit-тесты с miniredis

#### 6.4. Инфраструктурные изменения

- [x] `WithRegisterer` → `WithMetricsRegisterer` в metrics.go
- [x] `WithLogger` → `WithSchedulerLogger` в scheduler.go
- [x] `Scheduler.Health()` — реализация через states map
- [x] `checks/factories.go` — регистрация фабрик чекеров из пакета checks

### Артефакты фазы 6

```text
sdk-go/dephealth/
├── dephealth.go          # Публичный API: DepHealth, New(), Start(), Stop(), Health()
├── dephealth_test.go     # 16 тестов публичного API
├── options.go            # Option, DependencyOption, фабрики, checker-обёртки
├── checks/
│   └── factories.go      # RegisterCheckerFactory — мост между dephealth и checks
└── contrib/
    ├── sqldb/
    │   ├── sqldb.go       # FromDB, FromMySQLDB
    │   └── sqldb_test.go  # 3 теста
    └── redispool/
        ├── redispool.go   # FromClient
        └── redispool_test.go  # 3 теста
```

### Критерии завершения фазы 6

- [x] API удобен и соответствует примерам из `sdk-architecture.md`
- [x] Option pattern работает корректно
- [x] Contrib-модули компилируются и тестируются независимо
- [x] `go build ./...` — без ошибок
- [x] `go test ./... -short` — 81 тест, все проходят
- [x] `go vet ./...` — без предупреждений
- [x] `golangci-lint run` — без предупреждений

---

## Фаза 7: Тестовый сервис на Go

**Цель**: создать пилотный микросервис, использующий Go SDK, для запуска в Kubernetes.

**Статус**: [x] Завершена

### Задачи фазы 7

#### 7.1. Тестовый сервис (`test-services/go-service/`)

- [x] Простой HTTP-сервис с endpoint-ами:
  - `GET /` — основной endpoint (возвращает JSON со статусом)
  - `GET /metrics` — Prometheus-метрики (promhttp.Handler)
  - `GET /health` — health check (для Kubernetes probes)
  - `GET /health/dependencies` — детальный статус зависимостей
- [x] Использует dephealth SDK для мониторинга:
  - PostgreSQL (через `*sql.DB` — contrib/sqldb)
  - Redis (через go-redis — contrib/redispool)
  - HTTP-заглушка (автономный режим)
  - gRPC-заглушка (автономный режим)
- [x] Конфигурация через environment variables:
  - `DATABASE_URL`, `REDIS_URL`, `HTTP_STUB_URL`, `GRPC_STUB_HOST`, `GRPC_STUB_PORT`
- [x] Graceful shutdown (SIGTERM/SIGINT)
- [x] Structured logging (slog)

#### 7.2. Dockerfile

- [x] Multi-stage build: Go build → alpine
- [x] Минимальный образ (< 30 MB)
- [x] Публикация в `harbor.kryukov.lan/library/dephealth-test-go`

#### 7.3. Kubernetes-манифесты

- [x] `test-services/k8s/` — манифесты для деплоя в тестовый кластер:
  - Namespace: `dephealth-test`
  - Deployment для go-service
  - Service (ClusterIP)
  - HTTPRoute (Gateway API) — для доступа через `test1.kryukov.lan`
  - ConfigMap для конфигурации
  - PostgreSQL (StatefulSet с NFS storage)
  - Redis (Deployment)
  - HTTP/gRPC заглушки (Deployments)

#### 7.4. Верификация

- [x] Проверка `/metrics` — 4 метрики `app_dependency_health` = 1 + histogram latency
- [x] Проверка `/health/dependencies` — JSON со всеми 4 зависимостями
- [x] Деплой в Kubernetes, проверка через Gateway API (`test1.kryukov.lan`)
- [x] Toggle stub → метрика = 0, восстановление → метрика = 1
- [ ] Настройка VictoriaMetrics scrape для сбора метрик (вынесено в Фазу 10)

### Артефакты фазы 7

```text
test-services/
├── go-service/
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
└── k8s/
    ├── namespace.yml
    ├── go-service/
    │   ├── deployment.yml
    │   ├── service.yml
    │   ├── httproute.yml
    │   └── configmap.yml
    ├── postgres/
    │   ├── statefulset.yml
    │   └── service.yml
    ├── redis/
    │   ├── deployment.yml
    │   └── service.yml
    └── stubs/
        ├── http-stub.yml
        └── grpc-stub.yml
```

### Критерии завершения фазы 7

- Тестовый сервис запускается в Kubernetes
- `/metrics` возвращает корректные Prometheus-метрики
- Все зависимости мониторятся (PG, Redis, HTTP, gRPC)
- Остановка зависимости → метрика переходит в 0
- Восстановление зависимости → метрика возвращается в 1

---

## Фаза 8: Conformance-прогон Go SDK

**Цель**: прогнать все conformance-сценарии, исправить найденные проблемы.

**Статус**: [x] Завершена

### Задачи фазы 8

#### 8.1. Подготовка conformance test service

- [x] Создать `conformance/test-service/main.go` — Go-сервис с 7 зависимостями
- [x] Создать `conformance/test-service/go.mod`, `Dockerfile`
- [x] Создать k8s-манифесты `conformance/k8s/test-service/`
- [x] Исправить StorageClass (`nfs-client`) в StatefulSet postgres и kafka
- [x] Исправить RabbitMQ probe timeouts (1s → 10s)
- [x] Исправить AMQP URL (guest:guest → dephealth:dephealth-test-pass)
- [x] Обновить `verify.py` — поддержка `pre_actions`/`post_actions`
- [x] Обновить `run.sh` — деплой test-service, автоматический port-forward

#### 8.2. Прогон сценариев

- [x] `basic-health.yml` — 14/14 PASSED (все 7 зависимостей = 1)
- [x] `labels.yml` — 3/3 PASSED (все метки корректны)
- [x] `latency.yml` — 3/3 PASSED (histogram бакеты присутствуют)
- [x] `initial-state.yml` — 1/1 PASSED (значения 0 или 1)
- [x] `partial-failure.yml` — 2/2 PASSED (primary=1, replica=0)
- [x] `full-failure.yml` — 1/1 PASSED (redis-cache=0)
- [x] `recovery.yml` — 1/1 PASSED (redis-cache=1 после восстановления)
- [x] `timeout.yml` — 1/1 PASSED (http-service=0 при задержке 10s)

#### 8.3. Исправления в процессе

- [x] RabbitMQ probe timeout: `timeoutSeconds: 10`, `initialDelaySeconds: 60`
- [x] AMQP credentials: соответствие `RABBITMQ_DEFAULT_USER/PASS`
- [x] Спецификация не требовала обновлений — SDK полностью соответствует

### Критерии завершения фазы 8

- Все 8 conformance-сценариев проходят
- Спецификация и SDK согласованы
- Нет открытых issues

---

## Фаза 9: Документация и CI/CD

**Цель**: создать документацию для разработчиков и настроить CI/CD.

**Статус**: [ ] Не начата

### Задачи фазы 9

#### 9.1. Документация

- [x] `docs/quickstart/go.md` — быстрый старт для Go
  - Установка: `go get github.com/BigKAA/topologymetrics`
  - Минимальный пример (5 строк)
  - Пример с несколькими зависимостями
  - Пример с contrib-модулями
  - Конфигурация через env vars
- [x] `docs/migration/go.md` — руководство по интеграции в существующий сервис
  - Пошаговая инструкция
  - Типичные конфигурации
  - Troubleshooting
- [x] `docs/specification.md` — обзор спецификации (ссылки на spec/)
- [x] `README.md` в корне — обзор проекта, ссылки на документацию

#### 9.2. CI/CD (GitHub Actions) — *отложено, не обязательно*

- [ ] `.github/workflows/go-sdk.yml`:
  - Trigger: push/PR в `sdk-go/`
  - Шаги: lint → unit tests → integration tests (с Docker services) → build
- [ ] `.github/workflows/conformance.yml`:
  - Trigger: push/PR в `sdk-go/` или `conformance/` или `spec/`
  - Шаги: поднять инфраструктуру → запустить тестовый сервис → прогнать сценарии
- [ ] `Makefile` в корне:
  - `make lint` — запуск линтеров
  - `make test` — unit-тесты
  - `make test-integration` — integration-тесты
  - `make conformance` — conformance-тесты
  - `make build` — сборка всех артефактов
  - `make docker` — сборка Docker-образов

#### 9.3. Линтинг и качество — *отложено, не обязательно*

- [ ] Настройка golangci-lint для Go SDK
- [ ] markdownlint для документации (`.markdownlint.json` уже есть)
- [ ] Pre-commit hooks (опционально)

### Артефакты фазы 9

```text
docs/
├── quickstart/
│   └── go.md
├── migration/
│   └── go.md
└── specification.md
.github/workflows/
├── go-sdk.yml
└── conformance.yml
Makefile
README.md
```

### Критерии завершения фазы 9

- Документация покрывает все сценарии использования
- CI/CD пайплайны проходят на чистом репозитории
- Makefile работает локально
- markdownlint проходит для всех .md файлов

---

## Фаза 10: Grafana дашборды и алерты

**Цель**: создать дашборды, правила алертинга, развернуть стек мониторинга и верифицировать на тестовом кластере.

**Статус**: [x] Завершена

TODO: Доработать дашборды Grafana.

### Задачи фазы 10

#### 10.1. Grafana дашборды

- [x] **Обзор всех сервисов** (`deploy/grafana/dashboards/overview.json`):
  - Таблица: сервис, зависимость, тип, статус (цвет)
  - Цветовая кодировка: зелёный/жёлтый/красный
  - Фильтры: по namespace, по типу зависимости
  - Переменные: `$namespace`, `$service`, `$dependency_type`
- [x] **Детали сервиса** (`deploy/grafana/dashboards/service-detail.json`):
  - Список зависимостей с текущим статусом
  - График `app_dependency_health` за последние 24ч
  - Heatmap латентности `app_dependency_latency_seconds`
  - Таблица endpoint-ов (host, port, status)
- [x] **Карта зависимостей** (`deploy/grafana/dashboards/dependency-map.json`):
  - Node Graph panel (Grafana 8+)
  - Или Flowchart plugin для визуализации графа

#### 10.2. Правила алертинга

- [x] `deploy/alerting/rules.yml` — PrometheusRule / VMRule:
  - `DependencyDown` — полный отказ зависимости (critical)
  - `DependencyDegraded` — частичная деградация (warning)
  - `DependencyHighLatency` — p99 латентность > 1s (warning)
  - `DependencyFlapping` — частые переключения 0/1 (info)
  - `DependencyAbsent` — метрики отсутствуют (warning)
- [x] `deploy/alerting/inhibition-rules.yml`:
  - Подавление каскадных алертов: если корневая зависимость down,
    гасить алерты от зависимых сервисов

#### 10.3. Деплой мониторинга (provisioning)

- [x] ConfigMap / provisioning для Grafana дашбордов

#### 10.4. Стек мониторинга в Kubernetes

Установка и настройка в namespace `dephealth-monitoring`.
Все компоненты в одноподовом (single-node) режиме для тестирования.

- [x] **VictoriaMetrics** (single-node):
  - StatefulSet-манифест, PVC на `nfs-client` StorageClass
  - Хранение: 2Gi, retention: 7 дней
  - Scrape config: тестовый сервис Go (`dephealth-test` namespace)
  - Endpoint: `http://victoriametrics:8428`
- [x] **VMAlert**:
  - Загрузка правил из ConfigMap (5 алертов)
  - Отправка алертов в Alertmanager
- [x] **Alertmanager** (single-pod):
  - Deployment + Service
  - Конфигурация: `inhibit_rules` (5 правил подавления)
  - Receiver: `webhook` (для тестовой верификации) + `null` (silence)
  - Web UI для просмотра алертов
- [x] **Grafana** (single-pod):
  - Deployment с provisioning через ConfigMaps
  - Datasource → VictoriaMetrics, dashboards → ConfigMap
  - Доступ через Gateway API (`grafana.kryukov.lan`)
  - Три дашборда dephealth загружены автоматически
- [x] **Scrape-конфигурация**:
  - VictoriaMetrics scrape target: `go-service.dephealth-test.svc:8080/metrics`
  - Интервал scrape: 15s
  - Метки: `namespace`, `job` (имя сервиса)

#### 10.5. Верификация и тестирование

- [x] **Метрики доступны в VictoriaMetrics**:
  - Запрос `app_dependency_health` возвращает данные
  - Все 4 зависимости тестового сервиса видны (postgres, redis, http-stub, grpc-stub)
  - Histogram `app_dependency_latency_seconds` содержит бакеты
- [x] **Дашборды отображают данные**:
  - Overview: таблица заполнена, stat-панели показывают числа
  - Service Detail: timeline и heatmap отображают историю
  - Dependency Map: Node Graph показывает связи (или текстовая панель с инструкцией)
- [x] **Тестирование алертов — DependencyDown**:
  - Масштабировать Redis в 0 (`kubectl scale deployment redis --replicas=0`)
  - Подождать 1 минуту → алерт `DependencyDown` (critical) в Alertmanager
  - Проверить: дашборд Overview показывает Redis = DOWN (красный)
- [x] **Тестирование алертов — DependencyDegraded**:
  - Если есть replica PG — масштабировать в 0
  - Подождать 2 минуты → алерт `DependencyDegraded` (warning)
- [x] **Тестирование алертов — DependencyHighLatency**:
  - Включить задержку в HTTP-заглушке: `/admin/delay?ms=2000`
  - Подождать 5 минут → алерт `DependencyHighLatency` (warning)
- [x] **Тестирование алертов — DependencyFlapping**:
  - Быстро переключать HTTP-заглушку: `/admin/toggle` каждые 30 секунд × 6 раз
  - Подождать → алерт `DependencyFlapping` (info)
- [x] **Тестирование подавления каскадов**:
  - Масштабировать Redis в 0 → `DependencyDown` = active
  - Проверить: `DependencyFlapping` и `DependencyHighLatency` для Redis подавлены
  - Проверить inhibition-rules в Alertmanager UI
- [x] **Восстановление**:
  - Масштабировать Redis обратно в 1
  - Подождать → алерт `DependencyDown` = resolved
  - Дашборд Overview: Redis = UP (зелёный)

### Артефакты фазы 10

```text
deploy/
├── grafana/
│   ├── dashboards/
│   │   ├── overview.json
│   │   ├── service-detail.json
│   │   └── dependency-map.json
│   └── provisioning/
│       └── dashboards.yml
├── alerting/
│   ├── rules.yml
│   └── inhibition-rules.yml
└── monitoring/
    ├── namespace.yml
    ├── victoriametrics/
    │   ├── statefulset.yml
    │   ├── service.yml
    │   └── scrape-config.yml
    ├── vmalert/
    │   ├── deployment.yml
    │   └── service.yml
    ├── alertmanager/
    │   ├── deployment.yml
    │   ├── service.yml
    │   └── configmap.yml
    └── grafana/
        ├── deployment.yml
        ├── service.yml
        ├── httproute.yml
        └── configmap-datasource.yml
```

### Критерии завершения фазы 10

- VictoriaMetrics собирает метрики от тестового сервиса
- Grafana дашборды отображают данные
- Алерты срабатывают при остановке зависимости (DependencyDown)
- Алерты срабатывают при деградации (DependencyDegraded)
- Алерты срабатывают при высокой латентности (DependencyHighLatency)
- Каскадное подавление работает корректно в Alertmanager
- Восстановление зависимости → алерт resolved, дашборд обновлён

---

## Фаза 11: Инфраструктура контейнерной разработки

**Цель**: создать единый Makefile-интерфейс для сборки/тестирования SDK через Docker (без локальных компиляторов), задокументировать конвенции и подготовить scrape-конфигурацию для будущих сервисов.

**Статус**: [x] Завершена

### Задачи фазы 11

#### 11.1. Makefile для Go SDK (`sdk-go/Makefile`)

- [x] Цели: `build`, `test`, `test-coverage`, `lint`, `fmt`, `clean`
- [x] Каждая цель — `docker run` с монтированием `sdk-go/` и Go-кэшем:
  - Docker volume: `dephealth-go-cache` (`/go/pkg/mod` + build cache)
  - Образ: `golang:1.25-alpine` (или `golang:1.25`)
- [x] Цели для Docker-образов:
  - `image` — собрать Docker-образ тестового сервиса (`test-services/go-service/`)
  - `push` — загрузить в `harbor.kryukov.lan/library/dephealth-test-go:latest`
- [x] Переменные: `REGISTRY`, `IMAGE_TAG`, `GO_VERSION`
- [x] Проверка: `cd sdk-go && make test` работает без локального Go

#### 11.2. Конвенции Makefile (`plans/makefile-conventions.md`)

- [x] Документ с правилами для всех SDK Makefile:
  - Обязательные цели: `build`, `test`, `test-coverage`, `lint`, `fmt`, `image`, `push`, `clean`
  - Формат переменных: `REGISTRY`, `IMAGE_TAG`, `<LANG>_VERSION`
  - Docker volume naming: `dephealth-{lang}-cache`
  - Docker-образы сборки: `golang:X`, `python:X`, `maven:X`, `mcr.microsoft.com/dotnet/sdk:X`
  - Формат `Dockerfile.dev` — среда для тестов/линтинга
  - Registry: `harbor.kryukov.lan/library/dephealth-test-{lang}:latest`
- [x] Примеры вызовов для каждого языка

#### 11.3. Подготовка scrape-конфигурации VictoriaMetrics

- [x] Добавить закомментированные scrape-targets для будущих сервисов:
  - `python-service.dephealth-test.svc:8080/metrics`
  - `java-service.dephealth-test.svc:8080/metrics`
  - `csharp-service.dephealth-test.svc:8080/metrics`
- [x] Обновить `deploy/monitoring/victoriametrics/scrape-config.yml`

#### 11.4. Обновление `conformance/run.sh`

- [x] Добавить параметр `--lang go|python|java|csharp|all`
- [x] При `--lang go` — текущее поведение (без изменений)
- [x] При `--lang all` — последовательный прогон всех языков
- [x] Каждый язык: свой тестовый сервис в `conformance/test-service-{lang}/`
- [x] Существующий `conformance/test-service/` — Go (не переименовываем)

### Артефакты фазы 11

```text
sdk-go/
└── Makefile                          # Docker-обёртки для build/test/lint

plans/
└── makefile-conventions.md           # Конвенции Makefile для всех SDK

deploy/monitoring/victoriametrics/
└── scrape-config.yml                 # + закомментированные targets

conformance/
└── run.sh                            # + параметр --lang
```

### Критерии завершения фазы 11

- `cd sdk-go && make test` проходит без локального Go
- `cd sdk-go && make lint` проходит без локального golangci-lint
- `cd sdk-go && make image` собирает Docker-образ тестового сервиса
- `conformance/run.sh --lang go` работает как раньше
- `plans/makefile-conventions.md` содержит правила и примеры для всех 4 языков

---

## Фаза 12: Python SDK — Core + Checkers ✅

**Цель**: реализовать ядро Python SDK — абстракции, парсер, все 8 чекеров, Prometheus exporter, планировщик.

**Статус**: [ ] Не начата

### Задачи фазы 12

#### 12.1. Инициализация проекта (`sdk-python/`)

- [x] `pyproject.toml` — пакет `dephealth`:
  - Зависимости: `prometheus-client`, `aiohttp` (для HTTP checker)
  - Опциональные: `asyncpg`, `aiomysql`, `redis[hiredis]`, `aio-pika`, `aiokafka`, `grpcio`
  - Dev-зависимости: `pytest`, `pytest-asyncio`, `pytest-cov`, `ruff`, `mypy`
- [x] ~~`Dockerfile.dev`~~ — не нужен, используется `python:3.12-slim` напрямую через Makefile
- [x] `Makefile` — по конвенциям из фазы 11:
  - Docker volume: `dephealth-python-cache` (pip cache)
  - Цели: `build`, `test`, `test-coverage`, `lint`, `fmt`, `clean`

#### 12.2. Core-абстракции (`sdk-python/dephealth/`)

- [x] `dependency.py`:
  - Dataclass `Dependency`: `name`, `type`, `critical`, `endpoints`, `check_config`
  - Dataclass `Endpoint`: `host`, `port`, `metadata` (dict)
  - Dataclass `CheckConfig`: `interval`, `timeout`, `initial_delay`,
    `failure_threshold`, `success_threshold`
  - Значения по умолчанию из спецификации (15s, 5s, 5s, 1, 1)
- [x] `checker.py`:
  - Protocol `HealthChecker`:

    ```python
    class HealthChecker(Protocol):
        async def check(self, endpoint: Endpoint) -> None: ...
        def type(self) -> str: ...
    ```

  - Исключения: `CheckTimeoutError`, `ConnectionRefusedError`, `UnhealthyError`

#### 12.3. Парсер конфигураций (`sdk-python/dephealth/parser.py`)

- [x] Функция `parse_url(raw_url: str) -> list[ParsedConnection]`
  - Поддержка схем: `postgres://`, `postgresql://`, `redis://`, `rediss://`,
    `amqp://`, `amqps://`, `http://`, `https://`, `grpc://`, `kafka://`
  - Извлечение host, port, автоопределение type
  - Default ports (postgres:5432, redis:6379, и т.д.)
  - IPv6: `[::1]:5432`
- [x] Функция `parse_connection_string(conn_str: str) -> tuple[str, str]`
- [x] Функция `parse_jdbc(jdbc_url: str) -> list[ParsedConnection]`
- [x] Функция `parse_params(host: str, port: str) -> Endpoint`
- [x] Unit-тесты: `tests/test_parser.py`

#### 12.4. Health Checkers (`sdk-python/dephealth/checks/`)

- [x] `tcp.py` — TCPChecker: `asyncio.open_connection` → закрытие
- [x] `http.py` — HTTPChecker: `aiohttp.ClientSession.get` к healthPath, ожидание 2xx
- [x] `grpc.py` — GRPCChecker: `grpc.aio` Health/Check
- [x] `postgres.py` — PostgresChecker:
  - Автономный: `asyncpg.connect` → `SELECT 1` → закрытие
  - Pool-режим: принимает `asyncpg.Pool`
- [x] `mysql.py` — MySQLChecker:
  - Автономный: `aiomysql.connect` → `SELECT 1` → закрытие
  - Pool-режим: принимает `aiomysql.Pool`
- [x] `redis.py` — RedisChecker:
  - Автономный: `redis.asyncio.Redis.from_url` → `PING` → закрытие
  - Pool-режим: принимает `redis.asyncio.Redis`
- [x] `amqp.py` — AMQPChecker: `aio_pika.connect_robust` → закрытие
- [x] `kafka.py` — KafkaChecker: `aiokafka.AIOKafkaClient` → metadata → закрытие
- [x] Unit-тесты для каждого чекера: `tests/test_checks/`
  - Мок-серверы через `pytest-asyncio`

#### 12.5. Prometheus Exporter (`sdk-python/dephealth/metrics.py`)

- [x] Класс `MetricsExporter`:
  - Gauge `app_dependency_health` с метками `dependency`, `type`, `host`, `port`
  - Histogram `app_dependency_latency_seconds` с теми же метками
  - Бакеты: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
  - Методы: `set_health(dep, endpoint, value)`, `observe_latency(dep, endpoint, duration)`
  - Поддержка `CollectorRegistry` (кастомный или default)
- [x] Unit-тесты: `tests/test_metrics.py`

#### 12.6. Check Scheduler (`sdk-python/dephealth/scheduler.py`)

- [x] Класс `CheckScheduler`:
  - Двойная поддержка: asyncio (основной) и threading (fallback)
  - asyncio: `asyncio.create_task` для каждой зависимости
  - threading: `threading.Timer` для среды без event loop
  - `initial_delay` перед первой проверкой
  - Логика порогов (failure_threshold, success_threshold)
  - Обновление метрик после каждой проверки
  - `start()` / `stop()` — запуск/остановка
  - `health() -> dict[str, bool]` — текущее состояние
- [x] Unit-тесты: `tests/test_scheduler.py`

#### 12.7. Публичный API (`sdk-python/dephealth/api.py`)

- [x] Класс `DependencyHealth`:
  - Конструктор с опциями:

    ```python
    dh = DependencyHealth(
        http_check("payment", url="http://payment:8080"),
        postgres_check("db", url="postgres://..."),
        redis_check("cache", url="redis://..."),
        check_interval=timedelta(seconds=30),
        timeout=timedelta(seconds=10),
    )
    ```

  - Фабрики: `http_check()`, `grpc_check()`, `tcp_check()`, `postgres_check()`,
    `mysql_check()`, `redis_check()`, `amqp_check()`, `kafka_check()`
  - `async start()` / `async stop()` (asyncio)
  - `start_sync()` / `stop_sync()` (threading fallback)
  - `health() -> dict[str, bool]`
- [x] Unit-тесты: `tests/test_api.py`

#### 12.8. Линтинг и типизация

- [x] `ruff check .` — без ошибок
- [x] `mypy dephealth/ --strict` — без ошибок (или минимум)
- [x] `make lint` проходит в Docker

### Артефакты фазы 12

```text
sdk-python/
├── pyproject.toml
├── Dockerfile.dev
├── Makefile
├── dephealth/
│   ├── __init__.py
│   ├── dependency.py
│   ├── checker.py
│   ├── parser.py
│   ├── metrics.py
│   ├── scheduler.py
│   ├── api.py
│   └── checks/
│       ├── __init__.py
│       ├── tcp.py
│       ├── http.py
│       ├── grpc.py
│       ├── postgres.py
│       ├── mysql.py
│       ├── redis.py
│       ├── amqp.py
│       └── kafka.py
└── tests/
    ├── test_parser.py
    ├── test_metrics.py
    ├── test_scheduler.py
    ├── test_api.py
    └── test_checks/
        ├── test_tcp.py
        ├── test_http.py
        ├── test_grpc.py
        ├── test_postgres.py
        ├── test_mysql.py
        ├── test_redis.py
        ├── test_amqp.py
        └── test_kafka.py
```

### Критерии завершения фазы 12

- `cd sdk-python && make test` — все тесты проходят в Docker
- `cd sdk-python && make lint` — ruff + mypy без ошибок
- `cd sdk-python && make test-coverage` — покрытие парсера и чекеров > 80%
- Публичный API удобен и аналогичен Go SDK
- Двойная поддержка asyncio/threading работает

---

## Фаза 13: Python SDK — FastAPI + Conformance

**Цель**: создать FastAPI-интеграцию, тестовый сервис, прогнать conformance-сценарии.

**Статус**: [x] Завершена

### Задачи фазы 13

#### 13.1. FastAPI-интеграция (`sdk-python/dephealth_fastapi/`)

- [x] Пакет `dephealth-fastapi` (отдельный модуль в `pyproject.toml` или extras):
  - Middleware: автоматическое подключение `/metrics` endpoint
  - Lifespan: `dephealth_lifespan()` — запуск/остановка DependencyHealth
  - Endpoint `/health/dependencies` — JSON со статусом всех зависимостей
  - Интеграция с `prometheus_client` ASGI middleware
- [x] Пример использования:

  ```python
  from dephealth_fastapi import DepHealthMiddleware, dephealth_lifespan

  app = FastAPI(lifespan=dephealth_lifespan(
      http_check("payment", url="http://payment:8080"),
      postgres_check("db", url=os.environ["DATABASE_URL"]),
  ))
  app.add_middleware(DepHealthMiddleware)
  ```

- [x] Unit-тесты: `tests/test_fastapi.py`

#### 13.2. Тестовый сервис (`test-services/python-service/`)

- [x] FastAPI-сервис с 4 зависимостями:
  - PostgreSQL (через `asyncpg`)
  - Redis (через `redis.asyncio`)
  - HTTP-заглушка (автономный режим)
  - gRPC-заглушка (автономный режим)
- [x] Endpoint-ы:
  - `GET /` — JSON со статусом
  - `GET /metrics` — Prometheus-метрики
  - `GET /health` — health check для Kubernetes probes
  - `GET /health/dependencies` — детальный статус
- [x] Конфигурация через env vars: `DATABASE_URL`, `REDIS_URL`, `HTTP_STUB_URL`,
  `GRPC_STUB_HOST`, `GRPC_STUB_PORT`
- [x] `Dockerfile` — multi-stage (builder → python:3.12-slim)
- [ ] Минимальный образ (< 200 MB) — проверить при сборке
- [ ] Публикация: `harbor.kryukov.lan/library/dephealth-test-python:latest`

#### 13.3. Kubernetes-манифесты (`test-services/k8s/python-service/`)

- [x] Deployment, Service (ClusterIP), ConfigMap, HTTPRoute
- [x] Повторяет паттерн `test-services/k8s/go-service/`
- [x] Namespace: `dephealth-test` (общий с Go)

#### 13.4. Conformance test service (`conformance/test-service-python/`)

- [x] FastAPI-сервис с 7 зависимостями:
  - PostgreSQL primary, PostgreSQL replica, Redis, RabbitMQ, Kafka,
    HTTP-заглушка, gRPC-заглушка
- [x] `Dockerfile`, `requirements.txt`
- [x] K8s-манифесты: `conformance/test-service-python/k8s/`
- [ ] Публикация: `harbor.kryukov.lan/library/dephealth-conformance-python:latest`

#### 13.5. Conformance-прогон

- [ ] `conformance/run.sh --lang python`:
  - Деплой инфраструктуры (общая с Go)
  - Деплой `test-service-python`
  - Прогон всех 8 сценариев
- [ ] Все сценарии проходят:
  - `basic-health.yml` — все метрики = 1
  - `labels.yml` — метки корректны
  - `latency.yml` — histogram бакеты присутствуют
  - `initial-state.yml` — значения 0 или 1
  - `partial-failure.yml` — primary=1, replica=0
  - `full-failure.yml` — redis-cache=0
  - `recovery.yml` — redis-cache=1 после восстановления
  - `timeout.yml` — http-service=0 при задержке

#### 13.6. Раскомментировать scrape-target

- [x] Раскомментировать `python-service` в VictoriaMetrics scrape-config
- [ ] Проверить сбор метрик в Grafana

### Артефакты фазы 13

```text
sdk-python/
└── dephealth_fastapi/
    ├── __init__.py
    ├── middleware.py
    ├── lifespan.py
    └── endpoints.py

test-services/
├── python-service/
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
└── k8s/
    └── python-service/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml

conformance/
├── test-service-python/
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
└── k8s/
    └── test-service-python/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml
```

### Критерии завершения фазы 13

- Тестовый сервис запускается в Kubernetes, `/metrics` возвращает корректные метрики
- Все 8 conformance-сценариев проходят для Python SDK
- Метрики идентичны Go SDK (имена, метки, HELP, бакеты)
- FastAPI-интеграция покрыта тестами
- VictoriaMetrics собирает метрики Python-сервиса, дашборды отображают данные

---

## Фаза 14: Java SDK — Core + Spring Boot + Conformance ✅

**Цель**: реализовать полный Java SDK — ядро, 8 чекеров, Spring Boot starter, Micrometer bridge,
тестовый сервис и conformance-прогон.

**Статус**: [ ] Не начата

### Задачи фазы 14

#### 14.1. Инициализация проекта (`sdk-java/`)

- [ ] Parent POM (`sdk-java/pom.xml`):
  - GroupId: `com.github.bigkaa`
  - Модули: `dephealth-core`, `dephealth-spring-boot-starter`, `dephealth-micrometer`
  - Java 17+, Maven 3.9+
- [ ] `Dockerfile.dev` — Maven среда для тестов:
  - Базовый образ: `maven:3.9-eclipse-temurin-17`
  - Docker volume: `dephealth-maven-cache` (`~/.m2/repository`)
- [ ] `Makefile` — по конвенциям из фазы 11:
  - Цели: `build`, `test`, `test-coverage`, `lint`, `fmt`, `image`, `push`, `clean`
  - `build` → `mvn package -DskipTests`
  - `test` → `mvn test`
  - `lint` → `mvn checkstyle:check` или `spotbugs`

#### 14.2. Core-модуль (`sdk-java/dephealth-core/`)

- [ ] Модель (`model/`):
  - `Dependency.java`: name, type, critical, endpoints, checkConfig
  - `Endpoint.java`: host, port, metadata (Map)
  - `CheckConfig.java`: interval, timeout, initialDelay, failureThreshold, successThreshold
  - Builder pattern для Dependency
- [ ] Интерфейс `HealthChecker.java`:

  ```java
  public interface HealthChecker {
      void check(Endpoint endpoint) throws CheckException;
      String type();
  }
  ```

- [ ] Исключения: `CheckTimeoutException`, `ConnectionRefusedException`, `UnhealthyException`

#### 14.3. Парсер конфигураций (`sdk-java/dephealth-core/`)

- [ ] `ConfigParser.java`:
  - `parseUrl(String rawUrl) -> List<ParsedConnection>`
  - `parseConnectionString(String connStr) -> Pair<String, String>`
  - `parseJdbc(String jdbcUrl) -> List<ParsedConnection>`
  - `parseParams(String host, String port) -> Endpoint`
- [ ] Unit-тесты: все форматы, все схемы, IPv6, default ports

#### 14.4. Health Checkers (`sdk-java/dephealth-core/checks/`)

- [ ] `TcpChecker.java` — `Socket.connect()`
- [ ] `HttpChecker.java` — `HttpClient.newHttpClient()`, GET к healthPath, ожидание 2xx
- [ ] `GrpcChecker.java` — gRPC Health/Check (io.grpc)
- [ ] `JdbcChecker.java` — `SELECT 1` через `DataSource` или новое соединение
  - Поддержка PostgreSQL и MySQL через JDBC
- [ ] `RedisChecker.java` — Jedis/Lettuce `PING`
- [ ] `AmqpChecker.java` — RabbitMQ ConnectionFactory → connect → close
- [ ] `KafkaChecker.java` — AdminClient → metadata → close
- [ ] Unit-тесты для каждого чекера

#### 14.5. Prometheus Exporter (`sdk-java/dephealth-core/`)

- [ ] `PrometheusExporter.java`:
  - Gauge `app_dependency_health` (simpleclient)
  - Histogram `app_dependency_latency_seconds`
  - Бакеты: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
  - `setHealth()`, `observeLatency()`
  - Поддержка `CollectorRegistry`
- [ ] Unit-тесты

#### 14.6. Check Scheduler (`sdk-java/dephealth-core/`)

- [ ] `CheckScheduler.java`:
  - `ScheduledExecutorService` для периодических проверок
  - `initialDelay`, `checkInterval`, таймауты через `Future.get(timeout)`
  - Логика порогов (failureThreshold, successThreshold)
  - `start()` / `stop()` (graceful shutdown)
  - `health() -> Map<String, Boolean>`
- [ ] Unit-тесты с mock-чекерами

#### 14.7. Spring Boot Starter (`sdk-java/dephealth-spring-boot-starter/`)

- [ ] Auto-configuration:
  - `DepHealthAutoConfiguration.java` — `@EnableConfigurationProperties`
  - `DepHealthProperties.java` — `spring.dephealth.*` properties
  - Автосканирование `DataSource`, `RedisTemplate`, etc.
- [ ] `HealthIndicator` — интеграция с `/actuator/health`
- [ ] Endpoint `/actuator/dependencies` — детальный статус
- [ ] `META-INF/spring/org.springframework.boot.autoconfigure.AutoConfiguration.imports`
- [ ] Unit-тесты с `@SpringBootTest`

#### 14.8. Micrometer Bridge (`sdk-java/dephealth-micrometer/`)

- [ ] `MicrometerExporter.java`:
  - Мост между `PrometheusExporter` и `MeterRegistry`
  - Автоматическая регистрация в Spring Boot
- [ ] Unit-тесты

#### 14.9. Тестовый сервис (`test-services/java-service/`)

- [ ] Spring Boot сервис с 4 зависимостями:
  - PostgreSQL (через `DataSource` / Spring Data)
  - Redis (через `RedisTemplate`)
  - HTTP-заглушка (автономный)
  - gRPC-заглушка (автономный)
- [ ] Endpoint-ы: `/`, `/metrics`, `/health`, `/health/dependencies`
- [ ] `Dockerfile` — multi-stage (Maven build → eclipse-temurin:17-jre-alpine)
- [ ] Публикация: `harbor.kryukov.lan/library/dephealth-test-java:latest`
- [ ] K8s-манифесты: `test-services/k8s/java-service/`

#### 14.10. Conformance test service и прогон

- [ ] `conformance/test-service-java/`:
  - Spring Boot сервис с 7 зависимостями
  - `Dockerfile`, `pom.xml`
  - K8s-манифесты: `conformance/k8s/test-service-java/`
- [ ] `conformance/run.sh --lang java`:
  - Деплой, прогон всех 8 сценариев
- [ ] Все сценарии проходят
- [ ] Раскомментировать scrape-target в VictoriaMetrics

### Артефакты фазы 14

```text
sdk-java/
├── pom.xml                           # Parent POM
├── Dockerfile.dev
├── Makefile
├── dephealth-core/
│   ├── pom.xml
│   └── src/main/java/com/github/bigkaa/dephealth/
│       ├── model/
│       │   ├── Dependency.java
│       │   ├── Endpoint.java
│       │   └── CheckConfig.java
│       ├── HealthChecker.java
│       ├── ConfigParser.java
│       ├── PrometheusExporter.java
│       ├── CheckScheduler.java
│       └── checks/
│           ├── TcpChecker.java
│           ├── HttpChecker.java
│           ├── GrpcChecker.java
│           ├── JdbcChecker.java
│           ├── RedisChecker.java
│           ├── AmqpChecker.java
│           └── KafkaChecker.java
├── dephealth-spring-boot-starter/
│   ├── pom.xml
│   └── src/main/java/com/github/bigkaa/dephealth/spring/
│       ├── DepHealthAutoConfiguration.java
│       ├── DepHealthProperties.java
│       └── DepHealthHealthIndicator.java
└── dephealth-micrometer/
    ├── pom.xml
    └── src/main/java/com/github/bigkaa/dephealth/micrometer/
        └── MicrometerExporter.java

test-services/
├── java-service/
│   ├── pom.xml
│   ├── src/main/java/.../Application.java
│   └── Dockerfile
└── k8s/
    └── java-service/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml

conformance/
├── test-service-java/
│   ├── pom.xml
│   ├── src/main/java/.../ConformanceApplication.java
│   └── Dockerfile
└── k8s/
    └── test-service-java/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml
```

### Критерии завершения фазы 14

- `cd sdk-java && make test` — все тесты проходят в Docker
- `cd sdk-java && make lint` — checkstyle/spotbugs без ошибок
- Spring Boot auto-configuration корректно обнаруживает зависимости
- Тестовый сервис запускается в Kubernetes, `/metrics` корректен
- Все 8 conformance-сценариев проходят
- Метрики идентичны Go и Python SDK
- VictoriaMetrics собирает метрики, Grafana дашборды отображают данные

---

## Фаза 15: C# SDK — Core + ASP.NET + Conformance

**Цель**: реализовать полный C# SDK — ядро, 8 чекеров, ASP.NET интеграцию,
тестовый сервис и conformance-прогон.

**Статус**: [x] Завершена

### Задачи фазы 15

#### 15.1. Инициализация проекта (`sdk-csharp/`)

- [x] Solution: `DepHealth.sln`
  - Проекты: `DepHealth.Core`, `DepHealth.AspNetCore`, `DepHealth.EntityFramework`
  - .NET 8 LTS, C# 12
- [x] Docker через Harbor MCR proxy: `harbor.kryukov.lan/mcr/dotnet/sdk:8.0`
  - Docker volume: `dephealth-csharp-cache` (NuGet cache)
- [x] `Makefile` — по конвенциям из фазы 11:
  - `build` → `dotnet build`
  - `test` → `dotnet test`
  - `lint` → `dotnet format --verify-no-changes`

#### 15.2. Core-проект (`sdk-csharp/DepHealth.Core/`)

- [x] Модель:
  - `Dependency.cs`: Name, Type, Critical, Endpoints, CheckConfig
  - `Endpoint.cs`: Host, Port, Metadata (Dictionary)
  - `CheckConfig.cs`: Interval, Timeout, InitialDelay, FailureThreshold, SuccessThreshold
  - Builder pattern
- [x] Интерфейс `IHealthChecker`:

  ```csharp
  public interface IHealthChecker
  {
      Task CheckAsync(Endpoint endpoint, CancellationToken ct);
      string Type { get; }
  }
  ```

- [x] Исключения: `CheckTimeoutException`, `ConnectionRefusedException`, `UnhealthyException`

#### 15.3. Парсер конфигураций

- [x] `ConfigParser.cs`:
  - `ParseUrl()`, `ParseConnectionString()`, `ParseJdbc()`, `ParseParams()`
  - Аналогично Go/Python/Java
- [x] Unit-тесты (xUnit)

#### 15.4. Health Checkers (`sdk-csharp/DepHealth.Core/Checks/`)

- [x] `TcpChecker.cs` — `TcpClient.ConnectAsync()`
- [x] `HttpChecker.cs` — `HttpClient.GetAsync()`, ожидание 2xx
- [x] `GrpcChecker.cs` — gRPC Health/Check (Grpc.HealthCheck)
- [x] `PostgresChecker.cs` — Npgsql `SELECT 1`
  - Автономный: новое соединение
  - Pool-режим: принимает `NpgsqlDataSource`
- [x] `MySqlChecker.cs` — MySqlConnector `SELECT 1`
- [x] `RedisChecker.cs` — StackExchange.Redis `PING`
- [x] `AmqpChecker.cs` — RabbitMQ.Client connect → close
- [x] `KafkaChecker.cs` — Confluent.Kafka AdminClient → metadata
- [x] Unit-тесты для каждого чекера (xUnit + Moq)

#### 15.5. Prometheus Exporter

- [x] `PrometheusExporter.cs` (prometheus-net):
  - Gauge `app_dependency_health`
  - Histogram `app_dependency_latency_seconds`
  - Бакеты: `[0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0]`
- [x] Unit-тесты

#### 15.6. Check Scheduler

- [x] `CheckScheduler.cs`:
  - Task-based (`Task.Delay` + `CancellationToken`)
  - `initialDelay`, `checkInterval`, таймауты
  - Логика порогов
  - `Start()` / `Stop()`
  - `Health() -> Dictionary<string, bool>`
- [x] Unit-тесты

#### 15.7. ASP.NET интеграция (`sdk-csharp/DepHealth.AspNetCore/`)

- [x] `IHostedService` — запуск/остановка CheckScheduler с приложением
- [x] `IHealthCheck` — интеграция с `/health` (Microsoft.Extensions.Diagnostics.HealthChecks)
- [x] `ServiceCollectionExtensions`:

  ```csharp
  builder.Services.AddDepHealth(dh => {
      dh.AddHttp("payment", "http://payment:8080");
      dh.AddPostgres("db", connectionString);
      dh.AddRedis("cache", "redis://...");
  });
  ```

- [x] Middleware: `/metrics` endpoint (prometheus-net.AspNetCore)
- [x] Endpoint: `/health/dependencies` — JSON
- [x] Unit-тесты (3 теста)

#### 15.8. Entity Framework интеграция (`sdk-csharp/DepHealth.EntityFramework/`)

- [x] Extension: `AddNpgsqlFromContext<TContext>()` — автоматический PostgreSQL checker
- [x] Извлечение connection string из DbContext

#### 15.9. Тестовый сервис (`test-services/csharp-service/`)

- [x] ASP.NET Minimal API с 4 зависимостями:
  - PostgreSQL (через Npgsql)
  - Redis (через StackExchange.Redis)
  - HTTP-заглушка (автономный)
  - gRPC-заглушка (автономный)
- [x] Endpoint-ы: `/`, `/metrics`, `/health`, `/health/dependencies`
- [x] `Dockerfile` — multi-stage (dotnet publish → aspnet:8.0-alpine)
- [x] Публикация: `harbor.kryukov.lan/library/dephealth-test-csharp:latest`
- [x] K8s-манифесты: `test-services/k8s/csharp-service/`
- [x] HTTPRoute: `test4.kryukov.lan`

#### 15.10. Conformance test service и прогон

- [x] `conformance/test-service-csharp/`:
  - ASP.NET Minimal API с 7 зависимостями
  - `Dockerfile`, `.csproj`
  - K8s-манифесты: `conformance/test-service-csharp/k8s/`
- [x] `conformance/run.sh --lang csharp`:
  - Деплой, прогон всех 8 сценариев
- [x] Все 8 сценариев проходят (basic-health, full-failure, initial-state, labels, latency, partial-failure, recovery, timeout)
- [x] Активирован scrape-target в VictoriaMetrics

### Артефакты фазы 15

```text
sdk-csharp/
├── DepHealth.sln
├── Dockerfile.dev
├── Makefile
├── DepHealth.Core/
│   ├── DepHealth.Core.csproj
│   ├── Models/
│   │   ├── Dependency.cs
│   │   ├── Endpoint.cs
│   │   └── CheckConfig.cs
│   ├── IHealthChecker.cs
│   ├── ConfigParser.cs
│   ├── PrometheusExporter.cs
│   ├── CheckScheduler.cs
│   └── Checks/
│       ├── TcpChecker.cs
│       ├── HttpChecker.cs
│       ├── GrpcChecker.cs
│       ├── PostgresChecker.cs
│       ├── MySqlChecker.cs
│       ├── RedisChecker.cs
│       ├── AmqpChecker.cs
│       └── KafkaChecker.cs
├── DepHealth.AspNetCore/
│   ├── DepHealth.AspNetCore.csproj
│   ├── DepHealthHostedService.cs
│   ├── DepHealthHealthCheck.cs
│   ├── ServiceCollectionExtensions.cs
│   └── DepHealthEndpoints.cs
├── DepHealth.EntityFramework/
│   ├── DepHealth.EntityFramework.csproj
│   └── EntityFrameworkExtensions.cs
└── tests/
    ├── DepHealth.Core.Tests/
    └── DepHealth.AspNetCore.Tests/

test-services/
├── csharp-service/
│   ├── Program.cs
│   ├── csharp-service.csproj
│   └── Dockerfile
└── k8s/
    └── csharp-service/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml

conformance/
├── test-service-csharp/
│   ├── Program.cs
│   ├── test-service-csharp.csproj
│   └── Dockerfile
└── k8s/
    └── test-service-csharp/
        ├── deployment.yml
        ├── service.yml
        └── configmap.yml
```

### Критерии завершения фазы 15

- `cd sdk-csharp && make test` — все тесты проходят в Docker
- `cd sdk-csharp && make lint` — dotnet format без ошибок
- ASP.NET интеграция корректно стартует/останавливает checker
- Тестовый сервис запускается в Kubernetes, `/metrics` корректен
- Все 8 conformance-сценариев проходят
- Метрики идентичны Go, Python и Java SDK
- VictoriaMetrics собирает метрики, Grafana дашборды отображают данные

---

## Фаза 16: Кросс-языковая документация + Релиз v1.0

**Цель**: создать документацию для всех SDK, провести финальную кросс-языковую верификацию, подготовить релиз.

**Статус**: [~] В процессе (16.1–16.3 завершены)

### Задачи фазы 16

#### 16.1. Документация для каждого языка ✅

- [x] `docs/quickstart/python.md` — быстрый старт Python SDK
- [x] `docs/quickstart/java.md` — быстрый старт Java SDK
- [x] `docs/quickstart/csharp.md` — быстрый старт C# SDK
- [x] `docs/migration/python.md` — руководство по интеграции в существующий сервис
- [x] `docs/migration/java.md` — руководство по интеграции
- [x] `docs/migration/csharp.md` — руководство по интеграции

#### 16.2. Comparison matrix ✅

- [x] `docs/comparison.md` — все SDK side-by-side

#### 16.3. Обновление README.md ✅

- [x] Обзор проекта — все 4 языка
- [x] Примеры для каждого языка (краткие)
- [x] Ссылки на quickstart и migration guide для каждого языка
- [ ] Badges: CI/CD status, версии пакетов (когда опубликованы)

#### 16.4. Финальный conformance-прогон

- [x] `conformance/run.sh --lang all` — все 4 SDK последовательно:
  - Go: все 8 сценариев ✓
  - Python: все 8 сценариев ✓
  - Java: все 8 сценариев ✓
  - C#: все 8 сценариев ✓
- [x] Кросс-языковая проверка идентичности метрик:
  - Имена метрик (app_dependency_health, app_dependency_latency_seconds) ✓
  - Метки (dependency, type, host, port) ✓
  - HELP-строки ✓
  - Бакеты histogram ✓
  - Формат Prometheus text format ✓
- [x] Скрипт `conformance/runner/cross_verify.py` — автоматическая кросс-языковая верификация

#### 16.5. Подготовка к публикации

- [x] Go: LICENSE, README.md для pkg.go.dev
- [x] Python: `pyproject.toml` заполнен (authors, classifiers, urls), README.md, `python -m build`
- [x] Java: `pom.xml` заполнен (SCM, developers, licenses, release profile), версия 0.1.0, `mvn package`
- [x] C#: Directory.Build.props + `.csproj` заполнены (PackageId, metadata), `dotnet pack`
- [x] `.gitignore` — артефакты сборки (dist/, target/, bin/, obj/, nupkg)
- [x] Перевод публичной документации на английский (README, descriptions, CHANGELOG)

#### 16.6. Релиз

- [x] `CHANGELOG.md` — описание всех изменений v0.1.0
- [ ] Git tag `v0.1.0` + `sdk-go/v0.1.0`
- [ ] Release notes (GitHub Release)
- [ ] Публикация пакетов:
  - Go modules (git tag достаточен)
  - PyPI: `twine upload`
  - Maven Central: `mvn deploy -P release`
  - NuGet: пропущен

### Артефакты фазы 16

```text
docs/
├── quickstart/
│   ├── go.md          # уже есть
│   ├── python.md
│   ├── java.md
│   └── csharp.md
├── migration/
│   ├── go.md          # уже есть
│   ├── python.md
│   ├── java.md
│   └── csharp.md
├── comparison.md
└── specification.md   # уже есть

README.md              # обновлённый
CHANGELOG.md           # новый
```

### Критерии завершения фазы 16

- Документация покрывает все 4 языка: quickstart, migration, comparison
- `conformance/run.sh --lang all` — все 32 сценария (8 × 4 языка) проходят
- Метрики идентичны между всеми SDK
- README обновлён, CHANGELOG написан
- Пакеты опубликованы (или готовы к публикации)
- Git tag `v1.0.0` создан

---

## Зависимости между фазами

```text
Фаза 1 (Спецификация)
  ├── Фаза 2 (Conformance-тесты)
  └── Фаза 3 (Go Core)
        └── Фаза 4 (Go Checkers)
              └── Фаза 5 (Go Metrics + Scheduler)
                    └── Фаза 6 (Go API + Contrib)
                          └── Фаза 7 (Тестовый сервис)
                                └── Фаза 8 (Conformance-прогон) ← также от Фазы 2
                                      ├── Фаза 9 (Docs + CI/CD)
                                      └── Фаза 10 (Grafana + Алерты)
                                            └── Фаза 11 (Инфраструктура контейнерной разработки)
                                                  ├── Фаза 12 (Python SDK Core)
                                                  │     └── Фаза 13 (Python FastAPI + Conformance)
                                                  ├── Фаза 14 (Java SDK + Spring Boot + Conformance)
                                                  └── Фаза 15 (C# SDK + ASP.NET + Conformance)
                                                              ↓
                                                  Фаза 16 (Документация + Релиз v1.0) ← от 13, 14, 15
```

---

## Соглашения

- **Язык кода**: английский (имена переменных, функций, классов)
- **Язык коммуникации**: русский (комментарии, документация, commit-сообщения)
- **Git workflow**: GitHub Flow + Conventional Commits (см. `GIT-WORKFLOW.md`)
- **Тестирование**: Docker/Kubernetes (см. `AGENTS.md`)
- **Линтинг**: golangci-lint (Go), markdownlint (MD), соответствующие линтеры для других языков
- **Планы**: помечать завершённые фазы `[x]` в этом файле
