*[English version](getting-started.md)*

# Начало работы

Руководство по установке, базовой настройке и первой проверке состояния
зависимости с помощью Go SDK dephealth.

## Требования

- Go 1.21 или новее
- Работающая зависимость для мониторинга (PostgreSQL, Redis, HTTP-сервис и т.д.)

## Установка

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

Путь модуля: `github.com/BigKAA/topologymetrics/sdk-go/dephealth`.

## Регистрация чекеров

Перед созданием экземпляра `DepHealth` необходимо зарегистрировать фабрики
чекеров для типов зависимостей, которые вы планируете мониторить.
Есть два способа:

**Импортировать все чекеры сразу:**

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

**Импортировать только нужные (уменьшает размер бинарника):**

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Подробнее — в разделе [Выборочный импорт](selective-imports.ru.md).

## Минимальный пример

Мониторинг одной HTTP-зависимости с экспортом метрик Prometheus:

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

    // Регистрация фабрики HTTP-чекера
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.HTTP("payment-api",
            dephealth.FromURL("http://payment.svc:8080"),
            dephealth.Critical(true),
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

После запуска метрики Prometheus доступны по адресу `http://localhost:8080/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Ключевые концепции

### Имя и группа

Каждый экземпляр `DepHealth` требует два идентификатора:

- **name** — уникальное имя приложения (например, `"my-service"`)
- **group** — логическая группа сервиса (например, `"my-team"`, `"payments"`)

Оба значения появляются как метки во всех экспортируемых метриках.
Правила валидации: `[a-z][a-z0-9-]*`, от 1 до 63 символов.

Если не переданы как аргументы, SDK использует переменные окружения
`DEPHEALTH_NAME` и `DEPHEALTH_GROUP` как запасной вариант.

### Зависимости

Каждая зависимость регистрируется через функцию-фабрику, соответствующую
её типу:

| Функция | Тип зависимости |
| --- | --- |
| `dephealth.HTTP()` | HTTP-сервис |
| `dephealth.GRPC()` | gRPC-сервис |
| `dephealth.TCP()` | TCP-эндпоинт |
| `dephealth.Postgres()` | База данных PostgreSQL |
| `dephealth.MySQL()` | База данных MySQL |
| `dephealth.Redis()` | Сервер Redis |
| `dephealth.AMQP()` | RabbitMQ (AMQP-брокер) |
| `dephealth.Kafka()` | Брокер Apache Kafka |

Для каждой зависимости обязательны:

- **Имя** (первый аргумент) — идентификатор зависимости в метриках
- **Эндпоинт** — через `FromURL()` или `FromParams()`
- **Флаг критичности** — `Critical(true)` или `Critical(false)` (обязателен)

### Жизненный цикл

1. **Создание** — `dephealth.New(name, group, opts...)`
2. **Запуск** — `dh.Start(ctx)` запускает периодические проверки
3. **Работа** — проверки выполняются с заданным интервалом (по умолчанию 15 сек)
4. **Остановка** — `dh.Stop()` останавливает проверки и ожидает завершения горутин

## Несколько зависимостей

```go
dh, err := dephealth.New("my-service", "my-team",
    // Глобальные настройки
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),

    // PostgreSQL
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),

    // Redis
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
        dephealth.Critical(false),
    ),

    // HTTP-сервис
    dephealth.HTTP("auth-service",
        dephealth.FromURL("http://auth.svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),

    // gRPC-сервис
    dephealth.GRPC("user-service",
        dephealth.FromParams("user.svc", "9090"),
        dephealth.Critical(false),
    ),
)
```

## Проверка состояния

### Простой статус

```go
health := dh.Health()
// map[string]bool{
//   "postgres-main:pg.svc:5432":  true,
//   "redis-cache:redis.svc:6379": true,
//   "auth-service:auth.svc:8080": false,
// }
```

### Подробный статус

```go
details := dh.HealthDetails()
for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s latency=%v\n",
        key, *ep.Healthy, ep.Status, ep.Latency)
}
```

`HealthDetails()` возвращает структуру `EndpointStatus` с состоянием
здоровья, категорией статуса, задержкой, временными метками и
пользовательскими метками. До завершения первой проверки `Healthy`
равен `nil`, а `Status` — `"unknown"`.

## Дальнейшие шаги

- [Чекеры](checkers.ru.md) — подробное руководство по всем 8 встроенным чекерам
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Пулы соединений](connection-pools.ru.md) — интеграция с существующими пулами соединений
- [Аутентификация](authentication.ru.md) — авторизация для HTTP, gRPC и чекеров баз данных
- [Метрики](metrics.ru.md) — справочник по метрикам Prometheus и примеры PromQL
- [API Reference](api-reference.ru.md) — полный справочник по всем публичным символам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
- [Руководство по миграции](migration.ru.md) — инструкции по обновлению версий
- [Стиль кода](code-style.ru.md) — соглашения по стилю кода Go
- [Примеры](examples/) — полные рабочие примеры
