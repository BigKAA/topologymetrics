*[English version](check-behavior.md)*

# Контракт поведения проверок

> Версия спецификации: **2.0-draft**
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
| `headers` | Произвольные HTTP-заголовки, добавляемые к каждому запросу | `{}` (пусто) |
| `bearerToken` | Добавляет заголовок `Authorization: Bearer <token>` | `""` (отключено) |
| `basicAuth` | Добавляет заголовок `Authorization: Basic <base64(user:pass)>` | не задано |

**Алгоритм**:

1. Отправить `GET` (или настроенный метод) на `http(s)://{host}:{port}{healthPath}`.
2. Добавить настроенные заголовки (custom headers, bearer token или basic auth) к запросу.
3. Ожидать ответ в пределах `timeout`.
4. Если статус ответа в диапазоне `expectedStatuses` — **успех**.
5. Иначе — **неудача**.

**Аутентификация**:

- `headers` — произвольные пары ключ-значение, добавляемые как HTTP-заголовки
  к каждому запросу проверки здоровья.
- `bearerToken` — вспомогательный параметр; добавляет заголовок `Authorization: Bearer <token>`.
- `basicAuth` — вспомогательный параметр с полями `username` и `password`;
  добавляет заголовок `Authorization: Basic <base64(username:password)>`.
- Допускается только один метод аутентификации одновременно. Если указано более одного
  из следующих, SDK должен вернуть **ошибку валидации** при инициализации:
  - `bearerToken` задан И `headers` содержит ключ `Authorization`
  - `basicAuth` задан И `headers` содержит ключ `Authorization`
  - `bearerToken` задан И `basicAuth` задан

**Особенности**:

- Тело ответа не анализируется (только статус-код).
- Редиректы (3xx) следуются автоматически; проверяется статус финального ответа.
- При `https://` используется TLS; если сертификат невалиден и `tlsSkipVerify = false` — неудача.
- Заголовок `User-Agent: dephealth/<version>`. Пользовательский `User-Agent` в `headers` переопределяет его.
- HTTP-ответы 401 и 403 классифицируются как `auth_error` (см. раздел 6.2.3).

### 4.2. gRPC (`type: grpc`)

