*[English version](python.md)*

# Code Style Guide: Python SDK

Этот документ описывает соглашения по стилю кода для Python SDK (`sdk-python/`).
См. также: [Общие принципы](overview.ru.md) | [Тестирование](testing.ru.md)

## Соглашения об именовании

### Модули и пакеты

- `snake_case` для модулей и пакетов
- Короткие, описательные имена

```python
dephealth/          # основной пакет
dephealth_fastapi/  # FastAPI-интеграция (отдельный пакет)

# Модули
checker.py          # хорошо
health_checker.py   # хорошо, если нужно для ясности

HealthChecker.py    # плохо — не snake_case
```

### Классы

- `PascalCase` для всех классов
- Классы исключений заканчиваются на `Error`

```python
class HealthChecker(Protocol): ...
class Dependency: ...
class Endpoint: ...
class CheckScheduler: ...

# Исключения
class CheckError(Exception): ...
class CheckTimeoutError(CheckError): ...
class CheckConnectionRefusedError(CheckError): ...
```

### Функции и методы

- `snake_case` для всех функций и методов
- Глагол в начале для действий, существительное для геттеров

```python
async def check(self, endpoint: Endpoint) -> None: ...
def checker_type(self) -> str: ...
def parse_url(raw: str) -> Endpoint: ...

# Приватные (одинарное подчёркивание)
def _sanitize_url(self, raw: str) -> str: ...
def _schedule_check(self, dep: Dependency) -> None: ...
```

### Константы

- `UPPER_SNAKE_CASE` на уровне модуля

```python
DEFAULT_CHECK_INTERVAL = 15.0
DEFAULT_TIMEOUT = 5.0
DEFAULT_FAILURE_THRESHOLD = 1
DEFAULT_SUCCESS_THRESHOLD = 1
```

### Переменные

- `snake_case` для всех переменных
- Приватные атрибуты экземпляра: префикс одинарного подчёркивания `_`

```python
class CheckScheduler:
    def __init__(self) -> None:
        self._dependencies: list[Dependency] = []
        self._running = False
        self._tasks: list[asyncio.Task[None]] = []
```

## Структура пакетов

```text
sdk-python/
├── dephealth/
│   ├── __init__.py           # реэкспорт публичного API
│   ├── py.typed              # маркер PEP 561
│   ├── api.py                # удобные конструкторы (postgres_check, redis_check, ...)
│   ├── dependency.py         # Dependency, Endpoint dataclasses
│   ├── checker.py            # HealthChecker Protocol, исключения
│   ├── scheduler.py          # asyncio-планировщик
│   ├── parser.py             # парсер URL/параметров
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

## Обработка ошибок

### Иерархия исключений

```python
class CheckError(Exception):
    """Base exception for dependency check failures."""

class CheckTimeoutError(CheckError):
    """Check timed out."""

class CheckConnectionRefusedError(CheckError):
    """Connection refused."""

class UnhealthyError(CheckError):
    """Dependency is reachable but unhealthy."""
```

### Правила

- Checker-ы бросают подтипы `CheckError` при неудаче, возвращают `None` при успехе
- Ошибки конфигурации: бросать `ValueError` или `TypeError` немедленно
- Никогда не используйте голый `except:` — всегда ловите конкретные исключения
- Используйте `raise ... from cause` для сохранения цепочки исключений

```python
# Хорошо — конкретное исключение с контекстом
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

# Плохо — теряет контекст
except Exception:
    raise CheckError("failed")
```

## Docstrings

Используйте **Google style** docstrings:

```python
class HealthChecker(Protocol):
    """Protocol for dependency health checks.

    Implementations must be async-safe. Each check() call may
    run concurrently for different endpoints.
    """

    async def check(self, endpoint: Endpoint) -> None:
        """Check the dependency.

        Args:
            endpoint: The endpoint to check.

        Raises:
            CheckTimeoutError: If the check exceeded the timeout.
            CheckConnectionRefusedError: If the connection was refused.
            UnhealthyError: If the dependency is reachable but unhealthy.
        """
        ...
```

Правила:

- Первая строка: краткое описание на английском
- Секции `Args`, `Returns`, `Raises` по необходимости
- Документируйте все публичные классы, функции и методы
- Используйте type hints вместо документирования типов в docstrings

## Type Hints

Весь код должен проходить `mypy --strict`. Это означает:

- Все параметры функций и типы возвращаемых значений аннотированы
- Все атрибуты классов аннотированы
- Нет `# type: ignore` без комментария, объясняющего причину

```python
from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class Endpoint:
    """A single dependency endpoint."""

    host: str
    port: str
    labels: dict[str, str] = field(default_factory=dict)
```

### Типичные паттерны

```python
# Необязательные значения
from typing import Optional

def parse_url(raw: str, default_port: Optional[str] = None) -> Endpoint: ...

# Protocol вместо ABC (структурная типизация)
from typing import Protocol

class HealthChecker(Protocol):
    async def check(self, endpoint: Endpoint) -> None: ...
    def checker_type(self) -> str: ...

# Псевдонимы типов для ясности
DependencyName = str
CheckerFactory = Callable[[dict[str, Any]], HealthChecker]
```

## Async/Await паттерны

### Никогда не блокировать event loop

Все checker-ы — `async`. Никогда не используйте блокирующий I/O:

```python
# Хорошо — async I/O
async def check(self, endpoint: Endpoint) -> None:
    async with asyncpg.create_pool(dsn) as pool:
        await pool.fetchval("SELECT 1")

# Плохо — блокирует event loop
def check(self, endpoint: Endpoint) -> None:
    import psycopg2
    conn = psycopg2.connect(dsn)  # блокирующий вызов!
    conn.cursor().execute("SELECT 1")
```

### Используйте asyncio.timeout (Python 3.11+)

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
        """Start the scheduler."""
        for dep in self._dependencies:
            task = asyncio.create_task(self._check_loop(dep))
            self._tasks.append(task)

    async def stop(self) -> None:
        """Stop the scheduler and wait for completion."""
        for task in self._tasks:
            task.cancel()
        await asyncio.gather(*self._tasks, return_exceptions=True)
        self._tasks.clear()
```

## `__all__` для публичного API

Каждый `__init__.py` должен определять `__all__` для явного контроля публичного API:

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

## Линтеры

### ruff

Быстрый линтер и форматтер. Конфигурация в `sdk-python/pyproject.toml`:

Основные правила:

- `E` / `W` — ошибки и предупреждения pycodestyle
- `F` — pyflakes
- `I` — isort (сортировка импортов)
- `UP` — pyupgrade
- `B` — flake8-bugbear
- `SIM` — flake8-simplify
- `RUF` — ruff-специфичные правила

### mypy

Строгий режим: `sdk-python/pyproject.toml` секция `[tool.mypy]`.

Основные настройки:

- `strict = true`
- `warn_return_any = true`
- `disallow_untyped_defs = true`

### Запуск

```bash
cd sdk-python && make lint    # ruff check + mypy в Docker
cd sdk-python && make fmt     # ruff format
```

## Дополнительные соглашения

- **Версия Python**: 3.12+
- **`from __future__ import annotations`** в каждом файле (PEP 563)
- **Dataclasses с `frozen=True`** для неизменяемых моделей (`Dependency`, `Endpoint`)
- **Маркер `py.typed`** в корне пакета (PEP 561)
- **Нет изменяемых аргументов по умолчанию**: используйте `field(default_factory=...)` в dataclasses
- **f-strings** для форматирования строк (не `%` или `.format()`)
- **Только абсолютные импорты** (без относительных типа `from . import`)
