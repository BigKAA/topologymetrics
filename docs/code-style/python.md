*[Русская версия](python.ru.md)*

# Code Style Guide: Python SDK

This document describes code style conventions for the Python SDK (`sdk-python/`).
See also: [General Principles](overview.md) | [Testing](testing.md)

## Naming Conventions

### Modules and Packages

- `snake_case` for modules and packages
- Short, descriptive names

```python
dephealth/          # core package
dephealth_fastapi/  # FastAPI integration (separate package)

# Modules
checker.py          # good
health_checker.py   # good if needed for clarity

HealthChecker.py    # bad — not snake_case
```

### Classes

- `PascalCase` for all classes
- Exception classes end with `Error`

```python
class HealthChecker(Protocol): ...
class Dependency: ...
class Endpoint: ...
class CheckScheduler: ...

# Exceptions
class CheckError(Exception): ...
class CheckTimeoutError(CheckError): ...
class CheckConnectionRefusedError(CheckError): ...
```

### Functions and Methods

- `snake_case` for all functions and methods
- Verb-first for actions, noun for getters

```python
async def check(self, endpoint: Endpoint) -> None: ...
def checker_type(self) -> str: ...
def parse_url(raw: str) -> Endpoint: ...

# Private (single underscore)
def _sanitize_url(self, raw: str) -> str: ...
def _schedule_check(self, dep: Dependency) -> None: ...
```

### Constants

- `UPPER_SNAKE_CASE` at module level

```python
DEFAULT_CHECK_INTERVAL = 15.0
DEFAULT_TIMEOUT = 5.0
DEFAULT_FAILURE_THRESHOLD = 1
DEFAULT_SUCCESS_THRESHOLD = 1
```

### Variables

- `snake_case` for all variables
- Private instance attributes: single underscore prefix `_`

```python
class CheckScheduler:
    def __init__(self) -> None:
        self._dependencies: list[Dependency] = []
        self._running = False
        self._tasks: list[asyncio.Task[None]] = []
```

## Package Structure

```text
sdk-python/
├── dephealth/
│   ├── __init__.py           # public API re-exports
│   ├── py.typed              # PEP 561 marker
│   ├── api.py                # convenience constructors (postgres_check, redis_check, ...)
│   ├── dependency.py         # Dependency, Endpoint dataclasses
│   ├── checker.py            # HealthChecker Protocol, exceptions
│   ├── scheduler.py          # asyncio-based scheduler
│   ├── parser.py             # URL/params parser
│   ├── metrics.py            # prometheus_client gauges, histograms
│   └── checks/
│       ├── __init__.py
│       ├── http.py           # HTTPChecker
│       ├── grpc.py           # GRPCChecker
│       ├── tcp.py            # TCPChecker
│       ├── postgres.py       # PostgresChecker
│       ├── redis.py          # RedisChecker
│       ├── amqp.py           # AMQPChecker
│       └── kafka.py          # KafkaChecker
│
└── dephealth_fastapi/
    ├── __init__.py
    ├── middleware.py          # DepHealthMiddleware
    └── lifespan.py           # dephealth_lifespan()
```

## Error Handling

### Exception Hierarchy

```python
class CheckError(Exception):
    """Базовая ошибка проверки зависимости."""

class CheckTimeoutError(CheckError):
    """Таймаут при проверке."""

class CheckConnectionRefusedError(CheckError):
    """Соединение отклонено."""

class UnhealthyError(CheckError):
    """Зависимость доступна, но нездорова."""
```

### Rules

- Checkers raise `CheckError` subtypes on failure, return `None` on success
- Configuration errors: raise `ValueError` or `TypeError` immediately
- Never use bare `except:` — always catch specific exceptions
- Use `raise ... from cause` to preserve exception chains

```python
# Good — specific exception with context
async def check(self, endpoint: Endpoint) -> None:
    try:
        async with asyncio.timeout(self._timeout):
            resp = await self._client.get(url)
    except TimeoutError as exc:
        raise CheckTimeoutError(
            f"HTTP check {endpoint.host}:{endpoint.port} timed out"
        ) from exc
    except ConnectionError as exc:
        raise CheckConnectionRefusedError(
            f"HTTP check {endpoint.host}:{endpoint.port} refused"
        ) from exc

    if resp.status_code >= 300:
        raise UnhealthyError(
            f"HTTP check {endpoint.host}:{endpoint.port}: status {resp.status_code}"
        )

# Bad — loses context
except Exception:
    raise CheckError("failed")
```

## Docstrings

Use **Google style** docstrings:

