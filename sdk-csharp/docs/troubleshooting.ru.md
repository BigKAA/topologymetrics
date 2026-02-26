*[English version](troubleshooting.md)*

# Устранение неполадок

Типичные проблемы и решения при использовании C# SDK dephealth.

## Пустые метрики / Метрики не экспортируются

**Симптом:** Эндпоинт `/metrics` не содержит метрик `app_dependency_*`.

**Возможные причины:**

1. **`Start()` не вызван.** Метрики регистрируются и обновляются только
   после вызова `Start()`. Убедитесь, что `Start()` вызывается без ошибок.

   Для ASP.NET Core: проверьте, что `DepHealth.AspNetCore` зарегистрирован
   через `AddDepHealth` — он вызывает `Start()` автоматически через
   `DepHealthHostedService`.

2. **Неправильный эндпоинт Prometheus.** Метрики экспортируются на `/metrics`
   через `prometheus-net.AspNetCore`. Убедитесь, что пакет установлен и
   вызван `app.MapMetrics()`:

   ```csharp
   using Prometheus;

   var app = builder.Build();
   app.UseHttpMetrics();
   app.MapDepHealthEndpoints();
   app.MapMetrics();  // открывает /metrics
   app.Run();
   ```

3. **Registry не подключён.** Для программного API убедитесь, что один и тот
   же `CollectorRegistry` передаётся в builder через `WithRegistry()`, или
   используйте реестр по умолчанию `Metrics.DefaultRegistry`:

   ```csharp
   var registry = Metrics.NewCustomRegistry();
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithRegistry(registry)
       // ...
       .Build();
   dh.Start();

   // Тот же registry для обработчика /metrics
   var metricFactory = Metrics.WithCustomRegistry(registry);
   ```

## Все зависимости показывают 0 (unhealthy)

**Симптом:** `app_dependency_health` равен `0` для всех зависимостей.

**Возможные причины:**

1. **Сетевая доступность** — убедитесь, что целевые сервисы доступны из
   контейнера/пода сервиса.

2. **DNS-разрешение** — проверьте, что имена сервисов разрешаются корректно.

3. **Неправильный URL/host/port** — перепроверьте значения конфигурации.

4. **Тайм-аут слишком мал** — по умолчанию 5 сек. Увеличьте для медленных
   зависимостей:

   ```csharp
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithCheckTimeout(TimeSpan.FromSeconds(10))
       .AddPostgres("slow-db",
           Environment.GetEnvironmentVariable("DATABASE_URL")!,
           critical: true)
       .Build();
   ```

5. **Отладочное логирование** — включите отладку SDK через `appsettings.json`:

   ```json
   {
     "Logging": {
       "LogLevel": {
         "DepHealth": "Debug"
       }
     }
   }
   ```

   Или передайте логгер напрямую:

   ```csharp
   using Microsoft.Extensions.Logging;

   var loggerFactory = LoggerFactory.Create(b => b.AddConsole().SetMinimumLevel(LogLevel.Debug));
   var logger = loggerFactory.CreateLogger("DepHealth");

   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithLogger(logger)
       // ...
       .Build();
   ```

## Высокая латентность проверок PostgreSQL/MySQL

**Симптом:** `app_dependency_latency_seconds` показывает высокие значения
(100 мс+) для проверок баз данных.

**Причина:** Standalone-режим создаёт новое соединение при каждой проверке.
Это включает TCP-рукопожатие, согласование TLS и аутентификацию.

**Решение:** Используйте интеграцию с Entity Framework или pool-режим с
существующим `NpgsqlDataSource`:

```csharp
// Вместо standalone-режима
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("db-primary",
        "postgresql://user:pass@pg.svc:5432/mydb",
        critical: true)
);

// Используйте существующий NpgsqlDataSource (предпочтительно)
var dataSource = NpgsqlDataSource.Create(connectionString);
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddCustom("db-primary", DependencyType.Postgres,
        host: "pg.svc", port: "5432",
        checker: new PostgresChecker(dataSource),
        critical: true)
);

// Или используйте интеграцию с Entity Framework
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext("db-primary", dbContext, critical: true)
);
```

