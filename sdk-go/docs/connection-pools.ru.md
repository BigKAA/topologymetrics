*[English version](connection-pools.md)*

# Интеграция с пулами соединений

dephealth поддерживает два режима проверки зависимостей:

- **Автономный режим** — SDK создаёт новое соединение для каждой проверки
- **Режим пула** — SDK использует существующий пул соединений вашего сервиса

Режим пула предпочтительнее, так как он отражает реальную способность
сервиса работать с зависимостью. Если пул соединений исчерпан, проверка
это обнаружит.

## Автономный режим vs Режим пула

| Аспект | Автономный | Пул |
| --- | --- | --- |
| Соединение | Новое для каждой проверки | Переиспользует существующий пул |
| Отражает реальное состояние | Частично | Да |
| Настройка | Просто — только URL | Требует передачи объекта пула/клиента |
| Внешние зависимости | Нет (использует драйвер чекера) | Драйвер вашего приложения |
| Обнаруживает исчерпание пула | Нет | Да |

## Пакеты contrib

Директория `contrib/` предоставляет хелпер-функции для популярных
драйверов. Эти функции оборачивают объект пула/клиента и возвращают
`dephealth.Option`, который можно передать напрямую в `dephealth.New()`.

### PostgreSQL через `*sql.DB`

Пакет: `contrib/sqldb`

```go
import (
    "database/sql"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// Создаём пул соединений как обычно
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

dh, err := dephealth.New("my-service", "my-team",
    sqldb.FromDB("postgres-main", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

`sqldb.FromDB()` создаёт PostgreSQL-чекер, использующий предоставленный
`*sql.DB` для проверок. Необходимо указать `FromURL()` или
`FromParams()`, чтобы SDK знал host и port для меток метрик.

### MySQL через `*sql.DB`

Пакет: `contrib/sqldb`

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")
if err != nil {
    log.Fatal(err)
}

dh, err := dephealth.New("my-service", "my-team",
    sqldb.FromMySQLDB("mysql-main", db,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis через `*redis.Client`

Пакет: `contrib/redispool`

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr:     "redis.svc:6379",
    Password: "secret",
    DB:       0,
})

dh, err := dephealth.New("my-service", "my-team",
    redispool.FromClient("redis-cache", client,
        dephealth.Critical(false),
    ),
)
```

`redispool.FromClient()` автоматически извлекает host и port из
`client.Options().Addr`, поэтому `FromURL()` или `FromParams()` не
обязательны (но могут быть указаны для переопределения).

## Прямой режим пула через чекер

Если нужен больший контроль, можно создать чекеры напрямую с опциями
пула и зарегистрировать их через `AddDependency()`:

### PostgreSQL

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)

db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

checker := pgcheck.New(
    pgcheck.WithDB(db),
    pgcheck.WithQuery("SELECT 1"),
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("postgres-main", dephealth.TypePostgres, checker,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

### MySQL

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
)

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

checker := mysqlcheck.New(
    mysqlcheck.WithDB(db),
    mysqlcheck.WithQuery("SELECT 1"),
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("mysql-main", dephealth.TypeMySQL, checker,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr: "redis.svc:6379",
})

checker := redischeck.New(redischeck.WithClient(client))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("redis-cache", dephealth.TypeRedis, checker,
        dephealth.FromParams("redis.svc", "6379"),
        dephealth.Critical(false),
    ),
)
```

## contrib vs Прямой: когда что использовать

| Сценарий | Рекомендация |
| --- | --- |
| Стандартная настройка, один пул на зависимость | `contrib/sqldb` или `contrib/redispool` |
| Пользовательский запрос проверки | Прямой чекер с `pgcheck.WithQuery()` |
| Несколько БД на одном пуле `*sql.DB` | Прямой чекер (указать запрос для каждой) |
| Нестандартный драйвер или обёртка | Прямой чекер с объектом пула |

## Полный пример: смешанные режимы

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/redis/go-redis/v9"
    _ "github.com/jackc/pgx/v5/stdlib"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    // Существующие пулы соединений
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    rdb := redis.NewClient(&redis.Options{Addr: "redis.svc:6379"})

    dh, err := dephealth.New("my-service", "my-team",
        // Режим пула — PostgreSQL
        sqldb.FromDB("postgres-main", db,
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Режим пула — Redis
        redispool.FromClient("redis-cache", rdb,
            dephealth.Critical(false),
        ),

        // Автономный режим — HTTP (пул не нужен)
        dephealth.HTTP("payment-api",
            dephealth.FromURL("http://payment.svc:8080"),
            dephealth.Critical(true),
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

- [Чекеры](checkers.ru.md) — все детали чекеров, включая разделы о режиме пула
- [Кастомные чекеры](custom-checkers.ru.md) — создание собственного HealthChecker
- [API Reference](api-reference.ru.md) — `AddDependency`, функции contrib
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
