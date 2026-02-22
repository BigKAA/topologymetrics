*[English version](checkers.md)*

# Чекеры

Go SDK включает 8 встроенных чекеров для распространённых типов зависимостей.
Каждый чекер реализует интерфейс `HealthChecker` и может использоваться
через высокоуровневый API (`dephealth.HTTP()` и т.д.) или напрямую через
свой подпакет.

## HTTP

Проверяет HTTP-эндпоинты, отправляя GET-запрос и ожидая ответ с кодом 2xx.

### Регистрация

```go
dephealth.HTTP("payment-api",
    dephealth.FromURL("http://payment.svc:8080"),
    dephealth.Critical(true),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithHTTPHealthPath(path)` | `/health` | Путь для эндпоинта проверки |
| `WithHTTPTLS(enabled)` | `false` | Использовать HTTPS вместо HTTP |
| `WithHTTPTLSSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `WithHTTPHeaders(headers)` | — | Пользовательские HTTP-заголовки (map[string]string) |
| `WithHTTPBearerToken(token)` | — | Установить заголовок `Authorization: Bearer <token>` |
| `WithHTTPBasicAuth(user, pass)` | — | Установить заголовок `Authorization: Basic <base64>` |

### Полный пример

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.HTTP("payment-api",
            dephealth.FromURL("https://payment.svc:443"),
            dephealth.Critical(true),
            dephealth.WithHTTPHealthPath("/healthz"),
            dephealth.WithHTTPTLS(true),
            dephealth.WithHTTPTLSSkipVerify(true),
            dephealth.WithHTTPHeaders(map[string]string{
                "X-Request-Source": "dephealth",
            }),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Ответ 2xx | `ok` | `ok` |
| Ответ 401 или 403 | `auth_error` | `auth_error` |
| Другой не-2xx ответ | `unhealthy` | `http_<код>` (напр., `http_500`) |
| Сетевая ошибка | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

checker := httpcheck.New(
    httpcheck.WithHealthPath("/ready"),
    httpcheck.WithTLSEnabled(true),
)

err := checker.Check(ctx, dephealth.Endpoint{Host: "api.svc", Port: "8080"})
```

### Особенности поведения

- Автоматически следует HTTP-редиректам (3xx)
- Создаёт новый HTTP-клиент для каждой проверки
- Отправляет заголовок `User-Agent: dephealth/0.6.0`
- Пользовательские заголовки применяются после User-Agent и могут его перезаписать

---

## gRPC

Проверяет gRPC-сервисы через
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Регистрация

```go
dephealth.GRPC("user-service",
    dephealth.FromParams("user.svc", "9090"),
    dephealth.Critical(true),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithGRPCServiceName(name)` | `""` (пусто) | Имя сервиса; пустое — проверка всего сервера |
| `WithGRPCTLS(enabled)` | `false` | Включить TLS |
| `WithGRPCTLSSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `WithGRPCMetadata(md)` | — | Пользовательские метаданные gRPC (map[string]string) |
| `WithGRPCBearerToken(token)` | — | Установить метаданные `authorization: Bearer <token>` |
| `WithGRPCBasicAuth(user, pass)` | — | Установить метаданные `authorization: Basic <base64>` |

### Полный пример

```go
dh, err := dephealth.New("my-service", "my-team",
    // Проверка конкретного gRPC-сервиса
    dephealth.GRPC("user-service",
        dephealth.FromParams("user.svc", "9090"),
        dephealth.Critical(true),
        dephealth.WithGRPCServiceName("user.v1.UserService"),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCMetadata(map[string]string{
            "x-request-id": "dephealth",
        }),
    ),

    // Проверка состояния всего сервера (пустое имя сервиса)
    dephealth.GRPC("grpc-gateway",
        dephealth.FromParams("gateway.svc", "9090"),
        dephealth.Critical(false),
    ),
)
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Ответ SERVING | `ok` | `ok` |
| gRPC UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC PERMISSION_DENIED | `auth_error` | `auth_error` |
| Ответ NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Другой gRPC-статус | `unhealthy` | `grpc_unknown` |
| Ошибка соединения/RPC | классифицируется ядром | зависит от типа ошибки |

