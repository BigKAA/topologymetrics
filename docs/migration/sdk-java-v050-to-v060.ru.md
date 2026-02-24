*[English version](sdk-java-v050-to-v060.md)*

# Java SDK: Миграция с v0.5.0 на v0.6.0

Руководство по миграции Java SDK на версию v0.6.0.

> Этот релиз затрагивает **только Java SDK**. Go SDK остаётся на v0.7.0;
> Python и C# SDK остаются на v0.5.0.

## Что изменилось

Три новых метода `DepHealth` позволяют динамически управлять эндпоинтами
в рантайме:

| Метод | Описание |
| --- | --- |
| `addEndpoint` | Добавить мониторируемый эндпоинт после `start()` |
| `removeEndpoint` | Удалить эндпоинт (отменяет запланированную задачу, удаляет метрики) |
| `updateEndpoint` | Атомарно заменить эндпоинт новым |

Новое исключение `EndpointNotFoundException` (наследует `DepHealthException`)
выбрасывается методом `updateEndpoint`, если старый эндпоинт не найден.

---

## Нужно ли менять код?

**Нет.** Это полностью обратно совместимый релиз. Весь существующий код
продолжает работать без изменений.

---

## Новая возможность: динамические эндпоинты

До v0.6.0 все зависимости регистрировались заранее через вызовы
`.dependency()` на билдере. После вызова `build()` набор мониторируемых
эндпоинтов был зафиксирован.

Начиная с v0.6.0, можно добавлять, удалять и обновлять эндпоинты на
работающем экземпляре `DepHealth`:

```java
import biz.kryukov.dev.dephealth.*;

// После depHealth.start()...

// Добавить новый эндпоинт
depHealth.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpHealthChecker());

// Удалить эндпоинт
depHealth.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Заменить эндпоинт атомарно
depHealth.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpHealthChecker());
```

### Ключевые особенности

- **Потокобезопасность:** все три метода синхронизированы и могут вызываться
  из нескольких потоков.
- **Идемпотентность:** `addEndpoint` возвращает управление без ошибки, если
  эндпоинт уже существует. `removeEndpoint` возвращает управление без ошибки,
  если эндпоинт не найден.
- **Наследование глобальной конфигурации:** динамически добавленные эндпоинты
  используют глобальный интервал и тайм-аут, настроенные в билдере.
- **Жизненный цикл метрик:** `removeEndpoint` и `updateEndpoint` удаляют
  все метрики Prometheus для старого эндпоинта (health, latency, status,
  status\_detail).

### Валидация

`addEndpoint` и `updateEndpoint` проверяют входные параметры:

- `depName` должен соответствовать `[a-z][a-z0-9-]*`, макс. 63 символа
- `depType` не должен быть null
- `ep.host()` и `ep.port()` не должны быть пустыми
- `ep.labels()` не должны содержать зарезервированные имена меток

Некорректные входные данные приводят к `ValidationException`.

### Обработка ошибок

```java
try {
    depHealth.updateEndpoint("api", "old-host", "8080", newEp, checker);
} catch (EndpointNotFoundException e) {
    // старый эндпоинт не существует — используйте addEndpoint
} catch (IllegalStateException e) {
    // планировщик не запущен или уже остановлен
}
```

---

## Внутренние изменения

- `CheckScheduler` теперь хранит `ScheduledFuture` для каждого эндпоинта
  для поддержки отмены.
- Карта `states` изменена с `HashMap` на `ConcurrentHashMap` для
  безопасной конкурентной итерации в `health()` / `healthDetails()`.
- `ScheduledThreadPoolExecutor` заменяет фиксированный
  `ScheduledExecutorService` (поддерживает динамическое изменение размера).
- `MetricsExporter.deleteMetrics()` удаляет все 4 семейства метрик
  для указанного эндпоинта.

---

## Обновление версии

```xml
<!-- v0.5.0 -->
<version>0.5.0</version>

<!-- v0.6.0 -->
<version>0.6.0</version>
```
