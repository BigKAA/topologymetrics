"""Тесты для dependency.py — CheckConfig, Endpoint, Dependency."""

import pytest

from dephealth.dependency import (
    CheckConfig,
    Dependency,
    DependencyType,
    Endpoint,
    default_check_config,
    validate_name,
)


class TestCheckConfig:
    def test_defaults(self) -> None:
        cfg = default_check_config()
        assert cfg.interval == 15.0
        assert cfg.timeout == 5.0
        assert cfg.initial_delay == 5.0
        assert cfg.failure_threshold == 1
        assert cfg.success_threshold == 1

    def test_validate_ok(self) -> None:
        cfg = CheckConfig(interval=30, timeout=10, initial_delay=0)
        cfg.validate()

    @pytest.mark.parametrize(
        ("field", "value"),
        [
            ("interval", 0.5),
            ("interval", 301),
            ("timeout", 0.5),
            ("timeout", 61),
            ("initial_delay", -1),
            ("initial_delay", 301),
            ("failure_threshold", 0),
            ("failure_threshold", 101),
            ("success_threshold", 0),
            ("success_threshold", 101),
        ],
    )
    def test_validate_out_of_range(self, field: str, value: object) -> None:
        cfg = default_check_config()
        setattr(cfg, field, value)
        with pytest.raises(ValueError):
            cfg.validate()


class TestValidateName:
    @pytest.mark.parametrize("name", ["db", "my-service", "Cache_1", "a" * 63])
    def test_valid(self, name: str) -> None:
        validate_name(name)

    @pytest.mark.parametrize("name", ["", "1start", "-bad", "a" * 64, "a b"])
    def test_invalid(self, name: str) -> None:
        with pytest.raises(ValueError):
            validate_name(name)


class TestDependency:
    def test_validate_ok(self) -> None:
        dep = Dependency(
            name="postgres",
            type=DependencyType.POSTGRES,
            endpoints=[Endpoint(host="localhost", port="5432")],
        )
        dep.validate()

    def test_validate_no_endpoints(self) -> None:
        dep = Dependency(name="db", type=DependencyType.POSTGRES)
        with pytest.raises(ValueError, match="at least one endpoint"):
            dep.validate()

    def test_validate_bad_name(self) -> None:
        dep = Dependency(
            name="1bad",
            type=DependencyType.HTTP,
            endpoints=[Endpoint(host="localhost", port="80")],
        )
        with pytest.raises(ValueError, match="invalid dependency name"):
            dep.validate()
