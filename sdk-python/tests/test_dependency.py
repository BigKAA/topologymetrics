"""Тесты для dependency.py — CheckConfig, Endpoint, Dependency, labels."""

import pytest

from dephealth.dependency import (
    CheckConfig,
    Dependency,
    DependencyType,
    Endpoint,
    bool_to_yes_no,
    default_check_config,
    validate_label_name,
    validate_labels,
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


class TestValidateLabelName:
    @pytest.mark.parametrize("label", ["env", "region", "shard_id", "_private", "A1"])
    def test_valid(self, label: str) -> None:
        validate_label_name(label)

    @pytest.mark.parametrize("label", ["", "1bad", "a-b", "a b", "a.b"])
    def test_invalid_format(self, label: str) -> None:
        with pytest.raises(ValueError, match="invalid label name"):
            validate_label_name(label)

    @pytest.mark.parametrize("label", ["name", "dependency", "type", "host", "port", "critical"])
    def test_reserved(self, label: str) -> None:
        with pytest.raises(ValueError, match="reserved"):
            validate_label_name(label)


class TestValidateLabels:
    def test_valid(self) -> None:
        validate_labels({"env": "prod", "region": "us"})

    def test_empty(self) -> None:
        validate_labels({})

    def test_invalid_key(self) -> None:
        with pytest.raises(ValueError):
            validate_labels({"1bad": "val"})

    def test_reserved_key(self) -> None:
        with pytest.raises(ValueError, match="reserved"):
            validate_labels({"name": "val"})


class TestBoolToYesNo:
    def test_true(self) -> None:
        assert bool_to_yes_no(True) == "yes"

    def test_false(self) -> None:
        assert bool_to_yes_no(False) == "no"


class TestDependency:
    def test_validate_ok(self) -> None:
        dep = Dependency(
            name="postgres",
            type=DependencyType.POSTGRES,
            critical=True,
            endpoints=[Endpoint(host="localhost", port="5432")],
        )
        dep.validate()

    def test_validate_no_endpoints(self) -> None:
        dep = Dependency(name="db", type=DependencyType.POSTGRES, critical=True)
        with pytest.raises(ValueError, match="at least one endpoint"):
            dep.validate()

    def test_validate_bad_name(self) -> None:
        dep = Dependency(
            name="1bad",
            type=DependencyType.HTTP,
            critical=True,
            endpoints=[Endpoint(host="localhost", port="80")],
        )
        with pytest.raises(ValueError, match="invalid dependency name"):
            dep.validate()

    def test_validate_endpoint_labels(self) -> None:
        dep = Dependency(
            name="db",
            type=DependencyType.POSTGRES,
            critical=True,
            endpoints=[Endpoint(host="localhost", port="5432", labels={"env": "prod"})],
        )
        dep.validate()

    def test_validate_endpoint_labels_reserved(self) -> None:
        dep = Dependency(
            name="db",
            type=DependencyType.POSTGRES,
            critical=True,
            endpoints=[Endpoint(host="localhost", port="5432", labels={"name": "bad"})],
        )
        with pytest.raises(ValueError, match="reserved"):
            dep.validate()

    def test_validate_endpoint_labels_invalid(self) -> None:
        dep = Dependency(
            name="db",
            type=DependencyType.POSTGRES,
            critical=True,
            endpoints=[Endpoint(host="localhost", port="5432", labels={"1bad": "val"})],
        )
        with pytest.raises(ValueError, match="invalid label name"):
            dep.validate()