Подробнее — в [Пулы соединений](connection-pools.ru.md).

## gRPC: ошибка DeadlineExceeded

**Симптом:** gRPC-проверки завершаются по тайм-ауту или показывают
высокую латентность.

**Возможные причины:**

1. **gRPC-сервис недоступен** по указанному адресу.

2. **Сервис не реализует** `grpc.health.v1.Health/Check` — протокол
   gRPC Health Checking должен быть включён на целевом сервисе.

3. **Используйте `host` + `port`**, а не `url` для gRPC:

   ```csharp
   builder.Services.AddDepHealth("my-service", "my-team", dh => dh
       .AddGrpc("user-service",
           host: "user.svc",
           port: "9090",
           critical: true)
   );
   ```

4. **Несовпадение TLS** — если сервис использует TLS, установите `tlsEnabled: true`:

   ```csharp
   .AddGrpc("user-service",
       host: "user.svc",
       port: "443",
       tlsEnabled: true,
       critical: true)
   ```

5. **DNS-разрешение в Kubernetes** — используйте FQDN с точкой на конце:

   ```csharp
   .AddGrpc("user-service",
       host: "user-service.namespace.svc.cluster.local.",
       port: "9090",
       critical: true)
   ```

## Ошибки Connection Refused

**Симптом:** `app_dependency_status{status="connection_error"}` равен `1`.

**Возможные причины:**

1. **Сервис не запущен** — убедитесь, что целевой сервис работает и
   слушает на ожидаемом хосте и порту.

2. **Неправильный host или port** — перепроверьте значения URL или host/port.

3. **Network policies Kubernetes** — убедитесь, что трафик разрешён от
   пода чекера к целевому сервису.

4. **Правила файрвола** — для не-Kubernetes окружений проверьте файрвол.

## Ошибки тайм-аута

**Симптом:** `app_dependency_status{status="timeout"}` равен `1`.

**Возможные причины:**

1. **Тайм-аут по умолчанию слишком мал.** По умолчанию 5 сек. Увеличьте
   глобально — в текущей версии SDK переопределение на уровне отдельной
   зависимости недоступно, используйте глобальную опцию:

   ```csharp
   // Глобальный тайм-аут для всех зависимостей
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithCheckTimeout(TimeSpan.FromSeconds(10))
       // ...
       .Build();
   ```

2. **Сетевая латентность** — проверьте время отклика целевого сервиса.

3. **Перегрузка целевого сервиса** — сервис может быть слишком медленным.

## Неожиданные ошибки аутентификации

**Симптом:** `app_dependency_status{status="auth_error"}` равен `1`, хотя
учётные данные должны быть верными.

**Возможные причины:**

1. **Учётные данные не установлены или неверны**:

   ```csharp
   builder.AddHttp("payment-api",
       url: "https://payment.svc:8443",
       critical: true,
       bearerToken: Environment.GetEnvironmentVariable("API_TOKEN"))

   builder.AddGrpc("payments-grpc",
       host: "payments.svc",
       port: "9090",
       critical: true,
       bearerToken: Environment.GetEnvironmentVariable("GRPC_TOKEN"))
   ```

2. **Токен истёк** — bearer-токены имеют ограниченный срок действия.

3. **Неправильный метод аутентификации** — некоторые сервисы ожидают
   Basic auth вместо Bearer.

4. **Учётные данные БД** — для PostgreSQL, MySQL и AMQP проверьте
   корректность в URL:

   ```csharp
   .AddPostgres("db",
       url: "postgresql://user:password@host:5432/dbname",
       critical: true)
   ```

Подробнее — в [Аутентификация](authentication.ru.md).

## AMQP: ошибка подключения к RabbitMQ

**Симптом:** AMQP-чекер не может подключиться.

