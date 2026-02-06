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

> Фазы 11–16 (Java, C#, Python SDK) планируются после стабилизации Go SDK и спецификации.

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

**Статус**: [ ] В процессе (10.1–10.4 завершены; 10.5 не начата)

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

- [ ] **Метрики доступны в VictoriaMetrics**:
  - Запрос `app_dependency_health` возвращает данные
  - Все 4 зависимости тестового сервиса видны (postgres, redis, http-stub, grpc-stub)
  - Histogram `app_dependency_latency_seconds` содержит бакеты
- [ ] **Дашборды отображают данные**:
  - Overview: таблица заполнена, stat-панели показывают числа
  - Service Detail: timeline и heatmap отображают историю
  - Dependency Map: Node Graph показывает связи (или текстовая панель с инструкцией)
- [ ] **Тестирование алертов — DependencyDown**:
  - Масштабировать Redis в 0 (`kubectl scale deployment redis --replicas=0`)
  - Подождать 1 минуту → алерт `DependencyDown` (critical) в Alertmanager
  - Проверить: дашборд Overview показывает Redis = DOWN (красный)
- [ ] **Тестирование алертов — DependencyDegraded**:
  - Если есть replica PG — масштабировать в 0
  - Подождать 2 минуты → алерт `DependencyDegraded` (warning)
- [ ] **Тестирование алертов — DependencyHighLatency**:
  - Включить задержку в HTTP-заглушке: `/admin/delay?ms=2000`
  - Подождать 5 минут → алерт `DependencyHighLatency` (warning)
- [ ] **Тестирование алертов — DependencyFlapping**:
  - Быстро переключать HTTP-заглушку: `/admin/toggle` каждые 30 секунд × 6 раз
  - Подождать → алерт `DependencyFlapping` (info)
- [ ] **Тестирование подавления каскадов**:
  - Масштабировать Redis в 0 → `DependencyDown` = active
  - Проверить: `DependencyFlapping` и `DependencyHighLatency` для Redis подавлены
  - Проверить inhibition-rules в Alertmanager UI
- [ ] **Восстановление**:
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

## Фазы 11–16: SDK для остальных языков (план верхнего уровня)

> Детальное планирование — после стабилизации Go SDK и спецификации.

### Фаза 11: Java SDK — Core

- Модули: `dephealth-core` (Maven)
- Абстракции: `Dependency`, `Endpoint`, `HealthChecker` (interface)
- Чекеры: HTTP, gRPC, TCP, JDBC, Redis, AMQP, Kafka
- Prometheus exporter (simpleclient)
- Check scheduler (ScheduledExecutorService)
- Unit-тесты и integration-тесты

### Фаза 12: Java SDK — Spring Boot Integration

- Модуль: `dephealth-spring-boot-starter`
- Auto-configuration: `@EnableDependencyHealth`
- Интеграция с Spring Boot Actuator (`/actuator/health/dependencies`)
- Micrometer bridge (`dephealth-micrometer`)
- Conformance-прогон

### Фаза 13: C# SDK

- NuGet-пакеты: `DepHealth.Core`, `DepHealth.AspNetCore`, `DepHealth.EntityFramework`
- Аналогичная структура: абстракции → чекеры → метрики → scheduler
- ASP.NET middleware, IHealthCheck
- prometheus-net интеграция
- Conformance-прогон

### Фаза 14: Python SDK

- PyPI-пакеты: `dephealth`, `dephealth-fastapi`, `dephealth-django`
- Двойная поддержка: async (asyncio) и sync (threading)
- prometheus_client интеграция
- FastAPI lifespan + middleware
- Django app + management command
- Conformance-прогон

### Фаза 15: Кросс-языковая документация

- Quickstart для каждого языка
- Migration guide для каждого языка
- Единый README с примерами для всех языков
- Comparison matrix: возможности каждого SDK

### Фаза 16: Релиз v1.0

- Финальный conformance-прогон для всех SDK
- Публикация пакетов (Go modules, Maven Central, NuGet, PyPI)
- Создание тега `v1.0.0`
- Release notes
- Анонс

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
                                └── Фаза 8 (Conformance-прогон) ← также зависит от Фазы 2
                                      ├── Фаза 9 (Docs + CI/CD)
                                      ├── Фаза 10 (Grafana + Алерты)
                                      └── Фазы 11–16 (Остальные SDK)
```

---

## Соглашения

- **Язык кода**: английский (имена переменных, функций, классов)
- **Язык коммуникации**: русский (комментарии, документация, commit-сообщения)
- **Git workflow**: GitHub Flow + Conventional Commits (см. `GIT-WORKFLOW.md`)
- **Тестирование**: Docker/Kubernetes (см. `AGENTS.md`)
- **Линтинг**: golangci-lint (Go), markdownlint (MD), соответствующие линтеры для других языков
- **Планы**: помечать завершённые фазы `[x]` в этом файле
