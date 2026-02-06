"""Парсер URL, connection string и JDBC для извлечения host/port."""

from __future__ import annotations

from dataclasses import dataclass
from urllib.parse import urlparse

from dephealth.dependency import DependencyType, Endpoint

# Порты по умолчанию для каждой схемы.
DEFAULT_PORTS: dict[str, str] = {
    "postgres": "5432",
    "postgresql": "5432",
    "mysql": "3306",
    "redis": "6379",
    "rediss": "6379",
    "amqp": "5672",
    "amqps": "5671",
    "http": "80",
    "https": "443",
    "grpc": "443",
    "kafka": "9092",
}

# Маппинг схема → тип зависимости.
_SCHEME_TO_TYPE: dict[str, DependencyType] = {
    "postgres": DependencyType.POSTGRES,
    "postgresql": DependencyType.POSTGRES,
    "mysql": DependencyType.MYSQL,
    "redis": DependencyType.REDIS,
    "rediss": DependencyType.REDIS,
    "amqp": DependencyType.AMQP,
    "amqps": DependencyType.AMQP,
    "http": DependencyType.HTTP,
    "https": DependencyType.HTTP,
    "grpc": DependencyType.GRPC,
    "kafka": DependencyType.KAFKA,
}

# JDBC subprotocol → тип зависимости.
_JDBC_TO_TYPE: dict[str, DependencyType] = {
    "postgresql": DependencyType.POSTGRES,
    "mysql": DependencyType.MYSQL,
}


@dataclass
class ParsedConnection:
    """Результат парсинга URL/connection string."""

    host: str
    port: str
    conn_type: DependencyType


def parse_url(raw_url: str) -> list[ParsedConnection]:
    """Парсит URL и возвращает список соединений.

    Поддерживает схемы: postgres://, postgresql://, redis://, rediss://,
    amqp://, amqps://, http://, https://, grpc://, kafka://.
    Для kafka:// поддерживает multi-host: kafka://host1:port1,host2:port2.
    """
    if not raw_url:
        msg = "empty URL"
        raise ValueError(msg)

    parsed = urlparse(raw_url)
    scheme = parsed.scheme.lower()

    if not scheme or ("://" not in raw_url):
        msg = f"missing scheme in URL {raw_url!r}"
        raise ValueError(msg)

    conn_type = _SCHEME_TO_TYPE.get(scheme)
    if conn_type is None:
        msg = f"unsupported URL scheme {scheme!r}"
        raise ValueError(msg)

    default_port = DEFAULT_PORTS.get(scheme, "")

    # Kafka multi-host: kafka://host1:9092,host2:9092
    # Проверяем ДО обращения к parsed.port (запятые ломают urlparse).
    raw_netloc = parsed.netloc
    # Убираем userinfo если есть.
    if "@" in raw_netloc:
        raw_netloc = raw_netloc.split("@", 1)[1]

    if "," in raw_netloc:
        return _parse_multi_host(raw_netloc, default_port, conn_type)

    host = parsed.hostname or ""
    port_str = str(parsed.port) if parsed.port else ""
    port = port_str or default_port

    if not host:
        msg = f"missing host in URL {raw_url!r}"
        raise ValueError(msg)

    _validate_port(port)

    return [ParsedConnection(host=host, port=port, conn_type=conn_type)]


def _parse_multi_host(
    host_part: str, default_port: str, conn_type: DependencyType
) -> list[ParsedConnection]:
    """Парсит multi-host строку (host1:port1,host2:port2)."""
    results: list[ParsedConnection] = []
    for segment in host_part.split(","):
        segment = segment.strip()
        if not segment:
            continue
        host, port = _extract_host_port(segment, default_port)
        results.append(ParsedConnection(host=host, port=port, conn_type=conn_type))
    if not results:
        msg = "no hosts found in multi-host URL"
        raise ValueError(msg)
    return results


