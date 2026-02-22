*[English version](authentication.md)*

# Аутентификация

Руководство по настройке аутентификации для всех чекеров, поддерживающих
учётные данные — HTTP, gRPC и чекеры баз данных/кэшей.

## Обзор

| Чекер | Методы авторизации | Как передаются учётные данные |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, пользовательские заголовки | `WithHTTPBearerToken`, `WithHTTPBasicAuth`, `WithHTTPHeaders` |
| gRPC | Bearer token, Basic auth, пользовательские метаданные | `WithGRPCBearerToken`, `WithGRPCBasicAuth`, `WithGRPCMetadata` |
| PostgreSQL | Логин/пароль | Учётные данные в URL (`postgresql://user:pass@...`) |
| MySQL | Логин/пароль | Учётные данные в URL (`mysql://user:pass@...`) |
| Redis | Пароль | `WithRedisPassword` или пароль в URL (`redis://:pass@...`) |
| AMQP | Логин/пароль | Учётные данные в URL через `WithAMQPURL` (`amqp://user:pass@...`) |
| TCP | — | Без аутентификации |
| Kafka | — | Без аутентификации |

## HTTP-аутентификация

Допускается только один метод аутентификации на зависимость.
Одновременное указание `WithHTTPBearerToken` и `WithHTTPBasicAuth`
вызывает ошибку валидации.

### Bearer Token

```go
dephealth.HTTP("secure-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBearerToken(os.Getenv("API_TOKEN")),
)
```

Отправляет заголовок `Authorization: Bearer <token>` с каждым запросом
проверки.

### Basic Auth

```go
dephealth.HTTP("basic-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBasicAuth("admin", os.Getenv("API_PASSWORD")),
)
```

Отправляет заголовок `Authorization: Basic <base64(user:pass)>`.

### Пользовательские заголовки

Для нестандартных схем аутентификации (API-ключи, пользовательские токены):

```go
dephealth.HTTP("custom-auth-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPHeaders(map[string]string{
        "X-API-Key":    os.Getenv("API_KEY"),
        "X-API-Secret": os.Getenv("API_SECRET"),
    }),
)
```

Пользовательские заголовки применяются после заголовка `User-Agent`
и могут его перезаписать при необходимости.

## gRPC-аутентификация

Допускается только один метод аутентификации на зависимость, как и для HTTP.

### Bearer Token

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBearerToken(os.Getenv("GRPC_TOKEN")),
)
```

Отправляет `authorization: Bearer <token>` как gRPC-метаданные.

### Basic Auth

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBasicAuth("admin", os.Getenv("GRPC_PASSWORD")),
)
```

Отправляет `authorization: Basic <base64(user:pass)>` как gRPC-метаданные.

### Пользовательские метаданные

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCMetadata(map[string]string{
        "x-api-key": os.Getenv("GRPC_API_KEY"),
    }),
)
```

## Учётные данные баз данных

### PostgreSQL

Учётные данные включены в URL подключения:

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL("postgresql://myuser:mypass@pg.svc:5432/mydb"),
    dephealth.Critical(true),
)
```

На практике используйте переменные окружения:

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL(os.Getenv("DATABASE_URL")),
    dephealth.Critical(true),
)
```

### MySQL

Аналогичный подход — учётные данные в URL:

```go
dephealth.MySQL("mysql-main",
    dephealth.FromURL("mysql://myuser:mypass@mysql.svc:3306/mydb"),
    dephealth.Critical(true),
)
```

### Redis

Пароль можно задать через опцию или включить в URL:

```go
// Через опцию
dephealth.Redis("redis-cache",
    dephealth.FromParams("redis.svc", "6379"),
    dephealth.WithRedisPassword(os.Getenv("REDIS_PASSWORD")),
    dephealth.Critical(false),
)

// Через URL
dephealth.Redis("redis-cache",
    dephealth.FromURL("redis://:mypassword@redis.svc:6379/0"),
    dephealth.Critical(false),
)
```

При указании обоих вариантов значение из опции имеет приоритет над URL.

### AMQP (RabbitMQ)

Учётные данные являются частью AMQP URL:

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),
    dephealth.WithAMQPURL(os.Getenv("AMQP_URL")),
    dephealth.Critical(false),
)
```

URL по умолчанию (без `WithAMQPURL`): `amqp://guest:guest@host:port/`

## Классификация ошибок аутентификации

Когда зависимость отклоняет учётные данные, чекер классифицирует ошибку
как `auth_error`. Это позволяет настроить специфичные алерты на ошибки
аутентификации, не смешивая их с проблемами подключения или здоровья.

| Чекер | Триггер | Статус | Детализация |
| --- | --- | --- | --- |
| HTTP | Ответ 401 (Unauthorized) | `auth_error` | `auth_error` |
| HTTP | Ответ 403 (Forbidden) | `auth_error` | `auth_error` |
| gRPC | Код UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC | Код PERMISSION_DENIED | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28000 | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28P01 | `auth_error` | `auth_error` |
| MySQL | Ошибка 1045 (Access Denied) | `auth_error` | `auth_error` |
| Redis | Ошибка NOAUTH | `auth_error` | `auth_error` |
| Redis | Ошибка WRONGPASS | `auth_error` | `auth_error` |
| AMQP | 403 ACCESS_REFUSED | `auth_error` | `auth_error` |

### Пример PromQL: алертинг на ошибки аутентификации

```promql
# Алерт при auth_error статусе любой зависимости
app_dependency_status{status="auth_error"} == 1
```

## Лучшие практики безопасности

1. **Никогда не хардкодьте учётные данные** — всегда используйте
   переменные окружения или системы управления секретами
2. **Используйте короткоживущие токены** — если ваша система авторизации
   поддерживает ротацию токенов, чекер использует значение токена,
   переданное при создании
3. **Предпочитайте TLS** — включайте TLS для HTTP и gRPC чекеров
   при использовании аутентификации для защиты учётных данных при передаче
4. **Ограничивайте права** — используйте read-only учётные данные
   для проверок; `SELECT 1` не требует прав на запись

## Полный пример

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        // HTTP с Bearer-токеном
        dephealth.HTTP("payment-api",
            dephealth.FromURL("https://payment.svc:443"),
            dephealth.WithHTTPTLS(true),
            dephealth.WithHTTPBearerToken(os.Getenv("PAYMENT_TOKEN")),
            dephealth.Critical(true),
        ),

        // gRPC с пользовательскими метаданными
        dephealth.GRPC("user-service",
            dephealth.FromParams("user.svc", "9090"),
            dephealth.WithGRPCMetadata(map[string]string{
                "x-api-key": os.Getenv("USER_API_KEY"),
            }),
            dephealth.Critical(true),
        ),

        // PostgreSQL с учётными данными в URL
        dephealth.Postgres("postgres-main",
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Redis с паролем
        dephealth.Redis("redis-cache",
            dephealth.FromParams("redis.svc", "6379"),
            dephealth.WithRedisPassword(os.Getenv("REDIS_PASSWORD")),
            dephealth.Critical(false),
        ),

        // AMQP с учётными данными в URL
        dephealth.AMQP("rabbitmq",
            dephealth.FromParams("rabbitmq.svc", "5672"),
            dephealth.WithAMQPURL(os.Getenv("AMQP_URL")),
            dephealth.Critical(false),
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

## См. также

- [Чекеры](checkers.ru.md) — подробное руководство по всем 8 чекерам
- [Конфигурация](configuration.ru.md) — переменные окружения и опции
- [Метрики](metrics.ru.md) — метрики Prometheus, включая категории статусов
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
