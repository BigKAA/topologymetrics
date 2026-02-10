[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Check Behavior Contract

> Specification version: **1.0-draft**
>
> This document describes the behavior of dependency health checks:
> lifecycle, threshold logic, check types, operating modes, and error handling.
> All SDKs must implement the described behavior. Compliance is verified
> by conformance tests.

---

## 1. Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | `15s` | Interval between the start of consecutive checks |
| `timeout` | `5s` | Maximum time to wait for a response from the dependency |
| `initialDelay` | `5s` | Delay before the first check after SDK startup |
| `failureThreshold` | `1` | Number of consecutive failures to transition to unhealthy |
| `successThreshold` | `1` | Number of consecutive successes to return to healthy |

All parameters are configurable:

- Globally — for all dependencies.
- Individually — for a specific dependency (takes priority over global settings).

**Value constraints**:

| Parameter | Minimum | Maximum |
| --- | --- | --- |
| `checkInterval` | `1s` | `10m` |
| `timeout` | `100ms` | `30s` |
| `initialDelay` | `0s` | `5m` |
| `failureThreshold` | `1` | `10` |
| `successThreshold` | `1` | `10` |

The `timeout` value must be less than `checkInterval`. If `timeout >= checkInterval` is specified,
the SDK must return a configuration error during initialization.

---

## 2. Check Lifecycle

### 2.1. State Diagram

```text
Start()
  │
  ▼
[INIT] ── initialDelay ──► [CHECKING]
                               │
                    ┌──────────┤
                    │          │
                    ▼          ▼
              [HEALTHY]   [UNHEALTHY]
                    │          │
                    └────◄─────┘
                 (threshold logic)
                         │
                    Stop()│
                         ▼
                      [STOPPED]
```

### 2.2. Phases

#### INIT — initialization

1. SDK creates a goroutine / thread / task for each dependency.
2. Wait for `initialDelay`.
3. Metrics for this dependency are **not exported**.

#### CHECKING — active checks

1. First check is performed.
2. Based on the result, initial state is set: HEALTHY or UNHEALTHY.
3. Metrics start being exported.
4. Subsequent checks are performed at `checkInterval` intervals.

#### HEALTHY / UNHEALTHY — stable state

- State changes only when threshold is reached
  (see section 3 "Threshold Logic").
- Each check updates `app_dependency_latency_seconds`.
- When state changes, `app_dependency_health` is updated.

#### STOPPED — shutdown

1. Call `Stop()` / `close()` / graceful shutdown.
2. Context cancellation / thread interruption.
3. Wait for current checks to complete (with `timeout` deadline).
4. Metrics **remain** with last values (are not reset).

**Rationale**: resetting metrics on shutdown creates false alerts.
Metrics will disappear on next scrape if the process has terminated.

### 2.3. Check Interval

The `checkInterval` is measured from the **start** of the previous check,
not from its completion.

```text
t=0s     t=15s    t=30s    t=45s
│        │        │        │
▼        ▼        ▼        ▼
[check]  [check]  [check]  [check]
 3ms      5ms      2ms      4ms
```

If a check takes longer than `checkInterval`, the next check
starts immediately after the current one completes (without skipping).

---

## 3. Threshold Logic

### 3.1. Transition HEALTHY -> UNHEALTHY

```text
failureThreshold = 3

Check:     OK   OK   FAIL  FAIL  FAIL  → UNHEALTHY
Counter:   0    0    1     2     3
Metric:    1    1    1     1     0
```

- On each failed check, the `consecutiveFailures` counter increments by 1.
- On a successful check, the `consecutiveFailures` counter is reset to 0.
- When `consecutiveFailures >= failureThreshold`, state transitions to UNHEALTHY.
- The `app_dependency_health` metric changes to `0` **at the moment the threshold is reached**.

### 3.2. Transition UNHEALTHY -> HEALTHY

```text
successThreshold = 2

Check:     FAIL  FAIL  OK   OK   → HEALTHY
Counter:   0     0     1    2
Metric:    0     0     0    1
```

- On each successful check, the `consecutiveSuccesses` counter increments by 1.
- On a failed check, the `consecutiveSuccesses` counter is reset to 0.
- When `consecutiveSuccesses >= successThreshold`, state transitions to HEALTHY.
- The `app_dependency_health` metric changes to `1` **at the moment the threshold is reached**.

### 3.3. Initial State

Before the first check, state is **UNKNOWN** (metric is not exported).

The first check determines the initial state:

- Success → immediately HEALTHY (`app_dependency_health = 1`), without considering `successThreshold`.
- Failure → immediately UNHEALTHY (`app_dependency_health = 0`), without considering `failureThreshold`.

**Rationale**: at service startup, it's important to get the actual
dependency status as quickly as possible. Threshold logic protects against brief
failures in steady-state operation.

### 3.4. Counters with Threshold 1

With `failureThreshold = 1` and `successThreshold = 1` (default values),
each check immediately updates the state:

```text
Check:   OK   FAIL  OK   FAIL  OK
Metric:  1    0     1    0     1
```

---

## 4. Check Types

### 4.1. HTTP (`type: http`)

| Parameter | Description | Default Value |
| --- | --- | --- |
| `healthPath` | Path for the check | `/health` |
| `method` | HTTP method | `GET` |
| `expectedStatuses` | Expected HTTP status codes | `200-299` (any 2xx) |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |

**Algorithm**:

1. Send `GET` (or configured method) to `http(s)://{host}:{port}{healthPath}`.
2. Wait for response within `timeout`.
3. If response status is in the `expectedStatuses` range — **success**.
4. Otherwise — **failure**.

**Specifics**:

- Response body is not analyzed (only status code).
- Redirects (3xx) are followed automatically; final response status is checked.
- For `https://`, TLS is used; if certificate is invalid and `tlsSkipVerify = false` — failure.
- `User-Agent: dephealth/<version>` header is set.

### 4.2. gRPC (`type: grpc`)

**Protocol**: [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
(package `grpc.health.v1`, method `Health/Check`).

| Parameter | Description | Default Value |
| --- | --- | --- |
| `serviceName` | Service name for Health Check | `""` (empty string — overall status) |
| `tlsEnabled` | Use TLS | `false` |
| `tlsSkipVerify` | Skip TLS certificate verification | `false` |

**Algorithm**:

1. Establish gRPC connection to `{host}:{port}`.
2. Call `grpc.health.v1.Health/Check` with the specified `serviceName`.
3. If response is `SERVING` — **success**.
4. Other statuses (`NOT_SERVING`, `UNKNOWN`, `SERVICE_UNKNOWN`) — **failure**.

**Specifics**:

- Connection is created anew for each check (standalone mode).
- Timeout is passed via gRPC deadline.

### 4.3. TCP (`type: tcp`)

**Algorithm**:

1. Establish TCP connection to `{host}:{port}` (with `timeout`).
2. If connection is established — **success**.
3. Immediately close the connection.

**Specifics**:

- No data is sent or read.
- Suitable for arbitrary TCP services without a specific check protocol.

### 4.4. PostgreSQL (`type: postgres`)

| Parameter | Description | Default Value |
| --- | --- | --- |
| `query` | SQL query for the check | `SELECT 1` |

**Standalone mode**:

1. Establish new TCP connection to PostgreSQL (`{host}:{port}`).
2. Perform authentication (if required).
3. Execute `SELECT 1`.
4. If result is received — **success**.
5. Close the connection.

**Connection pool mode**:

1. Acquire connection from pool (`db.QueryContext(ctx, "SELECT 1")`).
2. If query succeeds — **success**.
3. Connection is returned to the pool.

**Specifics**:

- In pool mode, not only database availability is checked, but also pool health.
- If pool is exhausted and `timeout` expires before acquiring a connection — **failure**.
- TLS support (if specified in connection string).

### 4.5. MySQL (`type: mysql`)

Similar to PostgreSQL. The only difference is the connection driver.

| Parameter | Description | Default Value |
| --- | --- | --- |
| `query` | SQL query for the check | `SELECT 1` |

### 4.6. Redis (`type: redis`)

**Standalone mode**:

1. Establish TCP connection to Redis (`{host}:{port}`).
2. Perform authentication (if password is specified).
3. Send `PING` command.
4. Wait for `PONG` response — **success**.
5. Close the connection.

**Connection pool mode**:

1. Use existing client (`client.Ping(ctx)`).
2. If response is `PONG` — **success**.

**Specifics**:

- Redis Sentinel and Redis Cluster support is not included in v1.0.
- TLS support (`rediss://`).
- Database selection support (`/0`, `/1`, ...).

### 4.7. AMQP (`type: amqp`)

**Standalone mode**:

1. Establish AMQP connection to `{host}:{port}` (with specified `vhost`).
2. If connection is established (connection.open) — **success**.
3. Close the connection.

**Specifics**:

- Channel is not created (connection-level check is sufficient).
- TLS support (`amqps://`).
- Vhost support (from URL or separate parameter).
- Authentication: username/password from URL or configuration.

### 4.8. Kafka (`type: kafka`)

**Standalone mode**:

1. Create Kafka Admin Client (or minimal client).
2. Send Metadata request to broker `{host}:{port}`.
3. If response with metadata is received — **success**.
4. Close the client.

**Specifics**:

- Each broker is checked independently.
- Only network availability and response capability of the broker are checked.
- Topic-level checks are not included in v1.0.
- SASL authentication support is not included in v1.0.

---

## 5. Two Operating Modes

### 5.1. Standalone Mode

SDK **creates a new connection** for each check.

```text
SDK → net.Dial / http.Get / sql.Open → check → Close
```

**When to use**:

- Dependency does not have a connection pool in the service (HTTP, gRPC, TCP).
- Developer did not provide a reference to the pool.

**Advantages**:

- Simple configuration — URL is sufficient.
- Independent of the service's pool state.

**Disadvantages**:

- Does not reflect the actual ability of the service to work with the dependency.
- If pool is exhausted, standalone check will still show `healthy`.
- Additional overhead (creating/closing connections).

### 5.2. Connection Pool Mode (Integrated)

SDK **uses the service's existing connection pool**.

```text
SDK → pool.GetConnection() → SELECT 1 / PING → pool.Release()
```

**When to use**:

- Dependency uses a connection pool (DB, Redis).
- Developer provided a reference to the pool / client.

**Advantages**:

- Reflects the actual ability of the service to work with the dependency.
- If pool is exhausted — check shows `unhealthy`.
- No additional connections.

**Disadvantages**:

- Requires passing a pool reference during initialization.
- Depends on the specific library (go-redis, database/sql, etc.).

### 5.3. Priority

Pool mode is **preferred**. SDK should encourage its use
in documentation and examples. Standalone mode is a fallback for cases
when pool is not available.

---

## 6. Error Handling

All listed situations are treated as check **failure**:

| Situation | Behavior |
| --- | --- |
| Timeout (`timeout` expired) | Failure. Latency = actual time until timeout |
| DNS resolution failure | Failure. Latency = time until DNS error |
| Connection refused | Failure. Latency = time until receiving RST |
| Connection reset | Failure. Latency = time until reset |
| TLS handshake failure | Failure. Latency = time until TLS error |
| HTTP 5xx | Failure (not 2xx) |
| HTTP 3xx (redirect) | Follows redirect; result determined by final response |
| HTTP 4xx | Failure (not 2xx) |
| gRPC NOT_SERVING | Failure |
| SQL error (authentication, syntax) | Failure |
| Pool exhausted (connection acquisition timeout) | Failure |
| Panic / unhandled exception | Failure. SDK must catch and log |

### 6.1. Error Logging

SDK logs each check error at `WARN` level:

```text
WARN dephealth: check failed dependency=postgres-main host=pg.svc port=5432 error="connection refused"
```

First transition to unhealthy is logged at `ERROR` level:

```text
ERROR dephealth: dependency unhealthy dependency=postgres-main host=pg.svc port=5432 consecutive_failures=3
```

Return to healthy is logged at `INFO` level:

```text
INFO dephealth: dependency recovered dependency=postgres-main host=pg.svc port=5432
```

### 6.2. Panic / Unexpected Errors

SDK must catch panic (Go), unhandled exception (Java/C#/Python)
inside the check. Panic must not interrupt the scheduler's operation.
Check is considered failed, error is logged at `ERROR` level.

---

## 7. Concurrency and Thread Safety

- Checks for different dependencies run **in parallel** (each in its own goroutine / thread).
- Checks for different endpoints of the same dependency also run **in parallel**.
- Metric updates are **thread-safe** (guaranteed by Prometheus client).
- Internal state updates (threshold counters) are **thread-safe**
  (atomic / mutex / synchronized).
- `Start()` and `Stop()` can be called only once. Repeated call to `Start()`
  returns an error. Repeated call to `Stop()` is a no-op.

---

<a id="russian"></a>

# Контракт поведения проверок

> Версия спецификации: **1.0-draft**
>
> Этот документ описывает поведение проверок здоровья зависимостей:
> жизненный цикл, логику порогов, типы проверок, режимы работы и обработку ошибок.
> Все SDK обязаны реализовать описанное поведение. Соответствие проверяется
> conformance-тестами.

---

## 1. Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `checkInterval` | `15s` | Интервал между началом последовательных проверок |
| `timeout` | `5s` | Максимальное время ожидания ответа от зависимости |
| `initialDelay` | `5s` | Задержка перед первой проверкой после старта SDK |
| `failureThreshold` | `1` | Количество последовательных неудач для перехода в unhealthy |
| `successThreshold` | `1` | Количество последовательных успехов для возврата в healthy |

Все параметры настраиваются:

- Глобально — для всех зависимостей.
- Индивидуально — для конкретной зависимости (приоритет над глобальными).

**Ограничения значений**:

| Параметр | Минимум | Максимум |
| --- | --- | --- |
| `checkInterval` | `1s` | `10m` |
| `timeout` | `100ms` | `30s` |
| `initialDelay` | `0s` | `5m` |
| `failureThreshold` | `1` | `10` |
| `successThreshold` | `1` | `10` |

Значение `timeout` должно быть меньше `checkInterval`. Если задано `timeout >= checkInterval`,
SDK должен вернуть ошибку конфигурации при инициализации.

---

## 2. Жизненный цикл проверки

### 2.1. Диаграмма состояний

```text
Start()
  │
  ▼
[INIT] ── initialDelay ──► [CHECKING]
                               │
                    ┌──────────┤
                    │          │
                    ▼          ▼
              [HEALTHY]   [UNHEALTHY]
                    │          │
                    └────◄─────┘
                   (пороговая логика)
                         │
                    Stop()│
                         ▼
                      [STOPPED]
```

### 2.2. Фазы

#### INIT — инициализация

1. SDK создаёт горутину / поток / задачу для каждой зависимости.
2. Ожидание `initialDelay`.
3. Метрики для этой зависимости **не экспортируются**.

#### CHECKING — активные проверки

1. Выполняется первая проверка.
2. По результату устанавливается начальное состояние: HEALTHY или UNHEALTHY.
3. Метрики начинают экспортироваться.
4. Далее проверки выполняются с интервалом `checkInterval`.

#### HEALTHY / UNHEALTHY — устойчивое состояние

- Состояние меняется только при достижении порога
  (см. раздел 3 "Логика порогов").
- Каждая проверка обновляет `app_dependency_latency_seconds`.
- При изменении состояния обновляется `app_dependency_health`.

#### STOPPED — остановка

1. Вызов `Stop()` / `close()` / graceful shutdown.
2. Отмена контекста / прерывание потоков.
3. Ожидание завершения текущих проверок (с таймаутом `timeout`).
4. Метрики **остаются** с последними значениями (не обнуляются).

**Обоснование**: обнуление метрик при остановке создаёт ложные алерты.
Метрики исчезнут при следующем scrape, если процесс завершился.

### 2.3. Интервал проверок

Интервал `checkInterval` отсчитывается от **начала** предыдущей проверки,
а не от её завершения.

```text
t=0s     t=15s    t=30s    t=45s
│        │        │        │
▼        ▼        ▼        ▼
[check]  [check]  [check]  [check]
 3ms      5ms      2ms      4ms
```

Если проверка занимает больше `checkInterval`, следующая проверка
запускается немедленно после завершения текущей (без пропуска).

---

## 3. Логика порогов

### 3.1. Переход HEALTHY -> UNHEALTHY

```text
failureThreshold = 3

Проверка:  OK   OK   FAIL  FAIL  FAIL  → UNHEALTHY
Счётчик:   0    0    1     2     3
Метрика:   1    1    1     1     0
```

- При каждой неудачной проверке счётчик `consecutiveFailures` увеличивается на 1.
- При успешной проверке счётчик `consecutiveFailures` сбрасывается в 0.
- Когда `consecutiveFailures >= failureThreshold`, состояние переходит в UNHEALTHY.
- Метрика `app_dependency_health` меняется на `0` **в момент достижения порога**.

### 3.2. Переход UNHEALTHY -> HEALTHY

```text
successThreshold = 2

Проверка:  FAIL  FAIL  OK   OK   → HEALTHY
Счётчик:   0     0     1    2
Метрика:   0     0     0    1
```

- При каждой успешной проверке счётчик `consecutiveSuccesses` увеличивается на 1.
- При неудачной проверке счётчик `consecutiveSuccesses` сбрасывается в 0.
- Когда `consecutiveSuccesses >= successThreshold`, состояние переходит в HEALTHY.
- Метрика `app_dependency_health` меняется на `1` **в момент достижения порога**.

### 3.3. Начальное состояние

До первой проверки состояние — **UNKNOWN** (метрика не экспортируется).

Первая проверка определяет начальное состояние:

- Успех → сразу HEALTHY (`app_dependency_health = 1`), без учёта `successThreshold`.
- Неудача → сразу UNHEALTHY (`app_dependency_health = 0`), без учёта `failureThreshold`.

**Обоснование**: при старте сервиса важно как можно быстрее получить
реальный статус зависимости. Пороговая логика защищает от кратковременных
сбоев в устоявшемся режиме.

### 3.4. Счётчики при пороге 1

При `failureThreshold = 1` и `successThreshold = 1` (значения по умолчанию)
каждая проверка немедленно обновляет состояние:

```text
Проверка:  OK   FAIL  OK   FAIL  OK
Метрика:   1    0     1    0     1
```

---

## 4. Типы проверок

### 4.1. HTTP (`type: http`)

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| `healthPath` | Путь для проверки | `/health` |
| `method` | HTTP-метод | `GET` |
| `expectedStatuses` | Ожидаемые HTTP-статусы | `200-299` (любой 2xx) |
| `tlsSkipVerify` | Пропуск проверки TLS-сертификата | `false` |

**Алгоритм**:

1. Отправить `GET` (или настроенный метод) на `http(s)://{host}:{port}{healthPath}`.
2. Ожидать ответ в пределах `timeout`.
3. Если статус ответа в диапазоне `expectedStatuses` — **успех**.
4. Иначе — **неудача**.

**Особенности**:

- Тело ответа не анализируется (только статус-код).
- Редиректы (3xx) следуются автоматически; проверяется статус финального ответа.
- При `https://` используется TLS; если сертификат невалиден и `tlsSkipVerify = false` — неудача.
- Заголовок `User-Agent: dephealth/<version>`.

### 4.2. gRPC (`type: grpc`)

**Протокол**: [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
(пакет `grpc.health.v1`, метод `Health/Check`).

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| `serviceName` | Имя сервиса для Health Check | `""` (пустая строка — общий статус) |
| `tlsEnabled` | Использовать TLS | `false` |
| `tlsSkipVerify` | Пропуск проверки TLS-сертификата | `false` |

**Алгоритм**:

1. Установить gRPC-соединение с `{host}:{port}`.
2. Вызвать `grpc.health.v1.Health/Check` с указанным `serviceName`.
3. Если ответ `SERVING` — **успех**.
4. Иные статусы (`NOT_SERVING`, `UNKNOWN`, `SERVICE_UNKNOWN`) — **неудача**.

**Особенности**:

- Соединение создаётся заново для каждой проверки (автономный режим).
- Таймаут передаётся через gRPC deadline.

### 4.3. TCP (`type: tcp`)

**Алгоритм**:

1. Установить TCP-соединение с `{host}:{port}` (с таймаутом `timeout`).
2. Если соединение установлено — **успех**.
3. Немедленно закрыть соединение.

**Особенности**:

- Никакие данные не отправляются и не читаются.
- Подходит для произвольных TCP-сервисов без специфического протокола проверки.

### 4.4. PostgreSQL (`type: postgres`)

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| `query` | SQL-запрос для проверки | `SELECT 1` |

**Автономный режим**:

1. Установить новое TCP-соединение с PostgreSQL (`{host}:{port}`).
2. Выполнить аутентификацию (если требуется).
3. Выполнить `SELECT 1`.
4. Если получен результат — **успех**.
5. Закрыть соединение.

**Режим с connection pool**:

1. Получить соединение из pool (`db.QueryContext(ctx, "SELECT 1")`).
2. Если запрос успешен — **успех**.
3. Соединение возвращается в pool.

**Особенности**:

- В режиме с pool проверяется не только доступность БД, но и работоспособность pool.
- Если pool исчерпан и `timeout` истёк до получения соединения — **неудача**.
- Поддержка TLS (если указано в connection string).

### 4.5. MySQL (`type: mysql`)

Аналогично PostgreSQL. Единственное отличие — драйвер подключения.

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| `query` | SQL-запрос для проверки | `SELECT 1` |

### 4.6. Redis (`type: redis`)

**Автономный режим**:

1. Установить TCP-соединение с Redis (`{host}:{port}`).
2. Выполнить аутентификацию (если пароль указан).
3. Отправить команду `PING`.
4. Ожидать ответ `PONG` — **успех**.
5. Закрыть соединение.

**Режим с connection pool**:

1. Использовать существующий клиент (`client.Ping(ctx)`).
2. Если ответ `PONG` — **успех**.

**Особенности**:

- Поддержка Redis Sentinel и Redis Cluster не входит в v1.0.
- Поддержка TLS (`rediss://`).
- Поддержка выбора базы данных (`/0`, `/1`, ...).

### 4.7. AMQP (`type: amqp`)

**Автономный режим**:

1. Установить AMQP-соединение с `{host}:{port}` (с указанным `vhost`).
2. Если соединение установлено (connection.open) — **успех**.
3. Закрыть соединение.

**Особенности**:

- Channel не создаётся (достаточно connection-level проверки).
- Поддержка TLS (`amqps://`).
- Поддержка vhost (из URL или отдельного параметра).
- Аутентификация: username/password из URL или конфигурации.

### 4.8. Kafka (`type: kafka`)

**Автономный режим**:

1. Создать Kafka Admin Client (или минимальный клиент).
2. Отправить Metadata request к брокеру `{host}:{port}`.
3. Если получен ответ с метаданными — **успех**.
4. Закрыть клиент.

**Особенности**:

- Каждый брокер проверяется независимо.
- Проверяется только сетевая доступность брокера и его способность ответить.
- Topic-level проверки не входят в v1.0.
- Поддержка SASL-аутентификации не входит в v1.0.

---

## 5. Два режима работы

### 5.1. Автономный режим (Standalone)

SDK **создаёт новое соединение** для каждой проверки.

```text
SDK → net.Dial / http.Get / sql.Open → проверка → Close
```

**Когда использовать**:

- Зависимость не имеет connection pool в сервисе (HTTP, gRPC, TCP).
- Разработчик не предоставил ссылку на pool.

**Преимущества**:

- Простая конфигурация — достаточно URL.
- Не зависит от состояния pool сервиса.

**Недостатки**:

- Не отражает реальную способность сервиса работать с зависимостью.
- Если pool исчерпан, автономная проверка всё ещё покажет `healthy`.
- Дополнительная нагрузка (создание/закрытие соединений).

### 5.2. Режим с connection pool (Integrated)

SDK **использует существующий connection pool** сервиса.

```text
SDK → pool.GetConnection() → SELECT 1 / PING → pool.Release()
```

**Когда использовать**:

- Зависимость использует connection pool (БД, Redis).
- Разработчик предоставил ссылку на pool / client.

**Преимущества**:

- Отражает реальную способность сервиса работать с зависимостью.
- Если pool исчерпан — проверка покажет `unhealthy`.
- Нет дополнительных соединений.

**Недостатки**:

- Требует передачи ссылки на pool при инициализации.
- Зависит от конкретной библиотеки (go-redis, database/sql и т.д.).

### 5.3. Приоритет

Режим с pool является **предпочтительным**. SDK должен поощрять его использование
в документации и примерах. Автономный режим — fallback для случаев,
когда pool недоступен.

---

## 6. Обработка ошибок

Все перечисленные ситуации трактуются как **неудача** проверки:

| Ситуация | Поведение |
| --- | --- |
| Таймаут (`timeout` истёк) | Неудача. Латентность = фактическое время до таймаута |
| DNS resolution failure | Неудача. Латентность = время до ошибки DNS |
| Connection refused | Неудача. Латентность = время до получения RST |
| Connection reset | Неудача. Латентность = время до сброса |
| TLS handshake failure | Неудача. Латентность = время до ошибки TLS |
| HTTP 5xx | Неудача (не 2xx) |
| HTTP 3xx (redirect) | Следует редиректу; результат определяется финальным ответом |
| HTTP 4xx | Неудача (не 2xx) |
| gRPC NOT_SERVING | Неудача |
| SQL ошибка (аутентификация, синтаксис) | Неудача |
| Pool exhausted (таймаут получения соединения) | Неудача |
| Panic / unhandled exception | Неудача. SDK должен перехватить и логировать |

### 6.1. Логирование ошибок

SDK логирует каждую ошибку проверки с уровнем `WARN`:

```text
WARN dephealth: check failed dependency=postgres-main host=pg.svc port=5432 error="connection refused"
```

Первый переход в unhealthy логируется с уровнем `ERROR`:

```text
ERROR dephealth: dependency unhealthy dependency=postgres-main host=pg.svc port=5432 consecutive_failures=3
```

Возврат в healthy логируется с уровнем `INFO`:

```text
INFO dephealth: dependency recovered dependency=postgres-main host=pg.svc port=5432
```

### 6.2. Паника / неожиданные ошибки

SDK обязан перехватывать panic (Go), unhandled exception (Java/C#/Python)
внутри проверки. Паника не должна прерывать работу планировщика.
Проверка считается неудачной, ошибка логируется с уровнем `ERROR`.

---

## 7. Конкурентность и потокобезопасность

- Проверки разных зависимостей выполняются **параллельно** (каждая в своей горутине / потоке).
- Проверки разных endpoint-ов одной зависимости также выполняются **параллельно**.
- Обновление метрик — **потокобезопасно** (гарантируется Prometheus-клиентом).
- Обновление внутреннего состояния (счётчики порогов) — **потокобезопасно**
  (atomic / mutex / synchronized).
- `Start()` и `Stop()` можно вызвать только один раз. Повторный вызов `Start()`
  возвращает ошибку. Повторный вызов `Stop()` — no-op.
