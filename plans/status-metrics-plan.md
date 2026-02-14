# Plan: Добавление метрик `app_dependency_status` и `app_dependency_status_detail`

## Context

Проект dephealth — SDK для мониторинга зависимостей микросервисов (Go, Java, Python, C#).
Сейчас при `app_dependency_health == 0` невозможно определить **причину** недоступности
зависимости из метрик — информация есть только в логах. Это затрудняет диагностику
в Grafana, точный алертинг и автоматизацию реакции на инциденты.

Решение: добавить две новых Prometheus-метрики, не меняя существующие.

## Что добавляется (подробное описание для документации)

### Метрика 1: `app_dependency_status`

**Тип**: Gauge (enum-паттерн)
**HELP**: `Category of the last check result`
**Поведение**: для каждого endpoint всегда экспортируются **все 8 значений** метки `status`. Ровно одно из них = 1, остальные 7 = 0. Это исключает series churn при смене состояния.

**Значения метки `status`**:

| Значение | Описание | Типичные ситуации |
|----------|----------|-------------------|
| `ok` | Проверка успешна, зависимость доступна | HTTP 2xx, gRPC SERVING, TCP connected, SQL SELECT 1 OK, Redis PONG |
| `timeout` | Превышен таймаут проверки | Connection timeout, query timeout, gRPC DEADLINE_EXCEEDED, context deadline exceeded |
| `connection_error` | Невозможно установить TCP-соединение | Connection refused (RST), host unreachable, network unreachable, port not listening |
| `dns_error` | Ошибка разрешения DNS-имени | Hostname not found, DNS lookup failure, NXDOMAIN |
| `auth_error` | Ошибка аутентификации/авторизации | Wrong DB credentials, Redis NOAUTH/WRONGPASS, AMQP 403 Access Refused |
| `tls_error` | Ошибка TLS/SSL | Certificate validation failed, TLS handshake error, expired certificate |
| `unhealthy` | Сервис ответил, но сообщает о нездоровом состоянии | HTTP 4xx/5xx, gRPC NOT_SERVING, Kafka no brokers, Redis non-PONG, AMQP connection not open |
| `error` | Прочие неклассифицированные ошибки | Unexpected exceptions, panics, pool exhaustion, query syntax error |

**Метки**: `name`, `dependency`, `type`, `host`, `port`, `critical`, [custom labels...], `status`

**Пример вывода** (endpoint pg.svc:5432 доступен):
```text
# HELP app_dependency_status Category of the last check result
# TYPE app_dependency_status gauge
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="ok"} 1
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="timeout"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="connection_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="dns_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="auth_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="tls_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="unhealthy"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="error"} 0
```

### Метрика 2: `app_dependency_status_detail`

**Тип**: Gauge (info-паттерн)
**HELP**: `Detailed reason of the last check result`
**Поведение**: одна серия на endpoint, значение всегда = 1. Метка `detail` содержит конкретную причину. При смене причины — старая серия удаляется, новая создаётся (допустимый series churn).

**Метки**: `name`, `dependency`, `type`, `host`, `port`, `critical`, [custom labels...], `detail`

**Значения `detail` по типам чекеров**:

| Тип чекера | Возможные значения detail |
|------------|--------------------------|
| HTTP | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `http_NNN` (конкретный HTTP-код: `http_404`, `http_503` и т.д.), `error` |
| gRPC | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `grpc_not_serving`, `grpc_unknown`, `error` |
| TCP | `ok`, `timeout`, `connection_refused`, `dns_error`, `error` |
| PostgreSQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| MySQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| Redis | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `unhealthy`, `error` |
| AMQP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Kafka | `ok`, `timeout`, `connection_refused`, `dns_error`, `no_brokers`, `error` |

**Маппинг detail → status (категория)**:

| detail | status |
|--------|--------|
| `ok` | `ok` |
| `timeout` | `timeout` |
| `connection_refused`, `network_unreachable`, `host_unreachable` | `connection_error` |
| `dns_error` | `dns_error` |
| `auth_error` | `auth_error` |
| `tls_error` | `tls_error` |
| `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers` | `unhealthy` |
| `error`, `pool_exhausted`, `query_error` | `error` |

**Пример вывода** (payment-api вернул HTTP 503):
```text
# HELP app_dependency_status_detail Detailed reason of the last check result
# TYPE app_dependency_status_detail gauge
app_dependency_status_detail{name="order-api",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail="http_503"} 1
```

### Общие свойства обеих метрик

- **Initial value**: до первой проверки обе метрики **не экспортируются** (аналогично `app_dependency_health`)
- **Обратная совместимость**: существующие `app_dependency_health` и `app_dependency_latency_seconds` **не меняются**
- **Порядок меток**: `name`, `dependency`, `type`, `host`, `port`, `critical`, [custom alphabetical], `status`/`detail`
- **Влияние на хранение**: +9 серий на endpoint (8 от status + 1 от detail)

### PromQL-примеры

```promql
# Текущая категория состояния всех зависимостей
app_dependency_status == 1

# Все зависимости с таймаутом
app_dependency_status{status="timeout"} == 1

# Все зависимости с ошибкой аутентификации (alert-friendly)
app_dependency_status{status="auth_error"} == 1

# Детальная причина по конкретной зависимости
app_dependency_status_detail{name="order-api",dependency="payment-api"}

# Все HTTP 503 ошибки по всему кластеру
app_dependency_status_detail{detail="http_503"}

# Flapping detection (без series churn — работает корректно!)
changes(app_dependency_status{status="ok"}[15m]) > 4

# Распределение по статусам
count by (status) (app_dependency_status == 1)

# Корреляция: unhealthy зависимости с деталями через join
app_dependency_status{status="unhealthy"} == 1
  AND on (name, dependency, type, host, port)
app_dependency_status_detail
```

---

## Архитектурное решение: ClassifiedError interface (non-breaking)

Интерфейс `HealthChecker` **не меняется** ни в одном SDK. Вместо этого:

1. Вводится интерфейс `ClassifiedError` (Go: interface, Python: property на CheckError, Java/C#: базовый класс) — ошибка несёт `StatusCategory()` и `StatusDetail()`
2. Чекеры возвращают/бросают ошибки, реализующие этот интерфейс
3. Планировщик (scheduler) классифицирует ошибку: сначала проверяет `ClassifiedError`, затем sentinel errors, затем платформенные ошибки (`net.DNSError`, `tls.CertificateVerificationError` и т.д.), fallback → `"error"/"error"`
4. Старые/пользовательские чекеры, не реализующие интерфейс, получают классификацию через fallback — полная обратная совместимость

**Текущие типизированные ошибки в SDK**:

| SDK | Timeout | Connection Refused | Unhealthy | DNS | TLS | Auth |
|-----|---------|-------------------|-----------|-----|-----|------|
| Go | `ErrTimeout` | `ErrConnectionRefused` | `ErrUnhealthy` | — | — | — |
| Python | `CheckTimeoutError` | `CheckConnectionRefusedError` | `UnhealthyError` | — | — | — |
| Java | — | — | — | — | — | — |
| C# | `CheckTimeoutException` (unused) | `ConnectionRefusedException` (unused) | `UnhealthyException` | — | — | — |

Все SDK будут дополнены недостающими типами: DNS, TLS, Auth.

---

## Фазы реализации

### Фаза 1: Обновление спецификации ✅→☐

**Файлы**:
- `spec/metric-contract.md` — новые секции 8 и 9 (status + detail метрики)
- `spec/metric-contract.ru.md` — русский перевод
- `spec/check-behavior.md` — новая секция 6.2 "Error Classification" (категории, detail values, fallback rules)
- `spec/check-behavior.ru.md` — русский перевод

**Содержание**: полное описание обеих метрик, все значения, HELP text, примеры вывода, PromQL queries, таблица маппинга detail→status по типам чекеров.

---

### Фаза 2: Go SDK — ядро (инфраструктура + scheduler + metrics) ✅→☐

**Создать**:
- `sdk-go/dephealth/check_result.go` — тип `CheckResult{Category, Detail}`, константы `StatusOK`..`StatusError`, интерфейс `ClassifiedError`, тип `StatusError` (implements `error` + `ClassifiedError`)
- `sdk-go/dephealth/classify.go` — функция `classifyError(err error) CheckResult` с цепочкой: ClassifiedError → sentinel errors → `context.DeadlineExceeded` → `*net.DNSError` → `*net.OpError` (connection refused) → `*tls.CertificateVerificationError` → fallback
- `sdk-go/dephealth/check_result_test.go`
- `sdk-go/dephealth/classify_test.go`

**Изменить**:
- `sdk-go/dephealth/metrics.go`:
  - Новые поля: `status *prometheus.GaugeVec`, `statusDetail *prometheus.GaugeVec`, `prevDetails map[string]string` + `sync.Mutex`
  - `NewMetricsExporter()`: регистрация двух новых GaugeVec (label names + "status" / "detail")
  - Новые методы: `SetStatus(dep, ep, category)` — итерация по 8 значениям, одно=1, остальные=0; `SetStatusDetail(dep, ep, detail)` — удаление старой серии при смене detail, установка новой
  - `DeleteMetrics()`: удаление серий status/detail
- `sdk-go/dephealth/scheduler.go`:
  - В `executeCheck()`: после проверки вызвать `classifyError(checkErr)`, затем `SetStatus()` и `SetStatusDetail()`
- `sdk-go/dephealth/metrics_test.go` — тесты: все 8 серий enum, exactly-one-is-1, detail cleanup
- `sdk-go/dephealth/scheduler_test.go` — тесты: status/detail при healthy/unhealthy чекерах

---

### Фаза 3: Go SDK — обновление чекеров ✅→☐

**Изменить** (каждый чекер + его тест):
- `sdk-go/dephealth/checks/http.go` — non-2xx → `StatusError{Category:"unhealthy", Detail:fmt.Sprintf("http_%d",code)}`
- `sdk-go/dephealth/checks/grpc.go` — NOT_SERVING → `StatusError{Category:"unhealthy", Detail:"grpc_not_serving"}`
- `sdk-go/dephealth/checks/tcp.go` — wrapping dial errors с DNS/timeout detection
- `sdk-go/dephealth/checks/postgres.go` — auth errors через `pgconn.PgError` код `28P01`/`28000`
- `sdk-go/dephealth/checks/mysql.go` — auth errors через MySQL error 1045
- `sdk-go/dephealth/checks/redis.go` — `NOAUTH`/`WRONGPASS` → auth_error; non-PONG → unhealthy
- `sdk-go/dephealth/checks/amqp.go` — AMQP 403 → auth_error; TLS errors
- `sdk-go/dephealth/checks/kafka.go` — no brokers → `StatusError{Category:"unhealthy", Detail:"no_brokers"}`

Тесты: проверка `errors.As(err, &ClassifiedError)` с корректными category/detail.

---

### Фаза 4: Python SDK ✅→☐

**Создать**:
- `sdk-python/dephealth/check_result.py` — `CheckResult` frozen dataclass, `classify_error()` функция

**Изменить**:
- `sdk-python/dephealth/checker.py` — добавить `status_category`/`detail` properties к `CheckError`; новые типы: `CheckDnsError`, `CheckAuthError`, `CheckTlsError`
- `sdk-python/dephealth/metrics.py` — новые `Gauge` для status/detail; `set_status()`, `set_status_detail()` с delete-on-change
- `sdk-python/dephealth/scheduler.py` — classify error → set_status/set_status_detail
- Все 8 чекеров в `sdk-python/dephealth/checks/` — установка detail на исключениях (HTTP: `detail=f"http_{resp.status}"` и т.д.); новые типы ошибок (dns, auth, tls)
- `sdk-python/dephealth/__init__.py` — экспорт новых типов
- Тесты: `test_metrics.py`, `test_scheduler.py`, `test_check_result.py`

---

### Фаза 5: Java SDK ✅→☐

**Создать**:
- `CheckResult.java` — record class
- `StatusCategory.java` — string constants
- `CheckException.java` — базовый exception с `statusCategory()`/`detail()`
- Подклассы: `CheckTimeoutException`, `CheckConnectionException`, `CheckDnsException`, `CheckAuthException`, `CheckTlsException`, `UnhealthyException` (в пакете `checks/`)
- `ErrorClassifier.java` — `classify(Exception) → CheckResult` (instanceof chain + platform detection: `UnknownHostException` → dns_error, `SSLException` → tls_error и т.д.)

**Изменить**:
- `MetricsExporter.java` — новые ConcurrentHashMap для status gauges (8 AtomicReference per endpoint) и detail gauge; `setStatus()`, `setStatusDetail()` методы через Micrometer
- `CheckScheduler.java` — classify error в `runCheck()`, вызов setStatus/setStatusDetail
- Все 8 чекеров — бросать typed `CheckException` вместо generic `Exception`
- Тесты: `MetricsExporterTest`, `CheckSchedulerTest`, `ErrorClassifierTest`

---

### Фаза 6: C# SDK ✅→☐

**Создать**:
- `CheckResult.cs` — record struct
- `StatusCategory.cs` — static string constants
- `ErrorClassifier.cs` — `Classify(Exception) → CheckResult`
- `Exceptions/CheckDnsException.cs`, `CheckAuthException.cs`, `CheckTlsException.cs`

**Изменить**:
- `Exceptions/DepHealthException.cs` — добавить `virtual StatusCategory`/`Detail` properties
- `Exceptions/CheckTimeoutException.cs` — override StatusCategory → "timeout"
- `Exceptions/ConnectionRefusedException.cs` — override → "connection_error"
- `Exceptions/UnhealthyException.cs` — конструктор с detail; override → "unhealthy"
- `PrometheusExporter.cs` — новые `Gauge` для status/detail; `SetStatus()`, `SetStatusDetail()` через prometheus-net
- `CheckScheduler.cs` — classify error в `RunCheck()`, SetStatus/SetStatusDetail
- Все 8 чекеров — бросать typed exceptions с detail
- Тесты: `PrometheusExporterTests`, `ErrorClassifierTests`

---

### Фаза 7: Документация, версии, changelog ✅

**Файлы**:
- `docs/` — обновить quickstart, API docs (EN + RU)
- `docs/alerting/alert-rules.md` + `.ru.md` — примеры алертов с `app_dependency_status`
- `CHANGELOG.md` — записи для всех SDK
- Версии: Go → v0.4.0, Java → v0.4.0, Python → v0.4.0, C# → v0.4.0
- `README.md` / `README.ru.md` — обновить таблицу метрик

---

## Ключевые файлы для модификации

| SDK | Metrics exporter | Scheduler | Checker interface | Error types |
|-----|-----------------|-----------|-------------------|-------------|
| Go | `sdk-go/dephealth/metrics.go` | `sdk-go/dephealth/scheduler.go` | `sdk-go/dephealth/checker.go` | `sdk-go/dephealth/check_result.go` (new) |
| Java | `sdk-java/.../metrics/MetricsExporter.java` | `sdk-java/.../scheduler/CheckScheduler.java` | `sdk-java/.../HealthChecker.java` | `sdk-java/.../CheckException.java` (new) |
| Python | `sdk-python/dephealth/metrics.py` | `sdk-python/dephealth/scheduler.py` | `sdk-python/dephealth/checker.py` | `sdk-python/dephealth/check_result.py` (new) |
| C# | `sdk-csharp/DepHealth.Core/PrometheusExporter.cs` | `sdk-csharp/DepHealth.Core/CheckScheduler.cs` | `sdk-csharp/DepHealth.Core/IHealthChecker.cs` | `sdk-csharp/DepHealth.Core/CheckResult.cs` (new) |

## Существующие ресурсы для переиспользования

- **Go**: sentinel errors `ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy` в `checker.go`; метод `labels()` в `metrics.go`; `safeCheck()` panic recovery в `scheduler.go`
- **Python**: иерархия `CheckError` → `CheckTimeoutError`, `CheckConnectionRefusedError`, `UnhealthyError` в `checker.py`; `_labels()` в `metrics.py`
- **Java**: `EndpointState` threshold logic в `scheduler/EndpointState.java`; `buildTags()` в `MetricsExporter.java`; builder pattern в чекерах
- **C#**: Exceptions `UnhealthyException`, `ConnectionRefusedException`, `CheckTimeoutException` (определены, но не все используются) в `Exceptions/`; `BuildLabelValues()` в `PrometheusExporter.cs`

## Верификация

### Автоматическая (для каждого SDK после каждой фазы):
```bash
# Go
cd sdk-go && make test && make lint

# Java
cd sdk-java && make test && make lint

# Python
cd sdk-python && make test && make lint

# C#
cd sdk-csharp && make test && make lint
```

### Ручная (после всех фаз):
1. `docker compose --profile full up -d` — поднять все 7 зависимостей
2. Запустить тестовый сервис с SDK
3. Проверить `/metrics` endpoint — наличие всех 3 метрик (health, status, status_detail)
4. Остановить одну из зависимостей → проверить что `status != "ok"` и `detail` содержит конкретику
5. Проверить в Grafana: визуализация по `app_dependency_status`, группировка по `status`
6. Проверить PromQL-запросы из раздела "PromQL-примеры"

### Checklist per SDK:
- [ ] `make test` — все тесты зелёные
- [ ] `make lint` — без ошибок
- [ ] Метрика `app_dependency_status` экспортирует ровно 8 серий на endpoint
- [ ] Ровно одна серия = 1, остальные 7 = 0
- [ ] Метрика `app_dependency_status_detail` экспортирует 1 серию на endpoint
- [ ] При смене detail — старая серия удаляется
- [ ] До первой проверки — обе метрики не экспортируются
- [ ] Существующие `app_dependency_health` и `app_dependency_latency_seconds` не изменились
