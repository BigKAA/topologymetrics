*[English version](entity-framework.md)*

# Интеграция с Entity Framework

Пакет `DepHealth.EntityFramework` обеспечивает интеграцию с
Entity Framework Core. Он позволяет использовать существующий `DbContext`
для проверок состояния PostgreSQL вместо создания отдельных подключений.

## Установка

```bash
dotnet add package DepHealth.EntityFramework
```

Пакет зависит от `DepHealth.Core` и добавляет метод расширения
`AddNpgsqlFromContext<TContext>()`.

## Зачем использовать интеграцию с EF

| Аспект | Автономный режим | Entity Framework |
| --- | --- | --- |
| Подключение | Новое при каждой проверке | Повторно использует пул подключений DbContext |
| Отражает реальное состояние | Частично | Да |
| Обнаруживает исчерпание пула | Нет | Да |
| Конфигурация | Строка URL | Автоматически из DbContext |
| Дополнительная нагрузка на БД | Да (лишние подключения) | Нет |

## Базовое использование

```csharp
using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.EntityFramework;

var builder = WebApplication.CreateBuilder(args);

// Register your DbContext
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(builder.Configuration["DATABASE_URL"]));

// Register dephealth with EF integration
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main", critical: true)
    .AddRedis("redis-cache",
        builder.Configuration["REDIS_URL"]!,
        critical: false)
);

var app = builder.Build();
app.MapDepHealthEndpoints();
app.Run();
```

## Как это работает

`AddNpgsqlFromContext<TContext>()` выполняет следующие шаги:

1. Разрешает `TContext` из DI-контейнера
2. Извлекает строку подключения через `context.Database.GetConnectionString()`
3. Разбирает хост и порт из строки подключения
4. Создаёт `PostgresChecker`, используя строку подключения
5. Регистрирует зависимость в builder

Проверка состояния выполняет `SELECT 1`, используя подключение из того же
пула, что и приложение. Это означает:

- Если пул исчерпан, проверка состояния обнаруживает это
- Если учётные данные устарели, проверка завершается с ошибкой `auth_error`
- Задержка отражает реальную производительность базы данных, наблюдаемую приложением

## Справочник API

```csharp
public static DepHealthMonitor.Builder AddNpgsqlFromContext<TContext>(
    this DepHealthMonitor.Builder builder,
    string name,
    TContext context,
    bool? critical = null,
    Dictionary<string, string>? labels = null)
    where TContext : DbContext
```

| Параметр | Тип | Описание |
| --- | --- | --- |
| `name` | `string` | Имя зависимости |
| `context` | `TContext` | Экземпляр DbContext (разрешается из DI) |
| `critical` | `bool?` | Флаг критичности (`null` → fallback на переменную окружения) |
| `labels` | `Dictionary<string, string>?` | Пользовательские метки Prometheus |

**Возвращает:** `DepHealthMonitor.Builder` для цепочки вызовов.

**Выбрасывает:** `ConfigurationException`, если строка подключения не найдена
в DbContext.

## Пользовательские метки

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main",
        critical: true,
        labels: new Dictionary<string, string>
        {
            ["role"] = "primary",
            ["shard"] = "eu-west"
        })
);
```

## Несколько DbContext

```csharp
builder.Services.AddDbContext<OrderDbContext>(options =>
    options.UseNpgsql(builder.Configuration["ORDER_DB_URL"]));
builder.Services.AddDbContext<UserDbContext>(options =>
    options.UseNpgsql(builder.Configuration["USER_DB_URL"]));

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<OrderDbContext>("order-db", critical: true)
    .AddNpgsqlFromContext<UserDbContext>("user-db", critical: true)
);
```

## Формат строки подключения

Расширение разбирает стандартные строки подключения Npgsql:

```text
Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass
```

Или PostgreSQL URL-формат (преобразуется Npgsql):

```text
postgresql://user:pass@pg.svc:5432/mydb
```

## Ограничения

- Через интеграцию с Entity Framework поддерживается только PostgreSQL.
  Для MySQL, Redis и других баз данных используйте стандартные методы `Add*()`.
- `DbContext` должен иметь настроенную строку подключения
  (через `UseNpgsql()` или аналогичный метод).
- Проверка состояния использует тот же пул подключений, поэтому настройка
  размера пула влияет на доступность проверки.

## Смотрите также

- [Начало работы](getting-started.ru.md) — установка и базовая настройка
- [Пулы подключений](connection-pools.ru.md) — интеграция пулов для всех баз данных
- [Интеграция с ASP.NET Core](aspnetcore.ru.md) — регистрация в DI и эндпоинты
- [Checkers](checkers.ru.md) — подробности о PostgreSQL checker
- [Конфигурация](configuration.ru.md) — все параметры и значения по умолчанию
- [Устранение неполадок](troubleshooting.ru.md) — распространённые проблемы и решения
