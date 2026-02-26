*[English version](getting-started.md)*

# Начало работы

Руководство по установке, базовой настройке и первой проверке состояния
зависимости с помощью C# SDK dephealth.

## Требования

- .NET 8 или новее
- ASP.NET Core (Minimal API или MVC)
- Работающая зависимость для мониторинга (PostgreSQL, Redis, HTTP-сервис и т.д.)

## Установка

Core-пакет (программный API):

```bash
dotnet add package DepHealth.Core
```

Интеграция с ASP.NET Core (включает Core):

```bash
dotnet add package DepHealth.AspNetCore
```

Интеграция с Entity Framework:

```bash
dotnet add package DepHealth.EntityFramework
```

## Минимальный пример

Мониторинг одной HTTP-зависимости с экспортом метрик Prometheus:

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddHttp("payment-api", "http://payment.svc:8080", critical: true)
    .Build();

dh.Start();

// Метрики доступны через prometheus-net по адресу /metrics

// Graceful shutdown
Console.CancelKeyPress += (_, _) => dh.Stop();
```

После запуска метрики Prometheus доступны по адресу `/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Ключевые концепции

### Имя и группа

Каждый экземпляр `DepHealthMonitor` требует два идентификатора:

- **name** — уникальное имя приложения (например, `"my-service"`)
- **group** — логическая группа сервиса (например, `"my-team"`, `"payments"`)

Оба значения появляются как метки во всех экспортируемых метриках.
Правила валидации: `[a-z][a-z0-9-]*`, от 1 до 63 символов.

Если не переданы как аргументы, SDK использует переменные окружения
`DEPHEALTH_NAME` и `DEPHEALTH_GROUP` как запасной вариант.

### Зависимости

Каждая зависимость регистрируется через методы `Add*()` билдера
с указанием `DependencyType`:

| DependencyType | Описание |
| --- | --- |
| `Http` | HTTP-сервис |
| `Grpc` | gRPC-сервис |
| `Tcp` | TCP-эндпоинт |
| `Postgres` | База данных PostgreSQL |
| `MySql` | База данных MySQL |
| `Redis` | Сервер Redis |
| `Amqp` | RabbitMQ (AMQP-брокер) |
| `Kafka` | Брокер Apache Kafka |
| `Ldap` | LDAP-сервер каталогов |

Для каждой зависимости обязательны:

- **Имя** (первый аргумент) — идентификатор зависимости в метриках
- **Эндпоинт** — через URL-строку или `host` + `port`
- **Флаг критичности** — `critical: true` или `critical: false`

### Жизненный цикл

1. **Создание** — `DepHealthMonitor.CreateBuilder(...).Build()`
2. **Запуск** — `dh.Start()` запускает периодические проверки
3. **Работа** — проверки выполняются с заданным интервалом (по умолчанию 15 сек)
4. **Остановка** — `dh.Stop()` останавливает проверки и освобождает ресурсы

## Несколько зависимостей

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Глобальные настройки
    .WithCheckInterval(TimeSpan.FromSeconds(30))
    .WithCheckTimeout(TimeSpan.FromSeconds(3))

    // PostgreSQL
    .AddPostgres("postgres-main",
        Environment.GetEnvironmentVariable("DATABASE_URL")!,
        critical: true)

    // Redis
    .AddRedis("redis-cache",
        Environment.GetEnvironmentVariable("REDIS_URL")!,
        critical: false)

    // HTTP-сервис
    .AddHttp("auth-service", "http://auth.svc:8080",
        healthPath: "/healthz",
        critical: true)

    // gRPC-сервис
    .AddGrpc("user-service", "user.svc", "9090",
        critical: false)

    .Build();
```

## Проверка состояния

### Простой статус

```csharp
Dictionary<string, bool> health = dh.Health();
// {"postgres-main": true, "redis-cache": true}

// Использование для readiness probe
bool allHealthy = health.Values.All(v => v);
```

### Подробный статус

```csharp
Dictionary<string, EndpointStatus> details = dh.HealthDetails();
foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"latency={ep.LatencyMillis:F1}ms");
}
```

`HealthDetails()` возвращает объект `EndpointStatus` с состоянием
здоровья, категорией статуса, задержкой, временными метками и
пользовательскими метками. До завершения первой проверки `Healthy`
равен `null`, а `Status` — `"unknown"`.

## Дальнейшие шаги

- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 встроенным чекерам
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Пулы соединений](connection-pools.ru.md) — интеграция с существующими пулами соединений
- [Интеграция с ASP.NET Core](aspnetcore.ru.md) — регистрация в DI, hosted service, health endpoints
- [Entity Framework](entity-framework.ru.md) — проверки состояния на основе DbContext
- [Аутентификация](authentication.ru.md) — авторизация для HTTP, gRPC и чекеров БД
- [Метрики](metrics.ru.md) — справочник по метрикам Prometheus и примеры PromQL
- [API Reference](api-reference.ru.md) — полный справочник по всем публичным классам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
- [Руководство по миграции](migration.ru.md) — инструкции по обновлению версий
- [Стиль кода](code-style.ru.md) — соглашения по стилю кода C#
- [Примеры](examples/) — полные рабочие примеры
