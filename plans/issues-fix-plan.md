# План исправления issues от пользователей

> Создан: 2026-02-09
> Статус: черновик

## Обзор issues

| # | Issue | Затронутые SDK | Критичность |
|---|-------|---------------|-------------|
| 1 | [Разрешить zero dependencies](#issue-1-разрешить-zero-dependencies) | Go, Java (Python/C# — уже работает) | Средняя |
| 2 | [Go module path fix](#issue-2-go-module-path-fix) | Go | Высокая |
| 3 | [Передача credentials из URL](#issue-3-передача-credentials-из-url) | Go, Java, Python, C# | Высокая |

---

## Issue 1: Разрешить zero dependencies

### Проблема

`dephealth.New()` / `Builder.build()` возвращают ошибку при нулевых зависимостях.
Leaf-сервисы (без исходящих зависимостей) — валидный паттерн в микросервисной топологии.

### Анализ по SDK

| SDK | Поведение | Файл:строка |
|-----|-----------|-------------|
| **Go** | Ошибка `"no dependencies configured"` | `sdk-go/dephealth/dephealth.go:48-50` |
| **Java** | `ConfigurationException("At least one dependency must be configured")` | `sdk-java/dephealth-core/.../DepHealth.java:325-328` |
| **Python** | Работает корректно (нет проверки) | `sdk-python/dephealth/api.py` |
| **C#** | Работает корректно (нет проверки) | `sdk-csharp/DepHealth.Core/DepHealth.cs` |

Во всех SDK Scheduler, MetricsExporter и Health() корректно обрабатывают пустой список
зависимостей — циклы просто не выполняются.

### Спецификация

В `spec/` нет явного определения поведения для zero dependencies. Нужно добавить.

### План исправления

#### Фаза 1: Обновление спецификации

**Файл:** `spec/config-contract.md`

Добавить в раздел edge cases:

```markdown
### 9.X. Нулевые зависимости (leaf node)

Конфигурация без зависимостей — валидный случай для leaf-сервисов
(сервисы без исходящих подключений).

**Поведение:**
- Создание экземпляра **без ошибки** — возвращается валидный объект
- `Health()` возвращает пустую коллекцию (`{}` / пустой `Map`)
- `Start()` / `Stop()` — no-op
- Prometheus-метрики `app_dependency_health` и `app_dependency_latency_seconds`
  **не регистрируются**
- `/metrics` endpoint работает — возвращает метрики Go runtime / JVM / etc.
```

#### Фаза 2: Fix Go SDK

**Файл:** `sdk-go/dephealth/dephealth.go`

1. Удалить блок (строки 48-50):

   ```go
   if len(cfg.entries) == 0 {
       return nil, fmt.Errorf("dephealth: no dependencies configured")
   }
   ```

2. Добавить обработку: если `entries` пуст, всё равно создать `DepHealth`
   с пустым Scheduler и MetricsExporter (они уже корректно работают
   с пустыми списками).

**Файл:** `sdk-go/dephealth/dephealth_test.go`

3. Обновить тест `TestNew_NoDependencies` (строка 44-51):
   - Вместо проверки ошибки — проверить успешное создание
   - Проверить `Health()` возвращает `map[string]bool{}`
   - Проверить `Start(ctx)` и `Stop()` не паникуют

#### Фаза 3: Fix Java SDK

**Файл:** `sdk-java/dephealth-core/.../DepHealth.java`

1. Удалить блок (строки 325-328):

   ```java
   if (entries.isEmpty()) {
       throw new ConfigurationException("At least one dependency must be configured");
   }
   ```

2. В `CheckScheduler` (если `threadCount` = 0):

   ```java
   threadCount = Math.max(1, deps.stream()...sum())
   ```

   Изменить на: если `deps` пуст, не создавать `ScheduledExecutorService`.

**Файл:** `sdk-java/dephealth-core/.../DepHealthTest.java`

3. Обновить тест `noDependenciesThrows` → `noDependenciesAllowed`:
   - Проверить `build()` не бросает исключение
   - Проверить `health()` возвращает пустую `Map`
   - Проверить `start()` / `close()` работают без ошибок

#### Фаза 4: Тесты для Python и C#

**Python** (`sdk-python/tests/test_api.py`):

- Добавить `test_zero_dependencies` — проверить, что `DependencyHealth("leaf")`
  создаётся без ошибок, `health()` возвращает `{}`

**C#** (`sdk-csharp/DepHealth.Tests/DepHealthMonitorTests.cs`):

- Добавить `ZeroDependencies_BuildsSuccessfully` — аналогично

#### Фаза 5: Conformance тесты

Добавить сценарий `zero-deps.yaml` в `conformance/scenarios/`:

```yaml
name: zero-dependencies
description: Leaf service with no outgoing dependencies
dependencies: []
expected:
  health: {}
  metrics:
    absent:
      - app_dependency_health
      - app_dependency_latency_seconds
```

---

## Issue 2: Go module path fix

### Проблема

`go.mod` в `sdk-go/` объявляет module path как `github.com/BigKAA/topologymetrics`,
но файл находится в поддиректории. Из-за этого `go get` не работает.

### Решение

**Option A** (рекомендуется issue): сменить module path на
`github.com/BigKAA/topologymetrics/sdk-go`.

Это стандартный подход для Go модулей в поддиректориях монорепо.

### План исправления

#### Фаза 1: Обновление module path

**Основные изменения:**

1. `sdk-go/go.mod` — сменить module declaration:

   ```diff
   -module github.com/BigKAA/topologymetrics
   +module github.com/BigKAA/topologymetrics/sdk-go
   ```

2. **28 Go-файлов в `sdk-go/dephealth/`** — обновить внутренние импорты:

   ```diff
   -"github.com/BigKAA/topologymetrics/dephealth"
   +"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
   ```

   ```diff
   -"github.com/BigKAA/topologymetrics/dephealth/checks"
   +"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
   ```

   ```diff
   -"github.com/BigKAA/topologymetrics/dephealth/contrib/sqldb"
   +"github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
   ```

   ```diff
   -"github.com/BigKAA/topologymetrics/dephealth/contrib/redispool"
   +"github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
   ```

3. `sdk-go/Makefile` (строка 92) — обновить `-local`:

   ```diff
   -goimports -w -local github.com/BigKAA/topologymetrics . && \
   +goimports -w -local github.com/BigKAA/topologymetrics/sdk-go . && \
   ```

#### Фаза 2: Обновление тестовых сервисов

4. `test-services/go-service/go.mod`:
   - Обновить `replace` directive:

     ```diff
     -replace github.com/BigKAA/topologymetrics => ../../sdk-go
     +replace github.com/BigKAA/topologymetrics/sdk-go => ../../sdk-go
     ```

   - Обновить `require`:

     ```diff
     -github.com/BigKAA/topologymetrics v0.0.0-00010101000000-000000000000
     +github.com/BigKAA/topologymetrics/sdk-go v0.0.0-00010101000000-000000000000
     ```

5. `test-services/go-service/main.go` — обновить 4 импорта

6. `conformance/test-service/go.mod` — аналогично п.4

7. `conformance/test-service/main.go` — аналогично п.5

#### Фаза 3: Обновление документации

Файлы для обновления:

| Файл | Количество вхождений |
|------|---------------------|
| `sdk-go/README.md` | 2 |
| `docs/quickstart/go.md` | 7 |
| `docs/migration/go.md` | 8 |
| `docs/comparison.md` | 1 |
| `sdk-architecture.md` | 2 |
| `README.md` (root) | 2 |
| `CHANGELOG.md` | ссылки |

Шаблон замены:

- Установка: `go get github.com/BigKAA/topologymetrics/sdk-go/dephealth@latest`
- Импорт: `import "github.com/BigKAA/topologymetrics/sdk-go/dephealth"`

#### Фаза 4: Git tags и публикация

1. Удалить старый невалидный тег `sdk-go/v0.2.1` (если он указывает на старый module path)
2. После коммита — создать новый тег `sdk-go/v0.3.0` (это breaking change!)
3. Проверка:

   ```bash
   mkdir /tmp/test-sdk && cd /tmp/test-sdk
   go mod init test
   go get github.com/BigKAA/topologymetrics/sdk-go/dephealth@v0.3.0
   ```

#### Фаза 5: Проверка и тестирование

```bash
cd sdk-go && make test && make lint
cd test-services/go-service && go build .
cd conformance/test-service && go build .
```

### Важные замечания

- Это **breaking change** — все пользователи текущего Go SDK должны обновить import paths
- Следует выпустить как **v0.3.0** (минорная версия, т.к. < v1.0)
- После публикации на Go proxy — старый module path перестанет работать
- Нужен раздел в `docs/migration/` для миграции с v0.2.x на v0.3.0

---

## Issue 3: Передача credentials из URL

### Проблема

Все SDK парсят URL, но **явно отбрасывают userinfo** (user:pass@).
Checkers используют дефолтные credentials вместо указанных в URL.

### Соответствие спецификации

Спецификация **требует** передачу credentials:

- `spec/config-contract.md:79`: "Credentials | Userinfo (`user:pass@`) |
  Аутентификация при автономной проверке"
- `spec/check-behavior.md:301`: "Аутентификация: username/password из URL
  или конфигурации"

**Ни один SDK не реализует это требование спецификации.**

### Анализ по SDK

| SDK | Парсер | Endpoint | Checkers |
|-----|--------|----------|----------|
| **Go** | `url.Parse()` — userinfo в `u.User`, но не сохраняется | Нет полей credentials | Hardcoded defaults |
| **Java** | Явно удаляет: `rest.substring(atSign + 1)` | Нет полей credentials | Только через builder |
| **Python** | Явно удаляет: `raw_netloc.split("@", 1)[1]` | Нет полей credentials | Hardcoded defaults |
| **C#** | Явно удаляет: `rest[(atSign + 1)..]` | Нет полей credentials | Только через builder |

### Затронутые checkers

| Checker | Go | Java | Python | C# |
|---------|------|------|--------|------|
| Postgres | Hardcoded `user=root` | Builder only | Дефолт | Дефолт |
| MySQL | Hardcoded `user=root` | Builder only | Дефолт OS | Дефолт |
| Redis | Builder only | Builder only | Builder only | Builder only |
| AMQP | Hardcoded `guest:guest` | Builder / URL | Дефолт | Builder only |
| Kafka | N/A (нет auth) | N/A | N/A | N/A |

### План исправления

#### Фаза 1: Обновление спецификации

**Файл:** `spec/config-contract.md`

Уточнить раздел 2.2 (извлечение данных из URL):

```markdown
| Данные | Источник | Обязательное | Используется в |
| --- | --- | --- | --- |
| Credentials | Userinfo (`user:pass@`) | Нет | Автономная проверка |
| Database | Path компонент | Нет | Postgres, MySQL, Redis |

Если credentials указаны в URL, они ДОЛЖНЫ быть переданы соответствующему
checker для автономной проверки. Если credentials переданы и через URL,
и через отдельные параметры — отдельные параметры имеют приоритет.
```

**Файл:** `spec/check-behavior.md`

Для каждого checker добавить раздел "Credentials":

```markdown
**Приоритет credentials (для всех checkers):**
1. Явно заданные параметры (username/password через builder/options)
2. Credentials из URL (userinfo часть)
3. Дефолтные значения (для AMQP: guest/guest)
```

#### Фаза 2: Расширение структуры Endpoint / ParsedConnection

##### Go SDK

**Файл:** `sdk-go/dephealth/dependency.go`

```go
type Endpoint struct {
    Host     string
    Port     string
    Labels   map[string]string
    // Новые поля:
    Username string // из userinfo URL (может быть пустым)
    Password string // из userinfo URL (может быть пустым)
    Database string // из path компонента URL (может быть пустым)
}
```

**Файл:** `sdk-go/dephealth/parser.go`

В `ParseURL()` — сохранить userinfo и path:

```go
// Сохраняем credentials из URL
var username, password string
if u.User != nil {
    username = u.User.Username()
    password, _ = u.User.Password()
}

// Сохраняем database из path
database := strings.TrimPrefix(u.Path, "/")

return []ParsedConnection{{
    Host:     host,
    Port:     port,
    ConnType: connType,
    Username: username,
    Password: password,
    Database: database,
}}, nil
```

##### Java SDK

**Файл:** `sdk-java/dephealth-core/.../parser/ConfigParser.java`

Не удалять userinfo, а сохранять:

```java
String username = null;
String password = null;
int atSign = rest.indexOf('@');
if (atSign >= 0) {
    String userInfo = rest.substring(0, atSign);
    rest = rest.substring(atSign + 1);
    int colon = userInfo.indexOf(':');
    if (colon >= 0) {
        username = URLDecoder.decode(userInfo.substring(0, colon), UTF_8);
        password = URLDecoder.decode(userInfo.substring(colon + 1), UTF_8);
    } else {
        username = URLDecoder.decode(userInfo, UTF_8);
    }
}
```

Добавить в `Endpoint` поля `username`, `password`, `database`.

##### Python SDK

**Файл:** `sdk-python/dephealth/parser.py`

Сохранять userinfo:

```python
username = parsed.username  # urllib автоматически парсит
password = parsed.password
database = parsed.path.lstrip("/") if parsed.path else None
```

Добавить в `Endpoint` поля `username`, `password`, `database`.

##### C# SDK

**Файл:** `sdk-csharp/DepHealth.Core/ConfigParser.cs`

Аналогично Java — не удалять userinfo, а сохранять.
Добавить в `Endpoint` поля `Username`, `Password`, `Database`.

#### Фаза 3: Обновление Checkers

Для каждого checker (Postgres, MySQL, Redis, AMQP) во всех SDK:

**Приоритет credentials:**

```
explicit_option > url_credential > default_value
```

##### Пример: Go Postgres Checker

**Файл:** `sdk-go/dephealth/checks/postgres.go`

```go
func (c *PostgresChecker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
    dsn := c.dsn
    if dsn == "" {
        // Используем credentials из endpoint (из URL), если не заданы через опции
        username := c.username  // явная опция
        if username == "" {
            username = endpoint.Username  // из URL
        }
        password := c.password
        if password == "" {
            password = endpoint.Password
        }
        database := c.database
        if database == "" {
            database = endpoint.Database
        }
        if database == "" {
            database = "postgres"  // дефолт
        }

        u := &url.URL{
            Scheme: "postgres",
            Host:   net.JoinHostPort(endpoint.Host, endpoint.Port),
            Path:   "/" + database,
        }
        if username != "" {
            if password != "" {
                u.User = url.UserPassword(username, password)
            } else {
                u.User = url.User(username)
            }
        }
        dsn = u.String()
    }
    // ...
}
```

Аналогичные изменения для MySQL, Redis, AMQP checkers во всех SDK.

#### Фаза 4: Тестирование

Для каждого SDK добавить тесты:

1. **Parser tests:** URL с credentials корректно парсится,
   username/password/database сохраняются в Endpoint
2. **Checker tests:** credentials из Endpoint используются в connection string
3. **Priority tests:** явная опция > URL > дефолт

##### Conformance тесты

Добавить сценарий `url-credentials.yaml`:

```yaml
name: url-credentials
description: Credentials from URL must be used for standalone checks
dependencies:
  - name: pg-with-creds
    url: "postgres://testuser:testpass@postgresql:5432/testdb"
    type: postgres
expected:
  health:
    pg-with-creds: true
  # Сервис postgresql должен быть настроен с user=testuser, pass=testpass, db=testdb
```

#### Фаза 5: Безопасность

**ВАЖНО:** Credentials НЕ должны:

- Попадать в метрики Prometheus (label values)
- Логироваться в открытом виде (маскировать в логах)
- Быть доступны через `Health()` или другие публичные API

Проверить, что маскировка credentials работает в логах:

```
dephealth: check ok  dependency=pg  url=postgres://***:***@host:5432/db
```

---

## Порядок выполнения

### Рекомендуемый порядок

1. **Issue 1 (zero deps)** — простое изменение, минимальный риск
2. **Issue 3 (credentials)** — критичный баг, требует изменений во всех SDK
3. **Issue 2 (module path)** — breaking change, требует новую версию Go SDK

### Версионирование

| Issue | Версия | Тип изменения |
|-------|--------|--------------|
| Issue 1 (zero deps) | v0.3.0 | feature (minor) |
| Issue 3 (credentials) | v0.3.0 | fix (не breaking, но значительное) |
| Issue 2 (module path) | v0.3.0 Go only | breaking (Go import paths) |

Рекомендуется объединить все три исправления в один релиз **v0.3.0**,
т.к. Issue 2 уже является breaking change для Go SDK.

---

## Оценка объёма работ

| Фаза | Файлов | Описание |
|------|--------|----------|
| Issue 1 | ~10 | Spec + Go + Java + тесты Python/C# |
| Issue 2 | ~50 | Go module path + imports + docs |
| Issue 3 | ~40 | Spec + Endpoint + Parser + Checkers * 4 SDK |
| **Итого** | ~100 | |