### Прямое использование чекера

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"

checker := grpccheck.New(
    grpccheck.WithServiceName("user.v1.UserService"),
    grpccheck.WithTLS(true),
)

err := checker.Check(ctx, dephealth.Endpoint{Host: "user.svc", Port: "9090"})
```

### Особенности поведения

- Использует `passthrough:///` resolver для обхода DNS SRV-запросов
  (критично в Kubernetes, где `ndots:5` вызывает высокую задержку при
  использовании `dns:///` resolver)
- Создаёт новое gRPC-соединение для каждой проверки
- Пустое имя сервиса проверяет состояние всего сервера

---

## TCP

Проверяет TCP-подключение: устанавливает соединение и немедленно закрывает.
Простейший чекер — без протокола прикладного уровня.

### Регистрация

```go
dephealth.TCP("memcached",
    dephealth.FromParams("memcached.svc", "11211"),
    dephealth.Critical(false),
)
```

### Опции

Нет специфичных опций. TCP-чекер не имеет состояния.

### Полный пример

```go
dh, err := dephealth.New("my-service", "my-team",
    dephealth.TCP("memcached",
        dephealth.FromParams("memcached.svc", "11211"),
        dephealth.Critical(false),
        dephealth.CheckInterval(10 * time.Second),
    ),
    dephealth.TCP("custom-service",
        dephealth.FromParams("custom.svc", "5555"),
        dephealth.Critical(true),
    ),
)
```

### Классификация ошибок

TCP-чекер не производит классифицированных ошибок. Все ошибки
(отказ соединения, DNS-ошибки, таймауты) классифицируются ядром.

### Прямое использование чекера

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck"

checker := tcpcheck.New()
err := checker.Check(ctx, dephealth.Endpoint{Host: "memcached.svc", Port: "11211"})
```

### Особенности поведения

- Выполняет только TCP-рукопожатие (SYN/ACK) — данные не отправляются
  и не принимаются
- Соединение закрывается сразу после установки
- Подходит для сервисов без протокола проверки здоровья

---

## PostgreSQL

Проверяет PostgreSQL, выполняя запрос (по умолчанию `SELECT 1`).
Поддерживает автономный режим (новое соединение) и режим пула
(существующий `*sql.DB`).

### Регистрация

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL("postgresql://user:pass@pg.svc:5432/mydb"),
    dephealth.Critical(true),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithPostgresQuery(query)` | `SELECT 1` | Пользовательский SQL-запрос для проверки |

Для режима пула используйте пакет `contrib/sqldb` или создайте чекер
напрямую с `pgcheck.WithDB(db)`.

### Полный пример

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Автономный режим — создаёт новое соединение для каждой проверки
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
        dephealth.WithPostgresQuery("SELECT 1"),
    ),
)
```

### Режим пула

Использование существующего пула соединений:

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

checker := pgcheck.New(pgcheck.WithDB(db))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("postgres-main", dephealth.TypePostgres, checker,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

Или используйте хелпер `contrib/sqldb` — см. [Пулы соединений](connection-pools.ru.md).

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| SQLSTATE 28000 (неверная авторизация) | `auth_error` | `auth_error` |
| SQLSTATE 28P01 (ошибка аутентификации) | `auth_error` | `auth_error` |
| "password authentication failed" в ошибке | `auth_error` | `auth_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Особенности поведения

- В автономном режиме без указания URL строит DSN
  `postgres://host:port/postgres` (подключение к БД `postgres` по умолчанию)
- Режим пула переиспользует существующий пул — отражает реальную
  способность сервиса работать с зависимостью
- Использует драйвер `pgx` (`github.com/jackc/pgx/v5/stdlib`)

---

## MySQL

Проверяет MySQL, выполняя запрос (по умолчанию `SELECT 1`). Поддерживает
автономный режим и режим пула.

### Регистрация

