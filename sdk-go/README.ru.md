*[English version](README.md)*

# dephealth

Go SDK для мониторинга зависимостей микросервисов через метрики Prometheus.

## Возможности

- Автоматическая проверка здоровья зависимостей (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Экспорт метрик Prometheus: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Поддержка connection pool (предпочтительно) и автономных проверок
- Functional options pattern для конфигурации
- Пакеты contrib/ для популярных драйверов (pgx, go-redis, go-sql-driver)

## Установка

```bash
go get github.com/BigKAA/topologymetrics/sdk-go/dephealth
```

## Быстрый старт

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    // Регистрация фабрик чекеров
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.Postgres("postgres",
            dephealth.FromURL("postgresql://user:pass@localhost:5432/mydb"),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis",
            dephealth.FromURL("redis://localhost:6379"),
            dephealth.Critical(false),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    go func() {
        log.Fatal(http.ListenAndServe(":8080", nil))
    }()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh
}
```

## Конфигурация

| Опция | Значение по умолчанию | Описание |
| --- | --- | --- |
| `WithCheckInterval(d)` | `15s` | Интервал между проверками здоровья |
| `WithTimeout(d)` | `5s` | Таймаут одной проверки |
| `WithRegisterer(r)` | default | Пользовательский Prometheus registerer |
| `WithLogger(l)` | none | `*slog.Logger` для операций SDK |

## Поддерживаемые зависимости

| Тип | Формат URL |
| --- | --- |
| PostgreSQL | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| Redis | `redis://host:6379` |
| RabbitMQ | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka://host1:9092,host2:9092` |
| HTTP | `http://host:8080/health` |
| gRPC | через `dephealth.FromParams(host, port)` |
| TCP | `tcp://host:port` |
| LDAP | `ldap://host:389` или `ldaps://host:636` |

## LDAP-чекер

LDAP-чекер поддерживает четыре метода проверки и несколько режимов TLS:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"

dephealth.LDAP("directory",
    dephealth.FromURL("ldap://ldap.svc:389"),
    dephealth.Critical(true),
    dephealth.WithLDAPCheckMethod("root_dse"),
)

// Простая привязка с учётными данными
dephealth.LDAP("ad",
    dephealth.FromURL("ldaps://ad.corp:636"),
    dephealth.Critical(true),
    dephealth.WithLDAPCheckMethod("simple_bind"),
    dephealth.WithLDAPBindDN("cn=monitor,dc=corp,dc=com"),
    dephealth.WithLDAPBindPassword("secret"),
)

// Поиск с StartTLS
dephealth.LDAP("openldap",
    dephealth.FromURL("ldap://openldap.svc:389"),
    dephealth.Critical(false),
    dephealth.WithLDAPCheckMethod("search"),
    dephealth.WithLDAPBaseDN("dc=example,dc=com"),
    dephealth.WithLDAPSearchFilter("(objectClass=organizationalUnit)"),
    dephealth.WithLDAPSearchScope("one"),
    dephealth.WithLDAPStartTLS(true),
)
```

Методы проверки: `anonymous_bind`, `simple_bind`, `root_dse` (по умолчанию), `search`.

## Детализация здоровья

```go
details := dh.HealthDetails()
for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s latency=%v\n",
        key, *ep.Healthy, ep.Status, ep.Latency)
}
```

## Динамические эндпоинты

Добавление, удаление и замена мониторируемых эндпоинтов в рантайме на
работающем экземпляре `DepHealth`. Полезно для приложений, которые
обнаруживают зависимости динамически (например, элементы хранилища,
зарегистрированные через REST API).

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

// Добавить новый эндпоинт после Start()
err := dh.AddEndpoint("api-backend", dephealth.TypeHTTP, true,
    dephealth.Endpoint{Host: "backend-2.svc", Port: "8080"},
    httpcheck.New(),
)

// Удалить эндпоинт (отменяет горутину, удаляет метрики)
err = dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080")

// Атомарно заменить эндпоинт новым
err = dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    dephealth.Endpoint{Host: "backend-3.svc", Port: "8080"},
    httpcheck.New(),
)
```

Все три метода потокобезопасны и идемпотентны (`AddEndpoint` игнорирует
дубликаты, `RemoveEndpoint` игнорирует отсутствующие эндпоинты).

## Интеграция с connection pool

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"

db, _ := sql.Open("pgx", connString)

dh, _ := dephealth.New("my-service", "my-team",
    sqldb.FromDB("postgres", db,
        dephealth.FromURL("postgresql://localhost:5432/mydb"),
        dephealth.Critical(true),
    ),
)
```

Подробности в [руководстве по connection pools](docs/connection-pools.ru.md)
для Redis и MySQL.

## Селективные импорты

По умолчанию импорт пакета `checks` регистрирует все фабрики чекеров:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks" // все чекеры
```

Для уменьшения размера бинарника импортируйте только нужные:

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Доступные подпакеты: `tcpcheck`, `httpcheck`, `grpccheck`, `pgcheck`,
`mysqlcheck`, `redischeck`, `amqpcheck`, `kafkacheck`, `ldapcheck`.

## Аутентификация

HTTP и gRPC чекеры поддерживают Bearer token, Basic Auth и пользовательские заголовки/метаданные:

```go
dephealth.HTTP("secure-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBearerToken("eyJhbG..."),
)

dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBearerToken("eyJhbG..."),
)
```

Все опции описаны в [руководстве по аутентификации](docs/authentication.ru.md).

## Документация

Полная документация доступна в директории [docs/](docs/getting-started.ru.md).

| Руководство | Описание |
| --- | --- |
| [Начало работы](docs/getting-started.ru.md) | Установка, настройка, первая проверка здоровья |
| [Чекеры](docs/checkers.ru.md) | Подробное руководство по всем встроенным чекерам |
| [Конфигурация](docs/configuration.ru.md) | Все опции, значения по умолчанию, переменные окружения |
| [Connection Pools](docs/connection-pools.ru.md) | Интеграция с существующими connection pool |
| [Пользовательские чекеры](docs/custom-checkers.ru.md) | Реализация собственного health checker |
| [Аутентификация](docs/authentication.ru.md) | Аутентификация для HTTP, gRPC и БД |
| [Метрики](docs/metrics.ru.md) | Справка по Prometheus-метрикам и примеры PromQL |
| [Селективные импорты](docs/selective-imports.ru.md) | Оптимизация размера бинарника с раздельными пакетами |
| [API Reference](docs/api-reference.ru.md) | Полный справочник по публичным символам |
| [Решение проблем](docs/troubleshooting.ru.md) | Частые проблемы и решения |
| [Миграция](docs/migration.ru.md) | Инструкции по обновлению между версиями |
| [Code Style](docs/code-style.ru.md) | Конвенции кода Go для данного проекта |
| [Примеры](docs/examples/) | Полные рабочие примеры |

## Лицензия

Apache License 2.0 — см. [LICENSE](../LICENSE).
