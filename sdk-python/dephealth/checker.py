"""Интерфейс HealthChecker и ошибки проверок."""

from __future__ import annotations

from typing import Protocol

from dephealth.dependency import Endpoint


class CheckError(Exception):
    """Базовая ошибка проверки зависимости."""


class CheckTimeoutError(CheckError):
    """Таймаут при проверке."""


class CheckConnectionRefusedError(CheckError):
    """Соединение отклонено."""


class UnhealthyError(CheckError):
    """Зависимость доступна, но нездорова."""


class HealthChecker(Protocol):
    """Протокол проверки здоровья зависимости."""

    async def check(self, endpoint: Endpoint) -> None:
        """Проверяет зависимость. Бросает CheckError при неудаче."""
        ...

    def checker_type(self) -> str:
        """Возвращает тип проверки (http, grpc, tcp, ...)."""
        ...