```go
dephealth.MySQL("mysql-main",
    dephealth.FromURL("mysql://user:pass@mysql.svc:3306/mydb"),
    dephealth.Critical(true),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithMySQLQuery(query)` | `SELECT 1` | Пользовательский SQL-запрос для проверки |

Для режима пула используйте пакет `contrib/sqldb` или создайте чекер
напрямую с `mysqlcheck.WithDB(db)`.

### Полный пример

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.MySQL("mysql-main",
        dephealth.FromURL("mysql://user:pass@mysql.svc:3306/mydb"),
        dephealth.Critical(true),
    ),
)
```

### Режим пула

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
    _ "github.com/go-sql-driver/mysql"
)

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

checker := mysqlcheck.New(mysqlcheck.WithDB(db))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("mysql-main", dephealth.TypeMySQL, checker,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Конвертация URL в DSN

Пакет `mysqlcheck` предоставляет функцию `URLToDSN()` для конвертации
URL в формат DSN драйвера go-sql-driver:

```go
dsn := mysqlcheck.URLToDSN("mysql://user:pass@host:3306/db?charset=utf8")
// Результат: "user:pass@tcp(host:3306)/db?charset=utf8"
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Запрос успешен | `ok` | `ok` |
| MySQL ошибка 1045 (Access Denied) | `auth_error` | `auth_error` |
| "Access denied" в сообщении ошибки | `auth_error` | `auth_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Особенности поведения

- В автономном режиме без URL строит DSN `tcp(host:port)/`
- Использует драйвер `go-sql-driver/mysql`
- Парсинг URL сохраняет query-параметры

---

## Redis

Проверяет Redis, выполняя команду `PING`. Поддерживает автономный режим
и режим пула.

### Регистрация

```go
dephealth.Redis("redis-cache",
    dephealth.FromURL("redis://:password@redis.svc:6379/0"),
    dephealth.Critical(false),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithRedisPassword(password)` | `""` | Пароль для автономного режима |
| `WithRedisDB(db)` | `0` | Номер базы данных для автономного режима |

Для режима пула используйте пакет `contrib/redispool` или создайте чекер
напрямую с `redischeck.WithClient(client)`.

### Полный пример

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Пароль из URL
    dephealth.Redis("redis-cache",
        dephealth.FromURL("redis://:mypassword@redis.svc:6379/0"),
        dephealth.Critical(false),
    ),

    // Пароль через опцию
    dephealth.Redis("redis-sessions",
        dephealth.FromParams("redis-sessions.svc", "6379"),
        dephealth.WithRedisPassword("secret"),
        dephealth.WithRedisDB(1),
        dephealth.Critical(true),
    ),
)
```

### Режим пула

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr:     "redis.svc:6379",
    Password: "secret",
    DB:       0,
})

checker := redischeck.New(redischeck.WithClient(client))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("redis-cache", dephealth.TypeRedis, checker,
        dephealth.FromParams("redis.svc", "6379"),
        dephealth.Critical(false),
    ),
)
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| PING успешен | `ok` | `ok` |
| "NOAUTH" в ошибке | `auth_error` | `auth_error` |
| "WRONGPASS" в ошибке | `auth_error` | `auth_error` |
| Отказ соединения | `connection_error` | `connection_refused` |
| Таймаут соединения | `connection_error` | `connection_refused` |
| Context deadline exceeded | `connection_error` | `connection_refused` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Особенности поведения

- В автономном режиме устанавливает фиксированные внутренние таймауты:
  `DialTimeout=3s`, `ReadTimeout=3s`, `WriteTimeout=3s`
- `MaxRetries=0` — без автоматических повторов (повтор выполняет планировщик)
- Пароль из опций имеет приоритет над паролем из URL
- Номер БД из опций имеет приоритет над номером БД из URL

---

## AMQP (RabbitMQ)

Проверяет AMQP-брокеры, устанавливая соединение и немедленно закрывая.
Поддерживается только автономный режим.

### Регистрация

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),
    dephealth.Critical(false),
)
```

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithAMQPURL(url)` | — | Пользовательский AMQP URL (переопределяет `amqp://guest:guest@host:port/`) |