def _extract_host_port(segment: str, default_port: str) -> tuple[str, str]:
    """Извлекает host и port из сегмента host:port или [ipv6]:port."""
    # IPv6: [::1]:5432
    if segment.startswith("["):
        bracket_end = segment.find("]")
        if bracket_end == -1:
            msg = f"invalid IPv6 address {segment!r}"
            raise ValueError(msg)
        host = segment[1:bracket_end]
        rest = segment[bracket_end + 1 :]
        port = rest[1:] if rest.startswith(":") else default_port
    elif ":" in segment:
        parts = segment.rsplit(":", 1)
        host = parts[0]
        port = parts[1] if parts[1] else default_port
    else:
        host = segment
        port = default_port

    _validate_port(port)
    return host, port


def parse_connection_string(conn_str: str) -> tuple[str, str]:
    """Парсит key=value connection string (Postgres/MySQL).

    Ищет host и port в строке вида:
    host=localhost port=5432 dbname=mydb user=admin
    """
    if not conn_str:
        msg = "empty connection string"
        raise ValueError(msg)

    pairs = _parse_key_value_pairs(conn_str)

    host = _find_value(pairs, "host", "server")
    port = _find_value(pairs, "port")

    if not host:
        msg = "host not found in connection string"
        raise ValueError(msg)
    if not port:
        msg = "port not found in connection string"
        raise ValueError(msg)

    _validate_port(port)
    return host, port


def _parse_key_value_pairs(conn_str: str) -> dict[str, str]:
    """Парсит key=value пары из строки."""
    pairs: dict[str, str] = {}
    for part in conn_str.split():
        if "=" in part:
            key, _, value = part.partition("=")
            pairs[key.lower().strip()] = value.strip()
    return pairs


def _find_value(pairs: dict[str, str], *keys: str) -> str:
    """Ищет значение по нескольким возможным ключам."""
    for key in keys:
        if key in pairs:
            return pairs[key]
    return ""


def parse_jdbc(jdbc_url: str) -> list[ParsedConnection]:
    """Парсит JDBC URL.

    Поддерживает:
    - jdbc:postgresql://host:port/db
    - jdbc:mysql://host:port/db
    """
    if not jdbc_url:
        msg = "empty JDBC URL"
        raise ValueError(msg)

    if not jdbc_url.startswith("jdbc:"):
        msg = f"invalid JDBC URL {jdbc_url!r}: must start with 'jdbc:'"
        raise ValueError(msg)

    # Убираем jdbc: префикс
    rest = jdbc_url[5:]

    # Определяем subprotocol
    colon_idx = rest.find(":")
    if colon_idx == -1:
        msg = f"invalid JDBC URL {jdbc_url!r}: missing subprotocol"
        raise ValueError(msg)

    subprotocol = rest[:colon_idx].lower()
    conn_type = _JDBC_TO_TYPE.get(subprotocol)
    if conn_type is None:
        msg = f"unsupported JDBC subprotocol {subprotocol!r}"
        raise ValueError(msg)

    # Парсим оставшуюся часть как обычный URL
    inner_url = rest[colon_idx + 1 :]
    parsed = urlparse(inner_url)

    host = parsed.hostname or ""
    port = str(parsed.port) if parsed.port else DEFAULT_PORTS.get(subprotocol, "")

    if not host:
        msg = f"missing host in JDBC URL {jdbc_url!r}"
        raise ValueError(msg)

    _validate_port(port)

    return [ParsedConnection(host=host, port=port, conn_type=conn_type)]


def parse_params(host: str, port: str) -> Endpoint:
    """Создаёт Endpoint из явно заданных host и port."""
    if not host:
        msg = "host must not be empty"
        raise ValueError(msg)
    if not port:
        msg = "port must not be empty"
        raise ValueError(msg)
    _validate_port(port)
    return Endpoint(host=host, port=port)


def _validate_port(port: str) -> None:
    """Проверяет корректность порта."""
    try:
        port_int = int(port)
    except ValueError:
        msg = f"invalid port {port!r}: must be numeric"
        raise ValueError(msg) from None
    if not 1 <= port_int <= 65535:
        msg = f"port {port_int} out of range (1-65535)"
        raise ValueError(msg)
