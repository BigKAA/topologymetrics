*[English version](selective-imports.md)*

# Выборочный импорт

Начиная с v0.6.0, Go SDK поддерживает выборочный импорт — вы можете
импортировать только те чекеры, которые реально использует ваш сервис,
уменьшая размер бинарника и количество транзитивных зависимостей.

## Проблема

В v0.5.0 и ранее пакет `checks` был монолитным. Его импорт подтягивал
все 8 реализаций чекеров и их зависимости:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

Это означало, что даже если вашему сервису нужны только HTTP и PostgreSQL
проверки, скомпилированный бинарник включал библиотеки gRPC, Kafka, AMQP
и другие.

## Решение

В v0.6.0 каждый чекер живёт в собственном подпакете внутри `checks/`.
Импортируйте только нужное:

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Остальной API остаётся прежним — вы по-прежнему вызываете `dephealth.HTTP()`,
`dephealth.Postgres()` и т.д.

## Доступные подпакеты

| Подпакет | Путь импорта | Внешние зависимости |
| --- | --- | --- |
| `httpcheck` | `.../checks/httpcheck` | только stdlib |
| `tcpcheck` | `.../checks/tcpcheck` | только stdlib |
| `grpccheck` | `.../checks/grpccheck` | `google.golang.org/grpc` |
| `pgcheck` | `.../checks/pgcheck` | `github.com/jackc/pgx/v5` |
| `mysqlcheck` | `.../checks/mysqlcheck` | `github.com/go-sql-driver/mysql` |
| `redischeck` | `.../checks/redischeck` | `github.com/redis/go-redis/v9` |
| `amqpcheck` | `.../checks/amqpcheck` | `github.com/rabbitmq/amqp091-go` |
| `kafkacheck` | `.../checks/kafkacheck` | `github.com/segmentio/kafka-go` |

Полные пути импорта используют префикс модуля
`github.com/BigKAA/topologymetrics/sdk-go/dephealth/`.

## Как это работает

Каждый подпакет регистрирует свою фабрику чекеров через `init()`:

```go
// Внутри checks/httpcheck/httpcheck.go
func init() {
    dephealth.RegisterCheckerFactory(dephealth.TypeHTTP, NewFromConfig)
}
```

Когда вы делаете blank-import подпакета (`import _ ".../checks/httpcheck"`),
выполняется его `init()` и регистрирует фабрику. После этого
`dephealth.HTTP()` может создавать экземпляры HTTP-чекера.

Если вызвать `dephealth.HTTP()` без импорта `httpcheck` (или `checks`),
`New()` вернёт ошибку: `no checker factory registered for type "http"`.

## Полный импорт (все чекеры)

Пакет `checks` по-прежнему работает как раньше. Он делает blank-import
всех 8 подпакетов:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

Это эквивалентно импорту всех 8 подпакетов по отдельности. Используйте
этот способ, когда размер бинарника не критичен или когда нужны все типы
чекеров.

## Пример: выборочный импорт

Сервис, который мониторит только PostgreSQL и Redis:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    // Регистрируем только нужные чекеры
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.Postgres("postgres-main",
            dephealth.FromURL("postgresql://localhost:5432/mydb"),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis-cache",
            dephealth.FromParams("localhost", "6379"),
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

Этот бинарник **не будет** включать библиотеки gRPC, Kafka, AMQP,
HTTP-чекера, MySQL и TCP-чекера.

## Обратная совместимость

Пакет `checks` предоставляет устаревшие (deprecated) псевдонимы типов
и обёртки конструкторов для всех чекеров. Существующий код, использующий
`checks.HTTPChecker`, `checks.NewHTTPChecker()` и т.д., продолжает
компилироваться без изменений:

```go
// Старый стиль (работает, но устарел)
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"

checker := checks.NewHTTPChecker(checks.WithHealthPath("/ready"))
```

```go
// Новый стиль (рекомендуется)
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

checker := httpcheck.New(httpcheck.WithHealthPath("/ready"))
```

Все устаревшие псевдонимы перечислены в `checks/compat.go` со ссылками
в godoc на новые пакеты.

## Contrib-пакеты

Пакеты `contrib/` (`sqldb`, `redispool`) работают с обоими стилями
импорта. Они импортируют соответствующие подпакеты чекеров внутри себя,
поэтому дополнительный blank-import при использовании contrib не нужен:

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    // Не нужно: _ ".../checks/pgcheck" — sqldb импортирует его сам
)
```

## Миграция с v0.5.0

Если вы обновляетесь с v0.5.0, смотрите
[Руководство по миграции](../../docs/migration/v050-to-v060.ru.md)
с пошаговыми инструкциями.

## См. также

- [Начало работы](getting-started.ru.md) — базовая настройка и первый пример
- [Чекеры](checkers.ru.md) — подробное руководство по всем 8 чекерам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
- [Руководство по миграции v0.5.0 → v0.6.0](../../docs/migration/v050-to-v060.ru.md)
