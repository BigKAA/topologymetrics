*[English version](migration.md)*

# Руководство по миграции

Инструкции по миграции между версиями Go SDK dephealth.

## v0.7.0 → v0.8.0 (LDAP Checker)

v0.8.0 добавляет LDAP health checker. Обратная совместимость сохранена.

Новый импорт для поддержки LDAP:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"
```

Использование:

```go
dephealth.LDAP("directory",
    dephealth.FromURL("ldap://ldap.svc:389"),
    dephealth.Critical(true),
    dephealth.WithLDAPCheckMethod("root_dse"),
)
```

Методы проверки: `anonymous_bind`, `simple_bind`, `root_dse` (по умолчанию), `search`.

Подробнее в [общем руководстве по миграции](../../docs/migration/v070-to-v080.ru.md).

---

## v0.6.0 → v0.7.0 (Dynamic Endpoints)

v0.7.0 добавляет управление endpoint-ами в рантайме. Обратная совместимость сохранена.

Новые методы на работающем экземпляре `DepHealth`:

```go
// Добавить новый endpoint после Start()
err := dh.AddEndpoint("api-backend", dephealth.TypeHTTP, true,
    dephealth.Endpoint{Host: "backend-2.svc", Port: "8080"},
    httpcheck.New(),
)

// Удалить endpoint (отменяет горутину, удаляет метрики)
err = dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080")

// Атомарно заменить endpoint
err = dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    dephealth.Endpoint{Host: "backend-3.svc", Port: "8080"},
    httpcheck.New(),
)
```

Подробнее в [общем руководстве по миграции](../../docs/migration/v060-to-v070.ru.md).

---

## v0.5.0 → v0.6.0 (Split Checkers)

v0.6.0 вводит выборочный импорт чекеров для уменьшения размера бинарника.

### Breaking: обязательная регистрация чекеров

До v0.6.0 все чекеры регистрировались автоматически. Начиная с v0.6.0,
необходимо явно импортировать пакеты чекеров.

**Импорт всех чекеров (обратная совместимость):**

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

**Импорт только нужных:**

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Доступные подпакеты: `tcpcheck`, `httpcheck`, `grpccheck`, `pgcheck`,
`mysqlcheck`, `redischeck`, `amqpcheck`, `kafkacheck`.

Подробнее в [Выборочный импорт](selective-imports.ru.md) и
[общем руководстве по миграции](../../docs/migration/v050-to-v060.ru.md).

---

## v0.4.x → v0.5.0

### Breaking: обязательный параметр `group`

v0.5.0 добавляет обязательный параметр `group` (логическая группировка: команда, подсистема, проект).

```go
// v0.4.x
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)

// v0.5.0
dh, err := dephealth.New("my-service", "my-team",
    dephealth.Postgres("postgres-main", ...),
)
```

Альтернатива: переменная окружения `DEPHEALTH_GROUP` (API имеет приоритет).

Валидация: те же правила, что и для `name` — `[a-z][a-z0-9-]*`, 1-63 символа.

Подробнее в [общем руководстве по миграции](../../docs/migration/v042-to-v050.ru.md).

---

## v0.4.0 → v0.4.1

### Новое: HealthDetails() API

В v0.4.1 добавлен метод `HealthDetails()`, возвращающий детальный статус каждого
endpoint-а. Изменений в существующем API нет — это чисто аддитивная функция.

```go
details := dh.HealthDetails()
// map[string]dephealth.EndpointStatus

for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s detail=%s latency=%v\n",
        key, ep.Healthy, ep.Status, ep.Detail, ep.Latency)
}
```

Поля `EndpointStatus`: `Dependency`, `Type`, `Host`, `Port`, `Healthy` (`*bool`),
`Status`, `Detail`, `Latency`, `LastCheckedAt`, `Critical`, `Labels`.

До завершения первой проверки `Healthy` равен `nil`, а `Status` — `"unknown"`.

---

## v0.3.x → v0.4.0

### Новые метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически наряду с существующими `app_dependency_health` и `app_dependency_latency_seconds`.

### Влияние на хранилище

Каждый endpoint теперь создаёт 9 дополнительных временных рядов (8 для `app_dependency_status` + 1 для `app_dependency_status_detail`). Для сервиса с 5 endpoint-ами это добавляет 45 рядов.

### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

Полный список значений статуса см. в [Контракт метрик](../../spec/metric-contract.ru.md).

---

## v0.2 → v0.3.0

### Breaking: новый module path

В v0.3.0 module path изменён с `github.com/BigKAA/topologymetrics`
на `github.com/BigKAA/topologymetrics/sdk-go`.

Это исправляет работу `go get` — стандартный подход для Go-модулей
в монорепозиториях, где `go.mod` находится в поддиректории.

### Шаги миграции

1. Обновите зависимость:

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

1. Замените import paths во всех файлах:

```bash
# Массовая замена (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

1. Обновите `go.mod` — удалите старую зависимость:

```bash
go mod tidy
```

API и поведение SDK не изменились — только module path.

---

## v0.1 → v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `dephealth.New(...)` | `dephealth.New("my-service", ...)` | Обязательный первый аргумент `name` |
| `dephealth.Critical(true)` (необязателен) | `dephealth.Critical(true/false)` (обязателен) | Для каждой зависимости |
| `Endpoint.Metadata` | `Endpoint.Labels` | Переименование поля |
| `dephealth.WithMetadata(map)` | `dephealth.WithLabel("key", "value")` | Произвольные метки |
| `WithOptionalLabels(...)` | удалён | Произвольные метки через `WithLabel` |

### Обязательные изменения

1. Добавьте `name` первым аргументом в `dephealth.New()`:

```go
// v0.1
dh, err := dephealth.New(
    dephealth.Postgres("postgres-main", ...),
)

// v0.2
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)
```

1. Укажите `Critical()` для каждой зависимости:

```go
// v0.1 — Critical необязателен
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
)

// v0.2 — Critical обязателен
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
    dephealth.Critical(false),
)
```

1. Замените `WithMetadata` на `WithLabel` (если используется):

```go
// v0.1
dephealth.WithMetadata(map[string]string{"role": "primary"})

// v0.2
dephealth.WithLabel("role", "primary")
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.