**Протокол**: [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
(пакет `grpc.health.v1`, метод `Health/Check`).

| Параметр | Описание | Значение по умолчанию |
| --- | --- | --- |
| `serviceName` | Имя сервиса для Health Check | `""` (пустая строка — общий статус) |
| `tlsEnabled` | Использовать TLS | `false` |
| `tlsSkipVerify` | Пропуск проверки TLS-сертификата | `false` |
| `metadata` | Произвольные gRPC-метаданные, добавляемые к каждому вызову Health/Check | `{}` (пусто) |
| `bearerToken` | Добавляет метаданные `authorization: Bearer <token>` | `""` (отключено) |
| `basicAuth` | Добавляет метаданные `authorization: Basic <base64(user:pass)>` | не задано |

**Алгоритм**:

1. Установить gRPC-соединение с `{host}:{port}`.
2. Добавить настроенные метаданные (custom metadata, bearer token или basic auth) к вызову.
3. Вызвать `grpc.health.v1.Health/Check` с указанным `serviceName`.
4. Если ответ `SERVING` — **успех**.
5. Иные статусы (`NOT_SERVING`, `UNKNOWN`, `SERVICE_UNKNOWN`) — **неудача**.

**Аутентификация**:

- `metadata` — произвольные пары ключ-значение, добавляемые как gRPC-метаданные
  к каждому вызову Health/Check.
- `bearerToken` — вспомогательный параметр; добавляет метаданные `authorization: Bearer <token>`.
- `basicAuth` — вспомогательный параметр с полями `username` и `password`;
  добавляет метаданные `authorization: Basic <base64(username:password)>`.
- Те же правила валидации, что и для HTTP (раздел 4.1): допускается только один метод
  аутентификации. Конфликт между `bearerToken`, `basicAuth` и пользовательским ключом
  `authorization` в metadata приводит к **ошибке валидации**.
- gRPC-статусы `UNAUTHENTICATED` и `PERMISSION_DENIED` классифицируются как `auth_error`
  (см. раздел 6.2.3).

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

### 6.2. Классификация ошибок

Каждая ошибка проверки классифицируется в **категорию статуса** и **детальное значение**.
Эта классификация используется для заполнения метрик `app_dependency_status` и
`app_dependency_status_detail` (см. metric-contract.ru.md, разделы 8-9).

#### 6.2.1. Цепочка классификации

Планировщик классифицирует ошибки по следующей цепочке приоритетов:

1. **Интерфейс ClassifiedError** — если ошибка реализует интерфейс `ClassifiedError`
   (Go: `StatusCategory() string` + `StatusDetail() string`;
   Java/C#: базовый класс исключения; Python: свойства на `CheckError`),
   используются её категория и detail.
2. **Sentinel-ошибки** — известные типизированные ошибки (`ErrTimeout`, `ErrConnectionRefused`,
   `ErrUnhealthy` и т.д.) маппятся на соответствующую категорию/detail.
3. **Определение по платформенным ошибкам** — проверяются стандартные типы ошибок:
   - `context.DeadlineExceeded` / `TimeoutException` → `timeout` / `timeout`
   - `*net.DNSError` / `UnknownHostException` / `socket.gaierror` → `dns_error` / `dns_error`
   - `*net.OpError` (connection refused) / `ConnectException` → `connection_error` / `connection_refused`
   - `*tls.CertificateVerificationError` / `SSLException` → `tls_error` / `tls_error`
4. **Fallback** — нераспознанные ошибки → `error` / `error`.

#### 6.2.2. Успешная проверка

Успешная проверка (без ошибки) всегда даёт:

- категория статуса: `ok`
- detail: `ok`

#### 6.2.3. Значения detail по типам чекеров

| Тип чекера | Возможные значения detail |
| --- | --- |
| HTTP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `http_NNN` (например, `http_404`, `http_503`), `error` |
| gRPC | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `grpc_not_serving`, `grpc_unknown`, `error` |
| TCP | `ok`, `timeout`, `connection_refused`, `dns_error`, `error` |
| PostgreSQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| MySQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| Redis | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `unhealthy`, `error` |
| AMQP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Kafka | `ok`, `timeout`, `connection_refused`, `dns_error`, `no_brokers`, `error` |

#### 6.2.4. Маппинг detail → status

| detail | status |
| --- | --- |
| `ok` | `ok` |
| `timeout` | `timeout` |
| `connection_refused`, `network_unreachable`, `host_unreachable` | `connection_error` |
| `dns_error` | `dns_error` |
| `auth_error` | `auth_error` |
| `tls_error` | `tls_error` |
| `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers` | `unhealthy` |
| `error`, `pool_exhausted`, `query_error` | `error` |

#### 6.2.5. Обратная совместимость

Интерфейс `HealthChecker` **не меняется**. Чекеры, не возвращающие
классифицированные ошибки, обрабатываются определением по платформенным ошибкам
и fallback в цепочке классификации (шаги 3-4). Пользовательские чекеры
автоматически получают классификацию через этот механизм.

### 6.3. Паника / неожиданные ошибки

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

---

## 8. Программный API HealthDetails

Метод `HealthDetails()` предоставляет программный API, который раскрывает
детальное состояние проверок здоровья для каждого зарегистрированного endpoint-а.
В отличие от `Health()`, который возвращает простую карту `healthy/unhealthy`,
`HealthDetails()` возвращает богатую структуру с классификацией, латентностью,
метаданными и временными метками.

Этот API позволяет потребителям (страницы статуса, реверс-прокси,
кастомные операторы) строить обогащённые эндпоинты здоровья без
парсинга Prometheus-метрик.

### 8.1. Публичный метод

Главный фасад каждого SDK предоставляет метод `HealthDetails()`:

| SDK | Фасад | Метод | Тип возвращаемого значения |
| --- | --- | --- | --- |
| Go | `DepHealth` | `HealthDetails()` | `map[string]EndpointStatus` |
| Java | `DepHealth` | `healthDetails()` | `Map<String, EndpointStatus>` |
| Python | `DependencyHealth` | `health_details()` | `dict[str, EndpointStatus]` |
| C# | `DepHealthMonitor` | `HealthDetails()` | `Dictionary<string, EndpointStatus>` |

Метод делегирует вызов планировщику по тому же паттерну, что и `Health()`.

### 8.2. Формат ключей

Ключи карты используют формат `"dependency:host:port"`, единый для всех SDK.
Ключи в `HealthDetails()` **должны совпадать** с ключами в `Health()`
для Go, Java и C#.

> **Примечание**: Python `health()` агрегирует только по имени зависимости.
> Новый `health_details()` намеренно использует ключи вида
> `"dependency:host:port"` для единообразия с другими SDK и обеспечения
> гранулярности на уровне endpoint-ов. Это задокументированное отличие.

### 8.3. Структура `EndpointStatus`

Структура `EndpointStatus` содержит 11 полей. Все поля обязательны
в реализации каждого SDK.

| # | Поле | Go | Java | Python | C# | Описание |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Healthy | `*bool` | `Boolean` | `bool \| None` | `bool?` | `nil`/`null`/`None` = UNKNOWN |
| 2 | Status | `StatusCategory` | `StatusCategory` | `StatusCategory` | `StatusCategory` | Категория классификации |
| 3 | Detail | `string` | `String` | `str` | `string` | Конкретная причина сбоя |
| 4 | Latency | `time.Duration` | `Duration` | `float` (секунды) | `TimeSpan` | Длительность последней проверки |
| 5 | Type | `DependencyType` | `DependencyType` | `DependencyType` | `DependencyType` | Тип зависимости |
| 6 | Name | `string` | `String` | `str` | `string` | Логическое имя зависимости |
| 7 | Host | `string` | `String` | `str` | `string` | Хост endpoint-а |
| 8 | Port | `string` | `String` | `str` | `string` | Порт endpoint-а |
| 9 | Critical | `bool` | `boolean` | `bool` | `bool` | Является ли зависимость критической |
| 10 | LastCheckedAt | `time.Time` | `Instant` | `datetime \| None` | `DateTimeOffset?` | Временная метка последней проверки |
| 11 | Labels | `map[string]string` | `Map<String, String>` | `dict[str, str]` | `Dictionary<string, string>` | Пользовательские метки |

**Детали полей**:

- **Healthy**: три состояния — `true` (здоров), `false` (нездоров),
  `nil`/`null`/`None` (неизвестно, до первой проверки).
- **Status**: типизированная категория из `StatusCategory` (см. раздел 8.4).
- **Detail**: строка конкретной причины, соответствующая значениям detail
  из раздела 6.2.3. Для неизвестного состояния: `"unknown"`.
- **Latency**: длительность последней завершённой проверки здоровья.
  Нулевое значение до первой проверки.
- **Type**: настроенный тип зависимости (`"http"`, `"postgres"` и т.д.).
- **Name**: настроенное логическое имя зависимости.
- **Host**, **Port**: хост и порт endpoint-а.
- **Critical**: была ли зависимость отмечена как критическая.
- **LastCheckedAt**: wall-clock временная метка завершения последней проверки.
  Нулевое значение / `null` / `None` до первой проверки.
- **Labels**: пользовательские метки, настроенные на endpoint-е.
  Пустая карта (не null) если метки не настроены.

### 8.4. Тип `StatusCategory`

`StatusCategory` — это типизированный алиас для строковых значений
категорий статуса. Он оборачивает существующие константы статуса,
используемые системой классификации ошибок (раздел 6.2).

**Значения** (всего 9):

| Значение | Описание |
| --- | --- |
| `ok` | Проверка успешна |
| `timeout` | Проверка превысила таймаут |
| `connection_error` | Ошибка соединения (отказано, сброс, недоступен) |
| `dns_error` | Ошибка DNS-резолвинга |
| `auth_error` | Ошибка аутентификации |
| `tls_error` | Ошибка TLS-рукопожатия |
| `unhealthy` | Зависимость ответила, но сообщила о нездоровом статусе |
| `error` | Неклассифицированная ошибка |
| `unknown` | Проверка ещё не выполнялась (начальное состояние) |

**Реализация по языкам**:

| SDK | Реализация |
| --- | --- |
| Go | `type StatusCategory string` с типизированными константами |
| Java | `enum StatusCategory` с методом `String value()` |
| Python | Алиас типа `str` с константами на уровне модуля |
| C# | `static class StatusCategory` с константами `string` |

Первые 8 значений (`ok` — `error`) являются алиасами существующих
констант, используемых в метриках. Значение `unknown` — новое, добавлено
специально для API `HealthDetails()` для представления состояния
до первой проверки.

### 8.5. Состояние UNKNOWN

Endpoint-ы, ещё не завершившие первую проверку, **включаются**
в результат `HealthDetails()` со следующими значениями:

| Поле | Значение |
| --- | --- |
| Healthy | `nil` / `null` / `None` |
| Status | `"unknown"` |
| Detail | `"unknown"` |
| Latency | ноль |
| LastCheckedAt | нулевое значение / `null` / `None` |
| Type, Name, Host, Port, Critical, Labels | Заполняются из конфигурации |

> Это отличается от `Health()`, который **исключает** endpoint-ы
> в состоянии UNKNOWN.
> Обоснование: `HealthDetails()` предоставляет полное представление
> для страниц статуса; исключение endpoint-ов скрывает важную информацию
> о состоянии запуска.

### 8.6. Поведение жизненного цикла

| Состояние | `HealthDetails()` возвращает |
| --- | --- |
| До `Start()` | `nil` / `null` / empty (endpoint-ы не зарегистрированы) |
| После `Start()`, до первой проверки | Все endpoint-ы в состоянии UNKNOWN (раздел 8.5) |
| Работа | Текущее состояние всех endpoint-ов |
| После `Stop()` | Последнее известное состояние (замороженный снимок) |

### 8.7. Источники данных

Все данные берутся из существующего внутреннего состояния планировщика.
Следующие поля должны сохраняться в состоянии endpoint-а
во время `executeCheck()`:

| Поле | Источник | Когда сохраняется |
| --- | --- | --- |
| Healthy | Существующее поле `healthy` | Уже сохраняется |
| Status | `classifyError(err).Category` | После каждой проверки |
| Detail | `classifyError(err).Detail` | После каждой проверки |
| Latency | Длительность проверки | После каждой проверки |
| LastCheckedAt | `time.Now()` / `Instant.now()` / `datetime.now(UTC)` | После каждой проверки |
| Type | `dependency.Type` | При создании состояния |
| Name | `dependency.Name` | При создании состояния |
| Host | `endpoint.Host` | При создании состояния |
| Port | `endpoint.Port` | При создании состояния |
| Critical | `dependency.Critical` | При создании состояния |
| Labels | `endpoint.Labels` | При создании состояния |

### 8.8. Потокобезопасность

`HealthDetails()` **должен быть безопасен** для конкурентного вызова
из нескольких горутин / потоков. Реализация следует существующему паттерну
`Health()`:

1. Захватить мьютекс планировщика для доступа к карте состояний.
2. Итерировать состояния, блокируя каждое состояние endpoint-а индивидуально.
3. Скопировать значения в результирующую карту под блокировкой.
4. Вернуть результирующую карту (вызывающий владеет ей; модификации
   не влияют на внутреннее состояние).

### 8.9. JSON-сериализация

`EndpointStatus` должен сериализоваться в JSON без дополнительной работы.
Все SDK должны использовать имена полей в **snake_case** в JSON-выводе.

**Каноническй формат JSON**:

```json
{
  "postgres-main:pg.svc:5432": {
    "healthy": true,
    "status": "ok",
    "detail": "ok",
    "latency_ms": 2.3,
    "type": "postgres",
    "name": "postgres-main",
    "host": "pg.svc",
    "port": "5432",
    "critical": true,
    "last_checked_at": "2026-02-14T10:30:00Z",
    "labels": {"role": "primary"}
  },
  "redis-cache:redis.svc:6379": {
    "healthy": null,
    "status": "unknown",
    "detail": "unknown",
    "latency_ms": 0,
    "type": "redis",
    "name": "redis-cache",
    "host": "redis.svc",
    "port": "6379",
    "critical": false,
    "last_checked_at": null,
    "labels": {}
  }
}
```

**Правила сериализации полей**:

| Поле | Тип JSON | Примечания |
| --- | --- | --- |
| `healthy` | `boolean` или `null` | `null` для состояния UNKNOWN |
| `status` | `string` | Одно из 9 значений StatusCategory |
| `detail` | `string` | Значение detail из классификации ошибок |
| `latency_ms` | `number` | Миллисекунды как float (например, `2.3`) |
| `type` | `string` | Тип зависимости |
| `name` | `string` | Имя зависимости |
| `host` | `string` | Хост endpoint-а |
| `port` | `string` | Порт endpoint-а (всегда строка) |
| `critical` | `boolean` | Никогда не null |
| `last_checked_at` | `string` или `null` | Формат ISO 8601 UTC; `null` до первой проверки |
| `labels` | `object` | Пустой `{}` если нет меток (никогда не null) |

### 8.10. Обратная совместимость

- `Health()` остаётся без изменений во всех SDK.
- `EndpointStatus` — новый экспортируемый тип.
- `StatusCategory` — новый экспортируемый тип.
- `HealthDetails()` — новый метод на существующих фасадах.
- Поведение метрик не меняется.
- Конфигурация не меняется.
