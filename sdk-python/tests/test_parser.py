"""Tests for parser.py — ParseURL, ParseConnectionString, ParseJDBC, ParseParams."""

import pytest

from dephealth.dependency import DependencyType
from dephealth.parser import parse_connection_string, parse_jdbc, parse_params, parse_url


class TestParseURL:
    @pytest.mark.parametrize(
        ("url", "host", "port", "conn_type"),
        [
            ("postgres://localhost:5432/mydb", "localhost", "5432", DependencyType.POSTGRES),
            ("postgresql://db.example.com/test", "db.example.com", "5432", DependencyType.POSTGRES),
            ("mysql://db:3306/app", "db", "3306", DependencyType.MYSQL),
            ("redis://cache:6379/0", "cache", "6379", DependencyType.REDIS),
            ("redis://cache", "cache", "6379", DependencyType.REDIS),
            ("rediss://secure-cache:6380", "secure-cache", "6380", DependencyType.REDIS),
            ("amqp://user:pass@mq:5672/vhost", "mq", "5672", DependencyType.AMQP),
            ("amqps://mq:5671", "mq", "5671", DependencyType.AMQP),
            ("http://api:8080/health", "api", "8080", DependencyType.HTTP),
            ("https://api.example.com", "api.example.com", "443", DependencyType.HTTP),
            ("grpc://service:50051", "service", "50051", DependencyType.GRPC),
            ("kafka://broker:9092", "broker", "9092", DependencyType.KAFKA),
        ],
    )
    def test_single_host(self, url: str, host: str, port: str, conn_type: DependencyType) -> None:
        result = parse_url(url)
        assert len(result) == 1
        assert result[0].host == host
        assert result[0].port == port
        assert result[0].conn_type == conn_type

    def test_kafka_multi_host(self) -> None:
        result = parse_url("kafka://host1:9092,host2:9093,host3:9094")
        assert len(result) == 3
        assert result[0].host == "host1"
        assert result[0].port == "9092"
        assert result[1].host == "host2"
        assert result[1].port == "9093"
        assert result[2].host == "host3"
        assert result[2].port == "9094"

    def test_ipv6(self) -> None:
        result = parse_url("postgres://[::1]:5432/db")
        assert len(result) == 1
        assert result[0].host == "::1"
        assert result[0].port == "5432"

    def test_empty_url(self) -> None:
        with pytest.raises(ValueError, match="empty URL"):
            parse_url("")

    def test_missing_scheme(self) -> None:
        with pytest.raises(ValueError, match="missing scheme"):
            parse_url("localhost:5432")

    def test_unsupported_scheme(self) -> None:
        with pytest.raises(ValueError, match="unsupported URL scheme"):
            parse_url("ftp://localhost")

    def test_default_ports(self) -> None:
        """Verify default ports for all schemes."""
        cases = [
            ("postgres://host/db", "5432"),
            ("mysql://host/db", "3306"),
            ("redis://host", "6379"),
            ("amqp://host", "5672"),
            ("http://host", "80"),
            ("https://host", "443"),
            ("kafka://host", "9092"),
        ]
        for url, expected_port in cases:
            result = parse_url(url)
            assert result[0].port == expected_port, f"URL: {url}"


class TestParseConnectionString:
    def test_postgres_style(self) -> None:
        host, port = parse_connection_string("host=localhost port=5432 dbname=mydb user=admin")
        assert host == "localhost"
        assert port == "5432"

    def test_server_key(self) -> None:
        host, port = parse_connection_string("server=dbhost port=3306 database=app")
        assert host == "dbhost"
        assert port == "3306"

    def test_missing_host(self) -> None:
        with pytest.raises(ValueError, match="host not found"):
            parse_connection_string("port=5432 dbname=mydb")

    def test_missing_port(self) -> None:
        with pytest.raises(ValueError, match="port not found"):
            parse_connection_string("host=localhost dbname=mydb")

    def test_empty(self) -> None:
        with pytest.raises(ValueError, match="empty connection string"):
            parse_connection_string("")


class TestParseJDBC:
    def test_postgresql(self) -> None:
        result = parse_jdbc("jdbc:postgresql://db.example.com:5432/mydb")
        assert len(result) == 1
        assert result[0].host == "db.example.com"
        assert result[0].port == "5432"
        assert result[0].conn_type == DependencyType.POSTGRES

    def test_mysql(self) -> None:
        result = parse_jdbc("jdbc:mysql://db:3306/app")
        assert len(result) == 1
        assert result[0].host == "db"
        assert result[0].port == "3306"
        assert result[0].conn_type == DependencyType.MYSQL

    def test_default_port(self) -> None:
        result = parse_jdbc("jdbc:postgresql://db/mydb")
        assert result[0].port == "5432"

    def test_not_jdbc(self) -> None:
        with pytest.raises(ValueError, match="must start with 'jdbc:'"):
            parse_jdbc("postgres://localhost")

    def test_unsupported_subprotocol(self) -> None:
        with pytest.raises(ValueError, match="unsupported JDBC subprotocol"):
            parse_jdbc("jdbc:oracle://localhost:1521/xe")

    def test_empty(self) -> None:
        with pytest.raises(ValueError, match="empty JDBC URL"):
            parse_jdbc("")


class TestParseParams:
    def test_ok(self) -> None:
        ep = parse_params("localhost", "5432")
        assert ep.host == "localhost"
        assert ep.port == "5432"

    def test_empty_host(self) -> None:
        with pytest.raises(ValueError, match="host must not be empty"):
            parse_params("", "5432")

    def test_empty_port(self) -> None:
        with pytest.raises(ValueError, match="port must not be empty"):
            parse_params("localhost", "")

    def test_invalid_port(self) -> None:
        with pytest.raises(ValueError, match="invalid port"):
            parse_params("localhost", "abc")

    def test_port_out_of_range(self) -> None:
        with pytest.raises(ValueError, match="out of range"):
            parse_params("localhost", "99999")
