*[English version](sdk-csharp-v050-to-v060.md)*

# C# SDK: Миграция с v0.5.0 на v0.6.0

Руководство по миграции C# SDK на версию v0.6.0.

> Этот релиз затрагивает **только C# SDK**. Go SDK остаётся на v0.7.0;
> Java SDK остаётся на v0.6.0; Python SDK остаётся на v0.6.0.

## Что изменилось

Три новых метода `DepHealthMonitor` позволяют динамически управлять
эндпоинтами в рантайме:

| Метод | Описание |
| --- | --- |
| `AddEndpoint` | Добавить мониторируемый эндпоинт после `Start()` |
| `RemoveEndpoint` | Удалить эндпоинт (отменяет задачу, удаляет метрики) |
| `UpdateEndpoint` | Атомарно заменить эндпоинт новым |

Новое исключение `EndpointNotFoundException` (наследует
`InvalidOperationException`) выбрасывается методом `UpdateEndpoint`,
если старый эндпоинт не найден.

---

## Нужно ли менять код?

**Нет.** Это полностью обратно совместимый релиз. Весь существующий код
продолжает работать без изменений.

---

## Новая возможность: динамические эндпоинты

До v0.6.0 все зависимости регистрировались заранее через методы билдера
(`AddHttp()`, `AddPostgres()` и т.д.). После вызова `Build()` набор
мониторируемых эндпоинтов был зафиксирован.

Начиная с v0.6.0, можно добавлять, удалять и обновлять эндпоинты на
работающем экземпляре `DepHealthMonitor`:

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

### Ключевые особенности

- **Потокобезопасность:** все три метода используют `lock` и могут
  вызываться из нескольких потоков. Операции чтения (`Health()`,
  `HealthDetails()`) остаются lock-free благодаря `ConcurrentDictionary`.
- **Идемпотентность:** `AddEndpoint` возвращает управление без ошибки, если
  эндпоинт уже существует. `RemoveEndpoint` возвращает управление без ошибки,
  если эндпоинт не найден.
- **Наследование глобальной конфигурации:** динамически добавленные эндпоинты
  используют глобальный интервал проверки и тайм-аут, настроенные в билдере.
- **Жизненный цикл метрик:** `RemoveEndpoint` и `UpdateEndpoint` удаляют
  все метрики Prometheus для старого эндпоинта (health, latency, status,
  status\_detail).

### Валидация

`AddEndpoint` и `UpdateEndpoint` проверяют входные параметры:

- `depName` должен соответствовать `[a-z][a-z0-9-]*`, макс. 63 символа
- `depType` должен быть допустимым значением перечисления `DependencyType`
- `ep.Host` и `ep.Port` не должны быть пустыми
- `ep.Labels` не должны содержать зарезервированные имена меток

Некорректные входные данные приводят к `ValidationException`.

### Обработка ошибок

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

---

## Внутренние изменения

- `CheckScheduler` теперь хранит глобальный `CheckConfig` для динамических
  эндпоинтов.
- `_states` изменён с `Dictionary` на `ConcurrentDictionary` для безопасной
  параллельной итерации в `Health()` / `HealthDetails()`.
- Отслеживание `CancellationTokenSource` для каждого эндпоинта через словарь
  `_cancellations` (ключ `"name:host:port"`), связанный с глобальным CTS.
- `object _mutationLock` синхронизирует операции добавления/удаления/обновления,
  при этом операции чтения остаются lock-free.

---

## Обновление версии

```xml
<!-- v0.5.0 -->
<Version>0.5.0</Version>

<!-- v0.6.0 -->
<Version>0.6.0</Version>
```