**Важно**: путь `/` в URL означает vhost `/` (не пустой).

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddAmqp("rabbitmq",
        url: "amqp://rabbitmq.svc:5672/",
        critical: false)
);
```

Для явного указания учётных данных используйте формат
`amqp://user:pass@host:port/vhost`:

```csharp
// Явные учётные данные и vhost в URL
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/my-vhost",
    critical: false)
```

## LDAP: ошибки конфигурации

**Симптом:** LDAP-чекер выбрасывает `ConfigurationException` при старте.

**Типичные причины:**

1. **`SimpleBind` без учётных данных:**

   ```csharp
   // Неправильно — нет bindDN и bindPassword
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.SimpleBind,
       critical: false)

   // Правильно
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.SimpleBind,
       bindDN: "cn=monitor,dc=corp,dc=com",
       bindPassword: "secret",
       critical: false)
   ```

2. **`Search` без baseDN:**

   ```csharp
   // Неправильно — нет baseDN
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.Search,
       critical: false)

   // Правильно
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.Search,
       baseDN: "dc=example,dc=com",
       critical: false)
   ```

3. **startTLS с `ldaps://`** — несовместимы:

   ```csharp
   // Неправильно — нельзя использовать оба
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "636",
       useTls: true,
       startTls: true,  // несовместимо с useTls
       critical: false)

   // Правильно — используйте одно из двух
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "636",
       useTls: true,    // неявный TLS (ldaps://)
       critical: false)
   // ИЛИ
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       startTls: true,  // обновление до TLS через StartTLS
       critical: false)
   ```

## Произвольные метки не отображаются

**Симптом:** Метки, добавленные через словарь `labels`, не видны в метриках.

**Возможные причины:**

1. **Недопустимое имя метки.** Должно соответствовать `[a-zA-Z_][a-zA-Z0-9_]*`
   и не быть зарезервированным.

   Зарезервированные: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

2. **Несогласованные метки между зависимостями.** При использовании
   произвольных меток все эндпоинты должны использовать одинаковые
   имена меток. SDK собирает все имена меток из всех зависимостей и
   применяет их ко всем метрикам.

## Health() возвращает пустой словарь

**Симптом:** `monitor.Health()` возвращает пустой словарь сразу после `Start()`.

**Причина:** Первая проверка ещё не завершилась. Есть начальная задержка
(по умолчанию 0 сек, настраивается через `WithInitialDelay`) перед первой
проверкой. Даже без задержки первая проверка выполняется асинхронно.

**Решение:** Используйте `HealthDetails()`. До завершения первой проверки
он возвращает записи с `Healthy = null` и `Status = "unknown"`:

```csharp
var details = monitor.HealthDetails();
foreach (var (key, ep) in details)
{
    if (ep.Healthy is null)
    {
        Console.WriteLine($"{key}: ещё не проверен");
    }
    else
    {
        Console.WriteLine($"{key}: healthy={ep.Healthy}");
    }
}
```

## ASP.NET Core: метрики не на /metrics

**Проверьте:**

1. Пакет `prometheus-net.AspNetCore` установлен
2. `app.MapMetrics()` вызван после `app.Build()`
3. `AddDepHealth` зарегистрирован в `builder.Services`
4. Middleware приложения не фильтрует запросы к `/metrics`

## См. также

- [Начало работы](getting-started.ru.md) — установка и базовая настройка
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и правила валидации
- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 чекерам
- [Метрики](metrics.ru.md) — справочник метрик Prometheus и примеры PromQL
- [Аутентификация](authentication.ru.md) — опции аутентификации для HTTP, gRPC и баз данных
- [Пулы соединений](connection-pools.ru.md) — интеграция с NpgsqlDataSource и IConnectionMultiplexer
- [ASP.NET Core интеграция](aspnetcore.ru.md) — hosted service и эндпоинты здоровья
- [Entity Framework интеграция](entity-framework.ru.md) — проверки на основе DbContext
