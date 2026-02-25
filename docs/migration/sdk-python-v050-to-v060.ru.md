*[English version](sdk-python-v050-to-v060.md)*

# Python SDK: Миграция с v0.5.0 на v0.6.0

Руководство по миграции Python SDK на версию v0.6.0.

> Этот релиз затрагивает **только Python SDK**. Go SDK остаётся на v0.7.0;
> Java SDK остаётся на v0.6.0; C# SDK остаётся на v0.5.0.

## Что изменилось

Три новых асинхронных метода (плюс синхронные варианты) `DependencyHealth`
позволяют динамически управлять эндпоинтами в рантайме:

| Метод | Описание |
| --- | --- |
| `add_endpoint` / `add_endpoint_sync` | Добавить мониторируемый эндпоинт после `start()` |
| `remove_endpoint` / `remove_endpoint_sync` | Удалить эндпоинт (отменяет задачу, удаляет метрики) |
| `update_endpoint` / `update_endpoint_sync` | Атомарно заменить эндпоинт новым |

Новое исключение `EndpointNotFoundError` выбрасывается методом
`update_endpoint`, если старый эндпоинт не найден.

---

## Нужно ли менять код?

**Нет.** Это полностью обратно совместимый релиз. Весь существующий код
продолжает работать без изменений.

---

## Новая возможность: динамические эндпоинты

До v0.6.0 все зависимости регистрировались заранее через фабричные функции
(`http_check()`, `postgres_check()` и т.д.), передаваемые в конструктор
`DependencyHealth(...)`. После вызова `start()` набор мониторируемых эндпоинтов
был зафиксирован.

Начиная с v0.6.0, можно добавлять, удалять и обновлять эндпоинты на
работающем экземпляре `DependencyHealth`:

```python
from dephealth import DependencyType, Endpoint, EndpointNotFoundError, HealthChecker
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

### Синхронный режим

Для приложений, использующих threading-режим (`start_sync()`):

```python
dh.add_endpoint_sync("api-backend", DependencyType.HTTP, True,
    Endpoint(host="backend-2.svc", port="8080"), HTTPChecker())

dh.remove_endpoint_sync("api-backend", "backend-2.svc", "8080")

dh.update_endpoint_sync("api-backend", "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"), HTTPChecker())
```

### Ключевые особенности

- **Потокобезопасность:** все методы используют `threading.Lock` и могут
  вызываться из нескольких потоков или задач.
- **Идемпотентность:** `add_endpoint` возвращает управление без ошибки, если
  эндпоинт уже существует. `remove_endpoint` возвращает управление без ошибки,
  если эндпоинт не найден.
- **Наследование глобальной конфигурации:** динамически добавленные эндпоинты
  используют глобальный `check_interval` и `timeout`, настроенные в конструкторе.
- **Жизненный цикл метрик:** `remove_endpoint` и `update_endpoint` удаляют
  все метрики Prometheus для старого эндпоинта (health, latency, status,
  status\_detail).

### Валидация

`add_endpoint` и `update_endpoint` проверяют входные параметры:

- `dep_name` должен соответствовать `[a-z][a-z0-9-]*`, макс. 63 символа
- `dep_type` должен быть допустимым `DependencyType`
- `endpoint.host` и `endpoint.port` не должны быть пустыми
- `endpoint.labels` не должны содержать зарезервированные имена меток

Некорректные входные данные приводят к `ValueError`.

### Обработка ошибок

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

## Внутренние изменения

- `CheckScheduler` теперь хранит `threading.Lock` для безопасных мутаций.
- Отслеживание `asyncio.Task` для каждого эндпоинта через словарь `_ep_tasks`
  для поддержки отмены.
- Отслеживание `threading.Thread` + `threading.Event` для каждого эндпоинта
  в синхронном режиме.
- Словарь `_states` (`dict[str, _EndpointState]`) заменяет итерацию по
  `_entries` в `health()` и `health_details()`, защищён блокировкой.

---

## Обновление версии

```toml
# v0.5.0
version = "0.5.0"

# v0.6.0
version = "0.6.0"
```
