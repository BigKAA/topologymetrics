*[Русская версия](api-reference.ru.md)*

# Python SDK: API Reference

## DependencyHealth

Main SDK class. Manages dependency health monitoring, metrics export, and
dynamic endpoint lifecycle.

### Constructor

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

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `name` | `str` | — | Application name (or `DEPHEALTH_NAME` env var). Must match `[a-z][a-z0-9-]{0,62}` |
| `group` | `str` | — | Application group (or `DEPHEALTH_GROUP` env var). Same format as `name` |
| `*specs` | `_DependencySpec` | — | Dependency specifications from factory functions |
| `check_interval` | `timedelta \| None` | `15s` | Global check interval |
| `timeout` | `timedelta \| None` | `5s` | Global check timeout |
| `registry` | `CollectorRegistry \| None` | default | Prometheus registry |
| `log` | `Logger \| None` | `dephealth` | Logger instance |

### Lifecycle Methods

#### `start() -> None` (async)

Start health monitoring in asyncio mode. Creates one `asyncio.Task` per
endpoint.

#### `stop() -> None` (async)

Stop all asyncio monitoring tasks.

#### `start_sync() -> None`

Start health monitoring in threading mode. Creates one daemon `Thread` per
endpoint.

#### `stop_sync() -> None`

Stop all monitoring threads.

### Query Methods

#### `health() -> dict[str, bool]`

Return current health status grouped by dependency name. A dependency is
healthy if at least one of its endpoints is healthy.

#### `health_details() -> dict[str, EndpointStatus]`

Return detailed status for every endpoint. Keys use `"dependency:host:port"`
format.

### Dynamic Endpoint Management

Added in v0.6.0. All methods require the scheduler to be started (via
`start()` or `start_sync()`).

#### `add_endpoint(dep_name, dep_type, critical, endpoint, checker) -> None` (async)

Add a new monitored endpoint at runtime.

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

| Parameter | Type | Description |
| --- | --- | --- |
| `dep_name` | `str` | Dependency name. Must match `[a-z][a-z0-9-]{0,62}` |
| `dep_type` | `DependencyType` | Dependency type (`HTTP`, `POSTGRES`, etc.) |
| `critical` | `bool` | Whether the dependency is critical |
| `endpoint` | `Endpoint` | Endpoint to monitor |
| `checker` | `HealthChecker` | Health checker implementation |

**Idempotent:** returns silently if the endpoint already exists.

**Raises:**

- `ValueError` — invalid `dep_name`, `dep_type`, or empty `host`/`port`
- `RuntimeError` — scheduler not started or already stopped

#### `remove_endpoint(dep_name, host, port) -> None` (async)

Remove a monitored endpoint at runtime. Cancels the check task and deletes
all Prometheus metrics for the endpoint.

```python
async def remove_endpoint(
    self,
    dep_name: str,
    host: str,
    port: str,
) -> None
```

**Idempotent:** returns silently if the endpoint does not exist.

**Raises:** `RuntimeError` — scheduler not started or already stopped.

#### `update_endpoint(dep_name, old_host, old_port, new_endpoint, checker) -> None` (async)

Atomically replace an endpoint. Removes the old endpoint (cancels task,
deletes metrics) and adds the new one.

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

**Raises:**

- `EndpointNotFoundError` — old endpoint does not exist
- `ValueError` — invalid new endpoint (`host`/`port` empty, reserved labels)
- `RuntimeError` — scheduler not started or already stopped

#### `add_endpoint_sync(dep_name, dep_type, critical, endpoint, checker) -> None`

Synchronous variant of `add_endpoint()` for threading mode.

#### `remove_endpoint_sync(dep_name, host, port) -> None`

Synchronous variant of `remove_endpoint()` for threading mode.

#### `update_endpoint_sync(dep_name, old_host, old_port, new_endpoint, checker) -> None`

Synchronous variant of `update_endpoint()` for threading mode.

---

## Factory Functions

Factory functions create dependency specifications for the constructor.

### `http_check(name, *, url, critical, ...)`

Create an HTTP health check.

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `name` | `str` | — | Dependency name |
| `url` | `str` | `""` | HTTP URL (parsed for host/port) |
| `host` | `str` | `""` | Host (if `url` not provided) |
| `port` | `str` | `"80"` | Port |
| `health_path` | `str` | `"/health"` | Health endpoint path |
| `tls` | `bool` | `False` | Enable TLS |
| `tls_skip_verify` | `bool` | `False` | Skip TLS verification |
| `headers` | `dict[str, str] \| None` | `None` | Custom HTTP headers |
| `bearer_token` | `str \| None` | `None` | Bearer token |
| `basic_auth` | `tuple[str, str] \| None` | `None` | Basic auth `(user, pass)` |
| `critical` | `bool` | — | Whether the dependency is critical |
| `timeout` | `timedelta \| None` | global | Per-dependency timeout |
| `interval` | `timedelta \| None` | global | Per-dependency interval |
| `labels` | `dict[str, str] \| None` | `None` | Custom labels |

### `grpc_check(name, *, critical, ...)`

Create a gRPC health check.

### `tcp_check(name, *, host, port, critical, ...)`

Create a TCP health check.

### `postgres_check(name, *, url, critical, ...)`

Create a PostgreSQL health check. Supports connection pool via `pool` parameter.

### `mysql_check(name, *, url, critical, ...)`

Create a MySQL health check. Supports connection pool via `pool` parameter.

### `redis_check(name, *, url, critical, ...)`

Create a Redis health check. Supports existing client via `client` parameter.

### `amqp_check(name, *, url, critical, ...)`

Create an AMQP (RabbitMQ) health check.

### `kafka_check(name, *, url, critical, ...)`

Create a Kafka health check.

---

## Types

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

Methods:

- `latency_millis() -> float` — latency in milliseconds
- `to_dict() -> dict` — JSON-serializable dictionary

### `HealthChecker`

```python
class HealthChecker(ABC):
    @abstractmethod
    async def check(self, endpoint: Endpoint) -> None:
        """Raise CheckError on failure; return None on success."""
```

---

## Exceptions

| Exception | Description |
| --- | --- |
| `CheckError` | Base class for check failures |
| `CheckTimeoutError` | Check timed out |
| `CheckConnectionRefusedError` | Connection refused |
| `CheckDnsError` | DNS resolution failed |
| `CheckTlsError` | TLS handshake failed |
| `CheckAuthError` | Authentication/authorization failed |
| `UnhealthyError` | Endpoint reported unhealthy status |
| `EndpointNotFoundError` | Dynamic update/remove target not found (v0.6.0) |
