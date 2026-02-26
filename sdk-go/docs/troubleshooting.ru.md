*[English version](troubleshooting.md)*

# Устранение неполадок

Типичные проблемы и решения при использовании Go SDK dephealth.

## Пустые метрики / метрики не экспортируются

**Симптом:** эндпоинт `/metrics` не возвращает метрики `app_dependency_*`.

**Возможные причины:**

1. **Не импортирован пакет чекера.** Фабрики чекеров должны быть
   зарегистрированы через blank import до вызова `dephealth.New()`:

   ```go
   import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
   ```

   Или импортируйте отдельные подпакеты:

   ```go
   import (
       _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
       _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
   )
   ```

2. **Не вызван `Start()`.** Метрики регистрируются и обновляются только
   после вызова `dh.Start(ctx)`. Убедитесь, что `Start()` вызывается и
   не возвращает ошибку.

3. **Неправильный обработчик Prometheus.** Убедитесь, что вы используете
   `promhttp.Handler()` на пути, который ожидает ваш scraper (обычно
   `/metrics`):

   ```go
   http.Handle("/metrics", promhttp.Handler())
   ```

   Если вы используете пользовательский `prometheus.Registry` через
   `WithRegisterer()`, необходимо использовать
   `promhttp.HandlerFor(registry, promhttp.HandlerOpts{})` вместо
   `promhttp.Handler()`.

## Фабрика чекера не зарегистрирована

**Симптом:** `dephealth.New()` возвращает ошибку
`no checker factory registered for type "http"` (или другой тип).

**Причина:** подпакет для запрашиваемого типа чекера не импортирован.

**Решение:** добавьте соответствующий blank import. Например, если вы
используете `dephealth.HTTP()`:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
```

См. [Селективные импорты](selective-imports.ru.md) для полного списка
подпакетов и путей импорта.

## Высокая задержка gRPC (100-500 мс)

**Симптом:** проверки gRPC показывают задержку 100-500 мс, хотя целевой
сервис находится в той же сети.

**Причина:** стандартный DNS-резолвер gRPC выполняет SRV + A запросы.
В Kubernetes с `ndots:5` каждый запрос перебирает все search-домены
перед обращением к реальному FQDN.

**Решение:** используйте полное доменное имя (FQDN) с точкой в конце:

```go
dephealth.GRPC("user-service",
    dephealth.FromParams("user-service.namespace.svc.cluster.local.", "9090"),
    dephealth.Critical(true),
)
```

Точка в конце указывает резолверу пропустить перебор search-доменов.
SDK уже использует `passthrough:///` резолвер, но DNS-запрос
выполняется при установке соединения.