### Полный пример

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Учётные данные по умолчанию (guest:guest)
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.Critical(false),
    ),

    // Пользовательские учётные данные через URL
    dephealth.AMQP("rabbitmq-prod",
        dephealth.FromParams("rmq-prod.svc", "5672"),
        dephealth.WithAMQPURL("amqp://myuser:mypass@rmq-prod.svc:5672/myvhost"),
        dephealth.Critical(true),
    ),
)
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Соединение установлено | `ok` | `ok` |
| "403" в ошибке | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" в ошибке | `auth_error` | `auth_error` |
| Другие ошибки | классифицируются ядром | зависит от типа ошибки |

### Особенности поведения

- URL по умолчанию: `amqp://guest:guest@host:port/` (стандартные учётные
  данные RabbitMQ)
- Нет режима пула — всегда создаёт новое соединение
- Использует обёртку с горутиной для поддержки отмены через context
  (библиотека amqp091-go не поддерживает context нативно)
- Соединение закрывается сразу после успешного установления

---

## Kafka

Проверяет Kafka-брокеры, подключаясь и запрашивая метаданные кластера.
Чекер без состояния, без опций конфигурации.

### Регистрация

```go
dephealth.Kafka("kafka",
    dephealth.FromParams("kafka.svc", "9092"),
    dephealth.Critical(true),
)
```

Несколько хостов (множество брокеров):

```go
dephealth.Kafka("kafka-cluster",
    dephealth.FromURL("kafka://broker1:9092,broker2:9092,broker3:9092"),
    dephealth.Critical(true),
)
```

> Примечание: при использовании `FromURL` каждый брокер создаёт отдельный
> эндпоинт — каждый проверяется независимо и отображается отдельной строкой
> в метриках.

### Опции

Нет специфичных опций. Kafka-чекер не имеет состояния.

### Полный пример

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck"
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.Kafka("kafka",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(true),
    ),
)
```

### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Метаданные содержат брокеры | `ok` | `ok` |
| Нет брокеров в метаданных | `unhealthy` | `no_brokers` |
| Ошибка соединения/метаданных | классифицируется ядром | зависит от типа ошибки |

### Особенности поведения

- Подключается к брокеру, запрашивает метаданные, закрывает соединение
- Проверяет наличие хотя бы одного брокера в ответе
- Использует библиотеку `kafka-go` (`github.com/segmentio/kafka-go`)
- Нет поддержки аутентификации (только plain TCP)

---

## Сводка классификации ошибок

Все чекеры классифицируют ошибки по категориям статусов. Классификатор
ядра обрабатывает общие типы ошибок (таймауты, DNS-ошибки, TLS-ошибки,
отказ соединения). Чекер-специфичная классификация добавляет детали
на уровне протокола:

| Категория статуса | Значение | Типичные причины |
| --- | --- | --- |
| `ok` | Зависимость здорова | Проверка успешна |
| `timeout` | Таймаут проверки | Медленная сеть, перегруженный сервис |
| `connection_error` | Не удаётся подключиться | Сервис не работает, неверный host/port, firewall |
| `dns_error` | Ошибка DNS-разрешения | Неверный hostname, DNS-авария |
| `auth_error` | Ошибка аутентификации | Неверные учётные данные, истёкший токен |
| `tls_error` | Ошибка TLS-рукопожатия | Невалидный сертификат, ошибка конфигурации TLS |
| `unhealthy` | Подключён, но нездоров | Сервис сообщает о проблеме, возвращает код ошибки |
| `error` | Неожиданная ошибка | Неклассифицированные сбои |

## См. также

- [Начало работы](getting-started.ru.md) — базовая настройка и первый пример
- [Аутентификация](authentication.ru.md) — подробное руководство по авторизации для HTTP и gRPC
- [Пулы соединений](connection-pools.ru.md) — режим пула через contrib-пакеты
- [Кастомные чекеры](custom-checkers.ru.md) — создание собственного `HealthChecker`
- [Выборочный импорт](selective-imports.ru.md) — импорт только нужных чекеров
- [API Reference](api-reference.ru.md) — полный справочник по всем публичным символам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
