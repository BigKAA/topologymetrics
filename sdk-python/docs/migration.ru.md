*[English version](migration.md)*

# Руководство по миграции

Инструкции по обновлению версий dephealth Python SDK.

## v0.5.0 → v0.6.0

> Этот релиз затрагивает **только Python SDK**. Go SDK остаётся на v0.7.0;
> Java SDK остаётся на v0.6.0; C# SDK остаётся на v0.5.0.

### Нужно ли менять код?

**Нет.** Это полностью обратно совместимый релиз. Весь существующий код
продолжает работать без изменений.

### Новая возможность: динамические эндпоинты

Три новых асинхронных метода (плюс синхронные варианты) `DependencyHealth`
позволяют динамически управлять эндпоинтами в рантайме:

| Метод | Описание |
| --- | --- |
| `add_endpoint` / `add_endpoint_sync` | Добавить мониторируемый эндпоинт после `start()` |
| `remove_endpoint` / `remove_endpoint_sync` | Удалить эндпоинт (отменяет задачу, удаляет метрики) |
| `update_endpoint` / `update_endpoint_sync` | Атомарно заменить эндпоинт новым |

Новое исключение `EndpointNotFoundError` выбрасывается методом
`update_endpoint`, если старый эндпоинт не найден.

```python
from dephealth import DependencyType, Endpoint, EndpointNotFoundError
from dephealth.checks.http import HTTPChecker

# После dh.start()...

# Добавить новый эндпоинт
await dh.add_endpoint(
    "api-backend",
    DependencyType.HTTP,
    True,
    Endpoint(host="backend-2.svc", port="8080"),
    HTTPChecker(),
)

# Удалить эндпоинт
await dh.remove_endpoint("api-backend", "backend-2.svc", "8080")

# Заменить эндпоинт атомарно
await dh.update_endpoint(
    "api-backend",
    "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"),
    HTTPChecker(),
)
```

#### Синхронный режим

Для приложений, использующих threading-режим (`start_sync()`):

```python
dh.add_endpoint_sync("api-backend", DependencyType.HTTP, True,
    Endpoint(host="backend-2.svc", port="8080"), HTTPChecker())

dh.remove_endpoint_sync("api-backend", "backend-2.svc", "8080")

dh.update_endpoint_sync("api-backend", "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"), HTTPChecker())
```

#### Ключевые особенности

- **Потокобезопасность:** все методы используют `threading.Lock` и могут
  вызываться из нескольких потоков или задач.
- **Идемпотентность:** `add_endpoint` возвращает управление без ошибки, если
  эндпоинт уже существует. `remove_endpoint` возвращает управление без ошибки,
  если эндпоинт не найден.
- **Наследование глобальной конфигурации:** динамически добавленные эндпоинты
  используют глобальный `check_interval` и `timeout`.
- **Жизненный цикл метрик:** `remove_endpoint` и `update_endpoint` удаляют
  все метрики Prometheus для старого эндпоинта.

#### Обработка ошибок

```python
from dephealth import EndpointNotFoundError

try:
    await dh.update_endpoint("api", "old-host", "8080", new_ep, checker)
except EndpointNotFoundError:
    # старый эндпоинт не существует — используйте add_endpoint
    pass
except RuntimeError:
    # планировщик не запущен или уже остановлен
    pass
```

---

## v0.4.x → v0.5.0

См. также: [кросс-SDK миграция](../../docs/migration/v042-to-v050.md)

### Обязательный параметр `group`

v0.5.0 добавляет обязательный параметр `group` (логическая группировка:
команда, подсистема, проект).

```python
# v0.4.x
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", ...),
)

# v0.5.0
dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main", ...),
)
```

FastAPI:

```python
# v0.5.0
app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main", ...),
    )
)
```

Альтернатива: переменная окружения `DEPHEALTH_GROUP` (API имеет приоритет).

Валидация: те же правила, что и для `name` — `[a-z][a-z0-9-]*`, 1-63 символа.

---

## v0.4.0 → v0.4.1

### Новое: health_details() API

В v0.4.1 добавлен метод `health_details()`, возвращающий детальный статус
каждого эндпоинта. Изменений в существующем API нет.

```python
details = dh.health_details()
# dict[str, EndpointStatus]

for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"detail={ep.detail} latency={ep.latency_millis():.1f}ms")
```

Поля `EndpointStatus`: `dependency`, `type`, `host`, `port`,
`healthy` (`bool | None`, `None` = неизвестно), `status`, `detail`,
`latency`, `last_checked_at`, `critical`, `labels`.

> **Примечание:** `health_details()` использует ключи по эндпоинтам (`"dep:host:port"`),
> тогда как `health()` — агрегированные ключи (`"dep"`). Метод `to_dict()`
> сериализует `EndpointStatus` в JSON-совместимый словарь.

---

## v0.3.x → v0.4.0

### Новые метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на эндпоинт, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически.

### Влияние на хранилище

Каждый эндпоинт создаёт 9 дополнительных временных рядов. Для сервиса
с 5 эндпоинтами это добавляет 45 рядов.

### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

Полный список значений статуса см. в [Спецификации](../../spec/).

---

## v0.1 → v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `DependencyHealth(...)` | `DependencyHealth("my-service", ...)` | Обязательный первый аргумент `name` |
| `dephealth_lifespan(...)` | `dephealth_lifespan("my-service", ...)` | Обязательный первый аргумент `name` |
| `critical=True` (необязателен) | `critical=True/False` (обязателен) | Для каждой фабрики |
| нет | `labels={"key": "value"}` | Произвольные метки |

### Обязательные изменения

1. Добавьте `name` первым аргументом:

   ```python
   # v0.1
   dh = DependencyHealth(
       postgres_check("postgres-main", url="postgresql://..."),
   )

   # v0.2
   dh = DependencyHealth("my-service",
       postgres_check("postgres-main", url="postgresql://...", critical=True),
   )
   ```

2. Укажите `critical` для каждой зависимости:

   ```python
   # v0.1 — critical необязателен
   redis_check("redis-cache", url="redis://redis.svc:6379")

   # v0.2 — critical обязателен
   redis_check("redis-cache", url="redis://redis.svc:6379", critical=False)
   ```

3. Обновите `dephealth_lifespan` (FastAPI):

   ```python
   # v0.1
   app = FastAPI(
       lifespan=dephealth_lifespan(
           http_check("api", url="http://api:8080"),
       )
   )

   # v0.2
   app = FastAPI(
       lifespan=dephealth_lifespan("my-service",
           http_check("api", url="http://api:8080", critical=True),
       )
   )
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

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [API Reference](api-reference.ru.md) — полный справочник по публичным классам