Подробнее в разделе
[DNS Resolution in Kubernetes](../../docs/specification.md#dns-resolution-in-kubernetes)
спецификации.

## Ошибки отказа в соединении

**Симптом:** `app_dependency_status{status="connection_error"}` равен `1`,
в метке detail — `connection_refused`.

**Возможные причины:**

1. **Сервис не запущен** — убедитесь, что целевой сервис работает и
   слушает на ожидаемом хосте и порту.

2. **Неправильный хост или порт** — проверьте URL или параметры,
   переданные в `FromURL()` / `FromParams()`.

3. **Сетевые политики Kubernetes** — если приложение работает в Kubernetes,
   убедитесь, что сетевые политики разрешают трафик от пода с чекером
   к целевому сервису.

4. **Правила файрвола** — в среде без Kubernetes проверьте правила
   файрвола между чекером и целевым сервисом.

## Ошибки таймаута

**Симптом:** `app_dependency_status{status="timeout"}` равен `1`.

**Возможные причины:**

1. **Таймаут по умолчанию слишком мал.** Значение по умолчанию — 5 секунд.
   Для медленных зависимостей (холодные соединения к БД,
   кросс-региональные сервисы) увеличьте его:

   ```go
   // Глобально: для всех зависимостей
   dephealth.WithTimeout(10 * time.Second)

   // Для конкретной зависимости
   dephealth.Postgres("slow-db",
       dephealth.FromURL("postgresql://db.svc:5432/mydb"),
       dephealth.Critical(true),
       dephealth.Timeout(10 * time.Second),
   )
   ```

2. **Сетевые задержки** — проверьте время round-trip до целевого сервиса.
   Используйте гистограмму `app_dependency_latency_seconds` для
   отслеживания реального времени проверок.

3. **Перегрузка сервиса** — целевой сервис может быть слишком загружен,
   чтобы ответить в пределах таймаута.

## Неожиданные ошибки аутентификации

**Симптом:** `app_dependency_status{status="auth_error"}` равен `1`,
хотя учётные данные должны быть верными.

**Возможные причины:**

1. **Учётные данные не переданы или неверны** — проверьте значение
   токена или логина/пароля:

   ```go
   // HTTP: проверьте значение токена
   dephealth.WithHTTPBearerToken(os.Getenv("API_TOKEN"))

   // gRPC: проверьте значение токена
   dephealth.WithGRPCBearerToken(os.Getenv("GRPC_TOKEN"))
   ```

2. **Токен просрочен** — bearer-токены имеют ограниченный срок действия.
   Если токен истекает между перезапусками, обновите переменную окружения
   или источник токена.

3. **Неправильный метод аутентификации** — некоторые сервисы ожидают
   Basic auth вместо Bearer или наоборот. Проверьте документацию
   целевого сервиса.

4. **Учётные данные в URL** — для PostgreSQL, MySQL и AMQP учётные
   данные являются частью URL подключения:

   ```go
   dephealth.Postgres("db",
       dephealth.FromURL("postgresql://user:password@host:5432/dbname"),
       dephealth.Critical(true),
   )
   ```

См. [Аутентификация](authentication.ru.md) для подробностей об опциях
аутентификации каждого типа чекера.

## Дублирование регистрации метрик

**Симптом:** паника при запуске:
`duplicate metrics collector registration attempted`.

**Причина:** два экземпляра `DepHealth` регистрируют метрики в одном
Prometheus registerer (обычно `prometheus.DefaultRegisterer`).

**Решение:** используйте отдельные регистраторы для каждого экземпляра:

```go
reg1 := prometheus.NewRegistry()
reg2 := prometheus.NewRegistry()

dh1, _ := dephealth.New("service-a", "team-a",
    dephealth.WithRegisterer(reg1),
    // ...
)

dh2, _ := dephealth.New("service-b", "team-b",
    dephealth.WithRegisterer(reg2),
    // ...
)
```

На практике большинству приложений нужен только один экземпляр `DepHealth`.

## Пользовательские метки не отображаются

**Симптом:** метки, добавленные через `WithLabel()`, не видны в метриках.

**Возможные причины:**

1. **Недопустимое имя метки.** Имена меток должны соответствовать
   `[a-z_][a-z0-9_]*` и не совпадать с зарезервированными именами.

   Зарезервированные имена: `name`, `group`, `dependency`, `type`, `host`,
   `port`, `critical`.

   ```go
   // Корректно
   dephealth.WithLabel("region", "eu-west")

   // Ошибка — заглавные буквы
   dephealth.WithLabel("Region", "eu-west")

   // Ошибка — зарезервированное имя
   dephealth.WithLabel("type", "primary")
   ```

2. **Несогласованные метки между зависимостями.** Все зависимости в
   экземпляре `DepHealth` должны использовать одинаковый набор имён
   пользовательских меток. Если одна зависимость имеет
   `WithLabel("env", "prod")`, а другая — нет, валидация не пройдёт.

## Health() возвращает пустую map

**Симптом:** `dh.Health()` возвращает пустую map сразу после `Start()`.

**Причина:** первая проверка ещё не завершилась. Между запуском и первой
проверкой проходит интервал, равный периоду проверки (по умолчанию
15 секунд).

**Решение:** используйте `HealthDetails()`. До завершения первой проверки
`HealthDetails()` возвращает записи с `Healthy: nil` и
`Status: "unknown"`, что позволяет отличить «ещё не проверено» от
«неработоспособно»:

```go
details := dh.HealthDetails()
for key, ep := range details {
    if ep.Healthy == nil {
        fmt.Printf("%s: ещё не проверено\n", key)
    } else {
        fmt.Printf("%s: healthy=%v\n", key, *ep.Healthy)
    }
}
```

## Отладочное логирование

Для включения отладочного вывода SDK передайте `*slog.Logger` через
`WithLogger()`:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.WithLogger(logger),
    // ...
)
```

Логируемые сообщения включают:

- Детали регистрации зависимостей
- Результаты проверок (успех/ошибка, задержка, категория статуса)
- Ошибки подключения с полным текстом ошибки
- События регистрации метрик

## См. также

- [Начало работы](getting-started.ru.md) — установка и базовая настройка
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и правила валидации
- [Чекеры](checkers.ru.md) — подробное руководство по всем 8 чекерам
- [Метрики](metrics.ru.md) — справочник по метрикам Prometheus и примеры PromQL
- [Аутентификация](authentication.ru.md) — опции аутентификации для HTTP, gRPC и баз данных
- [Селективные импорты](selective-imports.ru.md) — оптимизация импортов
- [Примеры](examples/) — рабочие примеры кода
