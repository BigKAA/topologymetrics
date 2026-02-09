"""Базовые абстракции: Dependency, Endpoint, CheckConfig, DependencyType."""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from enum import StrEnum


class DependencyType(StrEnum):
    """Тип зависимости."""

    HTTP = "http"
    GRPC = "grpc"
    TCP = "tcp"
    POSTGRES = "postgres"
    MYSQL = "mysql"
    REDIS = "redis"
    AMQP = "amqp"
    KAFKA = "kafka"


# Значения по умолчанию из спецификации.
DEFAULT_CHECK_INTERVAL: float = 15.0
DEFAULT_TIMEOUT: float = 5.0
DEFAULT_INITIAL_DELAY: float = 5.0
DEFAULT_FAILURE_THRESHOLD: int = 1
DEFAULT_SUCCESS_THRESHOLD: int = 1

# Границы валидации.
MIN_CHECK_INTERVAL: float = 1.0
MAX_CHECK_INTERVAL: float = 300.0
MIN_TIMEOUT: float = 1.0
MAX_TIMEOUT: float = 60.0
MIN_INITIAL_DELAY: float = 0.0
MAX_INITIAL_DELAY: float = 300.0
MIN_THRESHOLD: int = 1
MAX_THRESHOLD: int = 100

_NAME_PATTERN = re.compile(r"^[a-zA-Z][a-zA-Z0-9_-]{0,62}$")

LABEL_NAME_PATTERN = re.compile(r"^[a-zA-Z_][a-zA-Z0-9_]*$")

RESERVED_LABELS: frozenset[str] = frozenset(
    {"name", "dependency", "type", "host", "port", "critical"}
)


def validate_label_name(label: str) -> None:
    """Проверяет имя произвольной метки."""
    if not LABEL_NAME_PATTERN.match(label):
        msg = f"invalid label name {label!r}: must match [a-zA-Z_][a-zA-Z0-9_]*"
        raise ValueError(msg)
    if label in RESERVED_LABELS:
        msg = f"label name {label!r} is reserved"
        raise ValueError(msg)


def validate_labels(labels: dict[str, str]) -> None:
    """Проверяет все произвольные метки."""
    for key in labels:
        validate_label_name(key)


@dataclass
class CheckConfig:
    """Конфигурация проверки зависимости."""

    interval: float = DEFAULT_CHECK_INTERVAL
    timeout: float = DEFAULT_TIMEOUT
    initial_delay: float = DEFAULT_INITIAL_DELAY
    failure_threshold: int = DEFAULT_FAILURE_THRESHOLD
    success_threshold: int = DEFAULT_SUCCESS_THRESHOLD

    def validate(self) -> None:
        """Проверяет корректность конфигурации."""
        if not MIN_CHECK_INTERVAL <= self.interval <= MAX_CHECK_INTERVAL:
            msg = f"interval must be between {MIN_CHECK_INTERVAL} and {MAX_CHECK_INTERVAL}"
            raise ValueError(msg)
        if not MIN_TIMEOUT <= self.timeout <= MAX_TIMEOUT:
            msg = f"timeout must be between {MIN_TIMEOUT} and {MAX_TIMEOUT}"
            raise ValueError(msg)
        if not MIN_INITIAL_DELAY <= self.initial_delay <= MAX_INITIAL_DELAY:
            msg = f"initial_delay must be between {MIN_INITIAL_DELAY} and {MAX_INITIAL_DELAY}"
            raise ValueError(msg)
        if not MIN_THRESHOLD <= self.failure_threshold <= MAX_THRESHOLD:
            msg = f"failure_threshold must be between {MIN_THRESHOLD} and {MAX_THRESHOLD}"
            raise ValueError(msg)
        if not MIN_THRESHOLD <= self.success_threshold <= MAX_THRESHOLD:
            msg = f"success_threshold must be between {MIN_THRESHOLD} and {MAX_THRESHOLD}"
            raise ValueError(msg)


def default_check_config() -> CheckConfig:
    """Возвращает конфигурацию со значениями по умолчанию."""
    return CheckConfig()


@dataclass
class Endpoint:
    """Адрес зависимости."""

    host: str
    port: str
    labels: dict[str, str] = field(default_factory=dict)


def bool_to_yes_no(value: bool) -> str:
    """Конвертирует bool в 'yes'/'no'."""
    return "yes" if value else "no"


@dataclass
class Dependency:
    """Описание зависимости."""

    name: str
    type: DependencyType
    critical: bool
    endpoints: list[Endpoint] = field(default_factory=list)
    config: CheckConfig = field(default_factory=default_check_config)

    def validate(self) -> None:
        """Проверяет корректность зависимости."""
        validate_name(self.name)
        if not self.endpoints:
            msg = "at least one endpoint required"
            raise ValueError(msg)
        for ep in self.endpoints:
            validate_labels(ep.labels)
        self.config.validate()


def validate_name(name: str) -> None:
    """Проверяет имя зависимости."""
    if not _NAME_PATTERN.match(name):
        msg = f"invalid dependency name {name!r}: must match [a-zA-Z][a-zA-Z0-9_-]{{{{0,62}}}}"
        raise ValueError(msg)
