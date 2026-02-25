*[English version](api-reference.md)*

# Python SDK: Справочник API

## DependencyHealth

Основной класс SDK. Управляет мониторингом состояния зависимостей, экспортом
метрик и жизненным циклом динамических эндпоинтов.

### Конструктор

```python
DependencyHealth(
    name: str,
    group: str,
    *specs: _DependencySpec,
    check_interval: timedelta | None = None,
    timeout: timedelta | None = None,
    registry: CollectorRegistry | None = None,
    log: logging.Logger | None = None,
)
```

| Параметр | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `name` | `str` | — | Имя приложения (или env `DEPHEALTH_NAME`). Формат: `[a-z][a-z0-9-]{0,62}` |
| `group` | `str` | — | Группа приложения (или env `DEPHEALTH_GROUP`). Тот же формат |
| `*specs` | `_DependencySpec` | — | Спецификации зависимостей (из фабричных функций) |
| `check_interval` | `timedelta \| None` | `15s` | Глобальный интервал проверок |
| `timeout` | `timedelta \| None` | `5s` | Глобальный тайм-аут проверок |
| `registry` | `CollectorRegistry \| None` | default | Реестр Prometheus |
| `log` | `Logger \| None` | `dephealth` | Логгер |

### Методы жизненного цикла

#### `start() -> None` (async)

Запуск мониторинга в режиме asyncio. Создаёт один `asyncio.Task` на каждый
эндпоинт.

#### `stop() -> None` (async)

Остановка всех задач мониторинга asyncio.

#### `start_sync() -> None`

Запуск мониторинга в режиме threading. Создаёт один daemon `Thread` на каждый
эндпоинт.

#### `stop_sync() -> None`

Остановка всех потоков мониторинга.

### Методы запроса состояния

#### `health() -> dict[str, bool]`

Текущее состояние здоровья, сгруппированное по имени зависимости. Зависимость
считается здоровой, если хотя бы один её эндпоинт здоров.

#### `health_details() -> dict[str, EndpointStatus]`

Детальный статус каждого эндпоинта. Ключи в формате `"dependency:host:port"`.

### Динамическое управление эндпоинтами

Добавлено в v0.6.0. Все методы требуют запущенного планировщика (через
`start()` или `start_sync()`).

#### `add_endpoint(dep_name, dep_type, critical, endpoint, checker) -> None` (async)

Добавление нового мониторируемого эндпоинта в рантайме.

```python
async def add_endpoint(
    self,
    dep_name: str,
    dep_type: DependencyType,
    critical: bool,
    endpoint: Endpoint,
    checker: HealthChecker,
) -> None
```

| Параметр | Тип | Описание |
| --- | --- | --- |
| `dep_name` | `str` | Имя зависимости. Формат: `[a-z][a-z0-9-]{0,62}` |
| `dep_type` | `DependencyType` | Тип зависимости (`HTTP`, `POSTGRES` и т.д.) |
| `critical` | `bool` | Критичность зависимости |
| `endpoint` | `Endpoint` | Эндпоинт для мониторинга |
| `checker` | `HealthChecker` | Реализация проверки здоровья |

**Идемпотентность:** возвращает управление без ошибки, если эндпоинт уже существует.

**Исключения:**

- `ValueError` — некорректный `dep_name`, `dep_type`, или пустой `host`/`port`
- `RuntimeError` — планировщик не запущен или уже остановлен

#### `remove_endpoint(dep_name, host, port) -> None` (async)

Удаление мониторируемого эндпоинта. Отменяет задачу проверки и удаляет
все метрики Prometheus для эндпоинта.

```python
async def remove_endpoint(
    self,
    dep_name: str,
    host: str,
    port: str,
) -> None
```

**Идемпотентность:** возвращает управление без ошибки, если эндпоинт не найден.

**Исключения:** `RuntimeError` — планировщик не запущен или уже остановлен.

#### `update_endpoint(dep_name, old_host, old_port, new_endpoint, checker) -> None` (async)

Атомарная замена эндпоинта. Удаляет старый эндпоинт (отменяет задачу,
удаляет метрики) и добавляет новый.

```python
async def update_endpoint(
    self,
    dep_name: str,
    old_host: str,
    old_port: str,
    new_endpoint: Endpoint,
    checker: HealthChecker,
) -> None
```

**Исключения:**

