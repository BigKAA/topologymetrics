*[English version](migration.md)*

# Руководство по миграции

Инструкции по обновлению версий C# SDK.

## v0.7.0 → v0.8.0

### Новое: LDAP Health Checker

v0.8.0 добавляет LDAP health checker. Изменений в существующем API нет —
это чисто аддитивная функция.

Новые типы: `LdapChecker`, `LdapCheckMethod`, `LdapSearchScope`.

Новый метод билдера: `AddLdap()`.

Подробности: [Checkers — LDAP](checkers.ru.md#ldap).

---

## v0.6.0 → v0.7.0

### Новое: метрики статуса

v0.7.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически
наряду с существующими `app_dependency_health` и
`app_dependency_latency_seconds`.

#### Влияние на хранилище

Каждый endpoint теперь создаёт 9 дополнительных временных рядов (8 для
`app_dependency_status` + 1 для `app_dependency_status_detail`). Для сервиса
с 5 endpoint-ами это добавляет 45 рядов.

#### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

---

## v0.5.0 → v0.6.0

### Новое: динамическое управление эндпоинтами

v0.6.0 добавляет три новых метода `DepHealthMonitor` для динамического
управления эндпоинтами в рантайме. Изменений в существующем API нет —
это полностью обратно совместимый релиз.

| Метод | Описание |
| --- | --- |
| `AddEndpoint` | Добавить мониторируемый эндпоинт после `Start()` |
| `RemoveEndpoint` | Удалить эндпоинт (отменяет задачу, удаляет метрики) |
| `UpdateEndpoint` | Атомарно заменить эндпоинт новым |

```csharp
using DepHealth;
using DepHealth.Checks;

// После dh.Start()...

// Добавить новый эндпоинт
dh.AddEndpoint("api-backend", DependencyType.Http, true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpChecker());

// Удалить эндпоинт
dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080");

// Заменить эндпоинт атомарно
dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpChecker());
```

#### Ключевые особенности

- **Потокобезопасность:** все три метода используют `lock` и могут
  вызываться из нескольких потоков. Операции чтения (`Health()`,
  `HealthDetails()`) остаются lock-free благодаря `ConcurrentDictionary`.
- **Идемпотентность:** `AddEndpoint` возвращает управление без ошибки, если
  эндпоинт уже существует. `RemoveEndpoint` возвращает управление без ошибки,
  если эндпоинт не найден.
- **Наследование глобальной конфигурации:** динамически добавленные эндпоинты
  используют глобальный интервал проверки и тайм-аут, настроенные в билдере.
- **Жизненный цикл метрик:** `RemoveEndpoint` и `UpdateEndpoint` удаляют
  все метрики Prometheus для старого эндпоинта.

#### Валидация

`AddEndpoint` и `UpdateEndpoint` проверяют входные параметры:

- `depName` должен соответствовать `[a-z][a-z0-9-]*`, макс. 63 символа
- `depType` должен быть допустимым значением перечисления `DependencyType`
- `ep.Host` и `ep.Port` не должны быть пустыми
- `ep.Labels` не должны содержать зарезервированные имена меток

Некорректные входные данные приводят к `ValidationException`.

#### Обработка ошибок

```csharp
using DepHealth.Exceptions;

try
{
    dh.UpdateEndpoint("api", "old-host", "8080", newEp, checker);
}
catch (EndpointNotFoundException)
{
    // старый эндпоинт не существует — используйте AddEndpoint
}
catch (InvalidOperationException)
{
    // планировщик не запущен или уже остановлен
}
```

Новое исключение: `EndpointNotFoundException`.

#### Обновление версии

```xml
<!-- v0.5.0 -->
<Version>0.5.0</Version>

<!-- v0.6.0 -->
<Version>0.6.0</Version>
```

---

## v0.4.1 → v0.5.0

### Ломающее изменение: обязательный параметр `group`

v0.5.0 добавляет обязательный параметр `group` (логическая группировка:
команда, подсистема, проект).

```csharp
// v0.4.x
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency(...)
);

// v0.5.0
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddDependency(...)
);
```

Для standalone API:

```csharp
// v0.4.x
DepHealthMonitor.CreateBuilder("my-service")

// v0.5.0
DepHealthMonitor.CreateBuilder("my-service", "my-team")
```

Альтернатива: переменная окружения `DEPHEALTH_GROUP` (API имеет приоритет).

Валидация: те же правила, что и для `name` — `[a-z][a-z0-9-]*`, 1-63 символа.

---

## v0.4.0 → v0.4.1

### Новое: HealthDetails() API

В v0.4.1 добавлен метод `HealthDetails()`, возвращающий детальный статус
каждого endpoint-а. Изменений в существующем API нет.

```csharp
Dictionary<string, EndpointStatus> details = dh.HealthDetails();

foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"detail={ep.Detail} latency={ep.LatencyMillis:F1}ms");
}
```

Свойства `EndpointStatus`: `Dependency`, `Type`, `Host`, `Port`,
`Healthy` (`bool?`, `null` = неизвестно), `Status`, `Detail`,
`Latency`, `LastCheckedAt`, `Critical`, `Labels`.

JSON-сериализация использует `System.Text.Json` с именованием snake_case.

---

## v0.3.x → v0.4.0

### Новое: метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически
наряду с существующими `app_dependency_health` и
`app_dependency_latency_seconds`.

#### Влияние на хранилище

Каждый endpoint теперь создаёт 9 дополнительных временных рядов (8 для
`app_dependency_status` + 1 для `app_dependency_status_detail`). Для сервиса
с 5 endpoint-ами это добавляет 45 рядов.

#### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

---

## v0.1 → v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `AddDepHealth(dh => ...)` | `AddDepHealth("my-service", dh => ...)` | Обязательный первый аргумент `name` |
| `CreateBuilder()` | `CreateBuilder("my-service")` | Обязательный аргумент `name` |
| `.Critical(true)` (необязателен) | `.Critical(true/false)` (обязателен) | Для каждой зависимости |
| нет | `.Label("key", "value")` | Произвольные метки |

### Обязательные изменения

1. Добавьте `name` в `AddDepHealth`:

```csharp
// v0.1
builder.Services.AddDepHealth(dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);

// v0.2
builder.Services.AddDepHealth("my-service", dh => dh
    .AddDependency("postgres-main", DependencyType.Postgres, d => d
        .Url("postgres://user:pass@pg.svc:5432/mydb")
        .Critical(true))
);
```

1. Укажите `.Critical()` для каждой зависимости:

```csharp
// v0.1 — Critical необязателен
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379"))

// v0.2 — Critical обязателен
.AddDependency("redis-cache", DependencyType.Redis, d => d
    .Url("redis://redis.svc:6379")
    .Critical(false))
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.

## См. также

- [Начало работы](getting-started.ru.md) — установка и базовая настройка
- [Конфигурация](configuration.ru.md) — все параметры, значения по умолчанию и валидация
- [Справочник API](api-reference.ru.md) — полный справочник всех публичных классов
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и их решения
