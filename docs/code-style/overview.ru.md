*[English version](overview.md)*

# Code Style Guide: общие принципы

Этот документ описывает общие для всех dephealth SDK принципы и соглашения.
Руководства по языкам: [Go](../../sdk-go/docs/code-style.ru.md) | [Java](../../sdk-java/docs/code-style.ru.md) | [Python](../../sdk-python/docs/code-style.ru.md) | [C#](../../sdk-csharp/docs/code-style.ru.md) | [Тестирование](testing.ru.md)

## Философия

dephealth — набор **нативных SDK**, каждый язык имеет собственную идиоматичную реализацию,
объединённую общей [спецификацией](../specification.ru.md).
Не sidecar, не FFI — глубокая интеграция с экосистемой каждого языка.

Ключевые принципы:

- **Идиоматичный код** — каждый SDK следует соглашениям своего языка, а не общему «мета-стилю»
- **Спецификация — источник правды** — имена метрик, метки, поведение проверок и контракты
  конфигурации определены в `spec/` и должны быть идентичны во всех SDK
- **Минимальный публичный API** — экспортируем только то, что нужно пользователям, скрываем детали реализации
- **Без сюрпризов** — разумные значения по умолчанию, предсказуемое поведение, информативные сообщения об ошибках

## Языковые соглашения

| Аспект | Английский | Русский |
| --- | --- | --- |
| Код (переменные, функции, классы) | Да | Нет |
| Комментарии в коде | Да | Нет |
| Файлы документации | Да (основной) | Да (суффикс `.ru.md`) |
| Сообщения коммитов | Да | Нет |
| Логи во время выполнения | Да | - |

## Архитектурные слои

Каждый SDK состоит из 6 слоёв. Код должен быть организован в соответствии с этой структурой:

```text
┌─────────────────────────────────────────────┐
│         Framework Integration               │  Spring Boot / ASP.NET / FastAPI
├─────────────────────────────────────────────┤
│         Metrics Exporter                    │  Prometheus gauges + histograms
├─────────────────────────────────────────────┤
│         Check Scheduler                     │  Периодические проверки
├─────────────────────────────────────────────┤
│         Health Checkers                     │  HTTP, gRPC, TCP, Postgres, Redis, ...
├─────────────────────────────────────────────┤
│         Connection Config Parser            │  URL / параметры / connection string
├─────────────────────────────────────────────┤
│         Core Abstractions                   │  Dependency, Endpoint, HealthChecker
└─────────────────────────────────────────────┘
```

Каждый слой зависит только от слоёв ниже. Framework Integration (верхний) зависит от
Metrics Exporter; Metrics Exporter — от Check Scheduler, и так далее.
Core Abstractions (нижний) не имеют внутренних зависимостей.

## Маппинг типов

Основные типы должны быть согласованы между SDK:

| Концепция | Go | Java | Python | C# |
| --- | --- | --- | --- | --- |
| Модель зависимости | `Dependency` struct | `Dependency` class | `Dependency` dataclass | `Dependency` record |
| Модель эндпоинта | `Endpoint` struct | `Endpoint` class | `Endpoint` dataclass | `Endpoint` record |
| Интерфейс проверки | `HealthChecker` interface | `HealthChecker` interface | `HealthChecker` Protocol | `IHealthChecker` interface |
| Тип зависимости | `string` константа | `DependencyType` enum | `str` литерал | `DependencyType` enum |
| Конфигурация | Functional options | Builder pattern | kwargs конструктора | Builder pattern |
| Результат проверки | `error` (nil = здоров) | `void` (бросает исключение) | `None` (бросает исключение) | `Task` (бросает исключение) |
| Планировщик | goroutines | `ScheduledExecutorService` | `asyncio.Task` | `Task` + `Timer` |

## Расширяемость: добавление нового checker-а

Все SDK следуют одному и тому же паттерну добавления нового checker-а:

1. **Реализовать интерфейс** — создать тип, удовлетворяющий `HealthChecker`
   (Go interface, Java interface, Python Protocol, C# interface)
2. **Зарегистрировать в фабрике** — чтобы планировщик мог создать экземпляр по `DependencyType`
3. **Добавить удобный конструктор** — публичную функцию/метод вида `Kafka("name", ...)`
   для создания `Dependency` с правильным типом и checker-ом
4. **Добавить тесты** — как минимум: happy path, ошибка соединения, таймаут

Интерфейс checker-а намеренно минимален — один метод для проверки, один для получения типа:

```text
Check(endpoint) → успех или ошибка
Type() → string
```

## Потокобезопасность и параллелизм

Весь код SDK, работающий в планировщике проверок, **должен быть потокобезопасным**
(или goroutine-safe, async-safe в зависимости от языка):

- **Checker-ы**: вызываются параллельно для разных эндпоинтов — не должны разделять изменяемое состояние
- **Экспортёр метрик**: обновление gauge/histogram должно быть атомарным (гарантируется клиентскими библиотеками Prometheus)
- **Планировщик**: управляет собственным жизненным циклом (start/stop), должен поддерживать graceful shutdown

Механизмы параллелизма по языкам:

| Язык | Механизм | Ключевое правило |
| --- | --- | --- |
| Go | goroutines + `context.Context` | Передавать `ctx` для отмены; нет разделяемого состояния без синхронизации |
| Java | `ScheduledExecutorService` | Реализации маркированы thread-safe; предпочитать неизменяемые объекты |
| Python | `asyncio` | Никогда не блокировать event loop; использовать `async`-checker-ы |
| C# | `Task` + `CancellationToken` | Использовать `ConfigureAwait(false)` в библиотечном коде |

## Конфигурация

Все SDK используют **builder pattern** (или эквивалент языка) для конфигурации:

- **Разумные значения по умолчанию** — `checkInterval=15s`, `timeout=5s`, `failureThreshold=1`
- **Валидация при построении** — fail fast при невалидной конфигурации (пустое имя, нулевой интервал и т.д.)
- **Неизменяемость после построения** — после создания конфигурация не может быть изменена

```text
// Псевдокод паттерна для всех языков:
DependencyHealth.builder()
    .dependency("name", type, endpoint_config, options...)
    .dependency(...)
    .checkInterval(15s)
    .build()
    .start()
```

Конфигурация соединений принимает несколько форматов:

- Полный URL: `postgres://user:pass@host:5432/db`
- Отдельные параметры: `host`, `port`
- Connection string: `Host=...;Port=...`

SDK автоматически извлекает `host` и `port` из любого формата для меток метрик.
**Учётные данные никогда не экспортируются** в метрики или логи.

## Обработка ошибок

Философия: **fail fast с информативными сообщениями**.

- Ошибки конфигурации (невалидный URL, отсутствующее имя) — немедленная ошибка при build/start
- Ошибки проверки (таймаут, отказ соединения) — отражаются в метриках, логируются
  на соответствующем уровне, **не** ломают приложение
- Неожиданные ошибки в checker-ах — перехватываются, логируются как error,
  эндпоинт отмечается как unhealthy

Каждый SDK определяет базовый тип исключения/ошибки:

| Язык | Базовый тип | Типичные подтипы |
| --- | --- | --- |
| Go | Sentinel errors (`ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`) | Обёрнуты через `fmt.Errorf("%w", ...)` |
| Java | `DepHealthException` (unchecked) | `CheckTimeoutException`, `ConnectionRefusedException` |
| Python | `CheckError` | `CheckTimeoutError`, `CheckConnectionRefusedError`, `UnhealthyError` |
| C# | `DepHealthException` | `CheckTimeoutException`, `ConnectionRefusedException` |

## Логирование

Все SDK используют стандартный фреймворк логирования своего языка:

| Язык | Фреймворк | Имя логгера |
| --- | --- | --- |
| Go | `log/slog` | `dephealth` |
| Java | SLF4J | `biz.kryukov.dev.dephealth` |
| Python | `logging` | `dephealth` |
| C# | `ILogger<T>` | `DepHealth.*` |

Уровни логирования:

| Уровень | Использование |
| --- | --- |
| `ERROR` | Неожиданные ошибки (panic recovery, ошибка регистрации метрик) |
| `WARN` | Ошибки проверки, предупреждения конфигурации |
| `INFO` | Запуск/остановка планировщика, зависимость зарегистрирована |
| `DEBUG` | Результаты отдельных проверок, тайминги |

Правила:

- **Никогда не логировать учётные данные** — URL-ы санитизируются перед логированием
- **Использовать структурированное логирование** где доступно (slog fields, SLF4J MDC, Python extra)
- **Параметризованные сообщения** — без конкатенации строк для сообщений логов
  (например, SLF4J `log.warn("Check failed for {}", name)`, а не `log.warn("Check failed for " + name)`)

## Сводка соглашений об именовании

| Концепция | Go | Java | Python | C# |
| --- | --- | --- | --- | --- |
| Пакет/пространство имён | `dephealth` | `biz.kryukov.dev.dephealth` | `dephealth` | `DepHealth` |
| Публичный тип | `PascalCase` | `PascalCase` | `PascalCase` | `PascalCase` |
| Публичный метод | `PascalCase` | `camelCase` | `snake_case` | `PascalCase` |
| Приватное поле | `camelCase` | `camelCase` | `_snake_case` | `_camelCase` |
| Константа | `PascalCase` | `UPPER_SNAKE_CASE` | `UPPER_SNAKE_CASE` | `PascalCase` |
| Тестовый метод | `TestXxx` | `xxxTest` / `@Test` | `test_xxx` | `Xxx_Should_Yyy` |

Полные правила именования — в руководствах по каждому языку.

## Формат документации

Все файлы документации следуют двуязычному формату EN/RU:

- Английский файл: оригинальное имя (например, `overview.md`)
- Русский файл: суффикс `.ru.md` (например, `overview.ru.md`)
- EN начинается с `*[Русская версия](overview.ru.md)*`
- RU начинается с `*[English version](overview.md)*`
- Внутренние ссылки в RU-файлах ведут на `.ru.md` версии
- Markdown должен проходить `markdownlint` (MD013 для длины строк отключено)