- `EndpointNotFoundError` — старый эндпоинт не найден
- `ValueError` — некорректный новый эндпоинт (пустой `host`/`port`, зарезервированные метки)
- `RuntimeError` — планировщик не запущен или уже остановлен

#### `add_endpoint_sync(dep_name, dep_type, critical, endpoint, checker) -> None`

Синхронный вариант `add_endpoint()` для режима threading.

#### `remove_endpoint_sync(dep_name, host, port) -> None`

Синхронный вариант `remove_endpoint()` для режима threading.

#### `update_endpoint_sync(dep_name, old_host, old_port, new_endpoint, checker) -> None`

Синхронный вариант `update_endpoint()` для режима threading.

---

## Фабричные функции

Фабричные функции создают спецификации зависимостей для конструктора.

### `http_check(name, *, url, critical, ...)`

HTTP-проверка.

| Параметр | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `name` | `str` | — | Имя зависимости |
| `url` | `str` | `""` | URL (host/port извлекаются автоматически) |
| `host` | `str` | `""` | Хост (если `url` не указан) |
| `port` | `str` | `"80"` | Порт |
| `health_path` | `str` | `"/health"` | Путь эндпоинта проверки |
| `tls` | `bool` | `False` | Включить TLS |
| `tls_skip_verify` | `bool` | `False` | Пропустить проверку TLS-сертификата |
| `headers` | `dict[str, str] \| None` | `None` | Кастомные HTTP-заголовки |
| `bearer_token` | `str \| None` | `None` | Bearer-токен |
| `basic_auth` | `tuple[str, str] \| None` | `None` | Basic auth `(user, pass)` |
| `critical` | `bool` | — | Критичность зависимости |
| `timeout` | `timedelta \| None` | global | Тайм-аут для зависимости |
| `interval` | `timedelta \| None` | global | Интервал для зависимости |
| `labels` | `dict[str, str] \| None` | `None` | Кастомные метки |

### `grpc_check(name, *, critical, ...)`

gRPC-проверка.

### `tcp_check(name, *, host, port, critical, ...)`

TCP-проверка.

### `postgres_check(name, *, url, critical, ...)`

PostgreSQL-проверка. Поддержка connection pool через параметр `pool`.

### `mysql_check(name, *, url, critical, ...)`

MySQL-проверка. Поддержка connection pool через параметр `pool`.

### `redis_check(name, *, url, critical, ...)`

Redis-проверка. Поддержка существующего клиента через параметр `client`.

### `amqp_check(name, *, url, critical, ...)`

AMQP (RabbitMQ) проверка.

### `kafka_check(name, *, url, critical, ...)`

Kafka-проверка.

---

## Типы

### `Endpoint`

```python
@dataclass
class Endpoint:
    host: str
    port: str
    labels: dict[str, str] = field(default_factory=dict)
```

### `DependencyType`

Enum: `HTTP`, `GRPC`, `TCP`, `POSTGRES`, `MYSQL`, `REDIS`, `AMQP`, `KAFKA`.

### `EndpointStatus`

```python
@dataclass(frozen=True)
class EndpointStatus:
    healthy: bool | None
    status: str
    detail: str
    latency: float
    type: str
    name: str
    host: str
    port: str
    critical: bool
    last_checked_at: datetime | None
    labels: dict[str, str]
```

Методы:

- `latency_millis() -> float` — задержка в миллисекундах
- `to_dict() -> dict` — словарь для JSON-сериализации

### `HealthChecker`

```python
class HealthChecker(ABC):
    @abstractmethod
    async def check(self, endpoint: Endpoint) -> None:
        """Бросить CheckError при ошибке; вернуть None при успехе."""
```

---

## Исключения

| Исключение | Описание |
| --- | --- |
| `CheckError` | Базовый класс ошибок проверки |
| `CheckTimeoutError` | Тайм-аут проверки |
| `CheckConnectionRefusedError` | Соединение отклонено |
| `CheckDnsError` | Ошибка DNS-разрешения |
| `CheckTlsError` | Ошибка TLS-рукопожатия |
| `CheckAuthError` | Ошибка аутентификации/авторизации |
| `UnhealthyError` | Эндпоинт сообщил о нездоровом статусе |
| `EndpointNotFoundError` | Целевой эндпоинт не найден при динамическом обновлении/удалении (v0.6.0) |