```python
class HealthChecker(Protocol):
    """Протокол проверки здоровья зависимости.

    Реализации должны быть async-safe. Каждый вызов check() может
    происходить параллельно для разных эндпоинтов.
    """

    async def check(self, endpoint: Endpoint) -> None:
        """Проверяет зависимость.

        Args:
            endpoint: Эндпоинт для проверки.

        Raises:
            CheckTimeoutError: Если проверка превысила таймаут.
            CheckConnectionRefusedError: Если соединение отклонено.
            UnhealthyError: Если зависимость доступна, но нездорова.
        """
        ...
```

Rules:

- First line: summary in Russian
- `Args`, `Returns`, `Raises` sections as needed
- Document all public classes, functions, and methods
- Use type hints instead of documenting types in docstrings

## Type Hints

All code must pass `mypy --strict`. This means:

- All function parameters and return types are annotated
- All class attributes are annotated
- No `# type: ignore` without a comment explaining why

```python
from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class Endpoint:
    """Один эндпоинт зависимости."""

    host: str
    port: int
    metadata: dict[str, str] = field(default_factory=dict)

    def __post_init__(self) -> None:
        if not self.host:
            raise ValueError("host must not be empty")
        if self.port <= 0 or self.port > 65535:
            raise ValueError(f"port must be 1-65535, got: {self.port}")
```

### Common patterns

```python
# Optional values
from typing import Optional

def parse_url(raw: str, default_port: Optional[int] = None) -> Endpoint: ...

# Protocols instead of ABCs (for structural typing)
from typing import Protocol

class HealthChecker(Protocol):
    async def check(self, endpoint: Endpoint) -> None: ...
    def checker_type(self) -> str: ...

# Type aliases for clarity
DependencyName = str
CheckerFactory = Callable[[dict[str, Any]], HealthChecker]
```

## Async/Await Patterns

### Never Block the Event Loop

All checkers are `async`. Never use blocking I/O:

```python
# Good — async I/O
async def check(self, endpoint: Endpoint) -> None:
    async with asyncpg.create_pool(dsn) as pool:
        await pool.fetchval("SELECT 1")

# Bad — blocks event loop
def check(self, endpoint: Endpoint) -> None:
    import psycopg2
    conn = psycopg2.connect(dsn)  # blocking!
    conn.cursor().execute("SELECT 1")
```

### Use asyncio.timeout (Python 3.11+)

```python
async def check(self, endpoint: Endpoint) -> None:
    try:
        async with asyncio.timeout(self._timeout):
            await self._do_check(endpoint)
    except TimeoutError as exc:
        raise CheckTimeoutError(...) from exc
```

### Graceful Shutdown

```python
class CheckScheduler:
    async def start(self) -> None:
        """Запуск планировщика."""
        for dep in self._dependencies:
            task = asyncio.create_task(self._check_loop(dep))
            self._tasks.append(task)

    async def stop(self) -> None:
        """Остановка планировщика с ожиданием завершения."""
        for task in self._tasks:
            task.cancel()
        await asyncio.gather(*self._tasks, return_exceptions=True)
        self._tasks.clear()
```

## `__all__` for Public API

Every `__init__.py` must define `__all__` to explicitly control the public API:

```python
# dephealth/__init__.py
from dephealth.api import postgres_check, redis_check, http_check
from dephealth.checker import CheckError, CheckTimeoutError
from dephealth.dependency import Dependency, Endpoint

__all__ = [
    "Dependency",
    "Endpoint",
    "CheckError",
    "CheckTimeoutError",
    "postgres_check",
    "redis_check",
    "http_check",
]
```

## Linters

### ruff

Fast linter and formatter. Configuration in `sdk-python/pyproject.toml`:

Key rules:

- `E` / `W` — pycodestyle errors and warnings
- `F` — pyflakes
- `I` — isort (import sorting)
- `UP` — pyupgrade
- `B` — flake8-bugbear
- `SIM` — flake8-simplify
- `RUF` — ruff-specific rules

### mypy

Strict mode: `sdk-python/pyproject.toml` section `[tool.mypy]`.

Key settings:

- `strict = true`
- `warn_return_any = true`
- `disallow_untyped_defs = true`

### Running

```bash
cd sdk-python && make lint    # ruff check + mypy in Docker
cd sdk-python && make fmt     # ruff format
```

## Additional Conventions

- **Python version**: 3.12+
- **`from __future__ import annotations`** in every file (PEP 563)
- **Dataclasses with `frozen=True`** for immutable models (`Dependency`, `Endpoint`)
- **`py.typed` marker** in package root (PEP 561)
- **No mutable default arguments**: use `field(default_factory=...)` in dataclasses
- **f-strings** for string formatting (not `%` or `.format()`)
- **Absolute imports** only (no relative imports like `from . import`)
