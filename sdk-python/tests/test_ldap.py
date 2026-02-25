"""Tests for LDAP health checker."""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import pytest

from dephealth.api import ldap_check
from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckDnsError,
    CheckTimeoutError,
    CheckTlsError,
    UnhealthyError,
)
from dephealth.checks.ldap import (
    LdapChecker,
    LdapCheckMethod,
    LdapSearchScope,
    _classify_socket_error,
)
from dephealth.dependency import Endpoint

# --- LdapChecker unit tests ---


class TestLdapCheckerType:
    def test_type(self) -> None:
        checker = LdapChecker()
        assert checker.checker_type() == "ldap"


class TestLdapCheckerDefaults:
    def test_defaults(self) -> None:
        checker = LdapChecker()
        assert checker._check_method == LdapCheckMethod.ROOT_DSE
        assert checker._search_filter == "(objectClass=*)"
        assert checker._search_scope == LdapSearchScope.BASE
        assert checker._use_tls is False
        assert checker._start_tls is False
        assert checker._tls_skip_verify is False
        assert checker._bind_dn == ""
        assert checker._bind_password == ""
        assert checker._base_dn == ""


class TestLdapCheckerOptions:
    def test_all_options(self) -> None:
        checker = LdapChecker(
            timeout=10.0,
            check_method=LdapCheckMethod.SIMPLE_BIND,
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="password",
            base_dn="dc=test,dc=local",
            search_filter="(uid=*)",
            search_scope=LdapSearchScope.SUB,
            use_tls=True,
            start_tls=False,
            tls_skip_verify=True,
        )
        assert checker._check_method == LdapCheckMethod.SIMPLE_BIND
        assert checker._bind_dn == "cn=admin,dc=test,dc=local"
        assert checker._bind_password == "password"
        assert checker._base_dn == "dc=test,dc=local"
        assert checker._search_filter == "(uid=*)"
        assert checker._search_scope == LdapSearchScope.SUB
        assert checker._use_tls is True
        assert checker._start_tls is False
        assert checker._tls_skip_verify is True
        assert checker._timeout == 10.0


# --- Check methods with mocked ldap3 ---


def _make_mock_ldap3() -> MagicMock:
    """Create a mock ldap3 module with required constants and exception classes."""
    mock = MagicMock()
    mock.NONE = 0
    mock.SIMPLE = "SIMPLE"
    mock.BASE = "BASE"
    mock.LEVEL = "LEVEL"
    mock.SUBTREE = "SUBTREE"

    mock.core.exceptions.LDAPInvalidCredentialsResult = type(
        "LDAPInvalidCredentialsResult", (Exception,), {}
    )
    mock.core.exceptions.LDAPInsufficientAccessRightsResult = type(
        "LDAPInsufficientAccessRightsResult", (Exception,), {}
    )
    mock.core.exceptions.LDAPSocketOpenError = type("LDAPSocketOpenError", (Exception,), {})
    mock.core.exceptions.LDAPSocketSendError = type("LDAPSocketSendError", (Exception,), {})
    mock.core.exceptions.LDAPBusyResult = type("LDAPBusyResult", (Exception,), {})
    mock.core.exceptions.LDAPUnavailableResult = type("LDAPUnavailableResult", (Exception,), {})
    mock.core.exceptions.LDAPUnwillingToPerformResult = type(
        "LDAPUnwillingToPerformResult", (Exception,), {}
    )
    mock.core.exceptions.LDAPExceptionError = type("LDAPExceptionError", (Exception,), {})
    return mock


def _make_mock_conn() -> MagicMock:
    """Create a mock ldap3 Connection."""
    conn = MagicMock()
    conn.open = MagicMock()
    conn.bind = MagicMock()
    conn.search = MagicMock(return_value=True)
    conn.unbind = MagicMock()
    conn.start_tls = MagicMock()
    return conn


class TestRootDSECheck:
    async def test_root_dse_success(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(check_method=LdapCheckMethod.ROOT_DSE)
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_conn.open.assert_called_once()
        mock_conn.search.assert_called_once()
        mock_conn.unbind.assert_called_once()


class TestAnonymousBindCheck:
    async def test_anonymous_bind_success(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(check_method=LdapCheckMethod.ANONYMOUS_BIND)
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_conn.bind.assert_called_once()
        mock_conn.unbind.assert_called_once()


class TestSimpleBindCheck:
    async def test_simple_bind_success(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.SIMPLE_BIND,
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="password",
        )
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        assert mock_conn.user == "cn=admin,dc=test,dc=local"
        assert mock_conn.password == "password"
        assert mock_conn.authentication == "SIMPLE"
        mock_conn.bind.assert_called_once()

    async def test_simple_bind_invalid_credentials(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.bind.side_effect = mock_ldap3.core.exceptions.LDAPInvalidCredentialsResult(
            "invalid credentials"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.SIMPLE_BIND,
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="wrong",
        )
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckAuthError, match="auth error"),
        ):
            await checker.check(ep)

    async def test_simple_bind_insufficient_access(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.bind.side_effect = mock_ldap3.core.exceptions.LDAPInsufficientAccessRightsResult(
            "access denied"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.SIMPLE_BIND,
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="password",
        )
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckAuthError, match="access denied"),
        ):
            await checker.check(ep)


class TestSearchCheck:
    async def test_search_success(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.SEARCH,
            base_dn="dc=test,dc=local",
            search_filter="(uid=testuser)",
            search_scope=LdapSearchScope.SUB,
        )
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_conn.search.assert_called_once_with(
            search_base="dc=test,dc=local",
            search_filter="(uid=testuser)",
            search_scope="SUBTREE",
            attributes=["dn"],
            size_limit=1,
        )


class TestStartTLSCheck:
    async def test_start_tls(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.ROOT_DSE,
            start_tls=True,
            tls_skip_verify=True,
        )
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_conn.start_tls.assert_called_once()
        mock_conn.search.assert_called_once()


class TestLDAPSCheck:
    async def test_ldaps_creates_server_with_ssl(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(
            check_method=LdapCheckMethod.ROOT_DSE,
            use_tls=True,
        )
        ep = Endpoint(host="localhost", port="636")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_ldap3.Server.assert_called_once()
        call_kwargs = mock_ldap3.Server.call_args
        assert call_kwargs[1]["use_ssl"] is True


# --- Error classification tests ---


class TestErrorClassification:
    async def test_connection_refused(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.open.side_effect = mock_ldap3.core.exceptions.LDAPSocketOpenError(
            "connection refused"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="127.0.0.1", port="1")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckConnectionRefusedError, match="refused"),
        ):
            await checker.check(ep)

    async def test_dns_error(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.open.side_effect = mock_ldap3.core.exceptions.LDAPSocketOpenError(
            "Name or service not known"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="nonexistent.invalid", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckDnsError, match="DNS"),
        ):
            await checker.check(ep)

    async def test_tls_error(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.open.side_effect = mock_ldap3.core.exceptions.LDAPSocketOpenError(
            "TLS handshake failed: certificate verify failed"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker(use_tls=True)
        ep = Endpoint(host="localhost", port="636")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckTlsError, match="TLS"),
        ):
            await checker.check(ep)

    async def test_timeout(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.open.side_effect = mock_ldap3.core.exceptions.LDAPSocketOpenError(
            "connection timed out"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckTimeoutError, match="timed out"),
        ):
            await checker.check(ep)

    async def test_unhealthy_busy(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.search.side_effect = mock_ldap3.core.exceptions.LDAPBusyResult("server busy")
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(UnhealthyError, match="unhealthy"),
        ):
            await checker.check(ep)

    async def test_unhealthy_unavailable(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.search.side_effect = mock_ldap3.core.exceptions.LDAPUnavailableResult(
            "unavailable"
        )
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(UnhealthyError, match="unhealthy"),
        ):
            await checker.check(ep)

    async def test_socket_send_error(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_conn = _make_mock_conn()
        mock_conn.search.side_effect = mock_ldap3.core.exceptions.LDAPSocketSendError("broken pipe")
        mock_ldap3.Connection.return_value = mock_conn

        checker = LdapChecker()
        ep = Endpoint(host="localhost", port="389")

        with (
            patch.dict(
                "sys.modules",
                {
                    "ldap3": mock_ldap3,
                    "ldap3.core": mock_ldap3.core,
                    "ldap3.core.exceptions": mock_ldap3.core.exceptions,
                },
            ),
            pytest.raises(CheckConnectionRefusedError, match="failed"),
        ):
            await checker.check(ep)


class TestClassifySocketError:
    def test_tls_error(self) -> None:
        err = Exception("SSL: certificate verify failed")
        with pytest.raises(CheckTlsError):
            _classify_socket_error(err, "host:636")

    def test_dns_error(self) -> None:
        err = Exception("getaddrinfo failed: Name or service not known")
        with pytest.raises(CheckDnsError):
            _classify_socket_error(err, "bad-host:389")

    def test_refused_error(self) -> None:
        err = Exception("connection refused")
        with pytest.raises(CheckConnectionRefusedError):
            _classify_socket_error(err, "host:389")

    def test_timeout_error(self) -> None:
        err = Exception("connection timed out")
        with pytest.raises(CheckTimeoutError):
            _classify_socket_error(err, "host:389")

    def test_generic_socket_error(self) -> None:
        err = Exception("some unknown error")
        with pytest.raises(CheckConnectionRefusedError):
            _classify_socket_error(err, "host:389")


# --- Pool mode tests ---


class TestPoolMode:
    async def test_pool_mode_root_dse(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_client = _make_mock_conn()

        checker = LdapChecker(
            check_method=LdapCheckMethod.ROOT_DSE,
            client=mock_client,
        )
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_client.search.assert_called_once()

    async def test_pool_mode_simple_bind(self) -> None:
        mock_ldap3 = _make_mock_ldap3()
        mock_client = _make_mock_conn()

        checker = LdapChecker(
            check_method=LdapCheckMethod.SIMPLE_BIND,
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="password",
            client=mock_client,
        )
        ep = Endpoint(host="localhost", port="389")

        with patch.dict(
            "sys.modules",
            {
                "ldap3": mock_ldap3,
                "ldap3.core": mock_ldap3.core,
                "ldap3.core.exceptions": mock_ldap3.core.exceptions,
            },
        ):
            await checker.check(ep)

        mock_client.bind.assert_called_once()


# --- Factory function validation tests ---


class TestLdapCheckFactory:
    def test_default_creation(self) -> None:
        spec = ldap_check("ldap-svc", url="ldap://localhost:389", critical=True)
        assert spec.name == "ldap-svc"
        assert spec.dep_type.value == "ldap"
        assert spec.checker._check_method == LdapCheckMethod.ROOT_DSE
        assert spec.checker._use_tls is False

    def test_ldaps_url(self) -> None:
        spec = ldap_check("ldap-svc", url="ldaps://localhost:636", critical=True)
        assert spec.checker._use_tls is True

    def test_simple_bind_without_credentials_raises(self) -> None:
        with pytest.raises(ValueError, match="simple_bind requires bind_dn and bind_password"):
            ldap_check(
                "ldap-svc",
                url="ldap://localhost:389",
                check_method="simple_bind",
                critical=True,
            )

    def test_simple_bind_without_password_raises(self) -> None:
        with pytest.raises(ValueError, match="simple_bind requires bind_dn and bind_password"):
            ldap_check(
                "ldap-svc",
                url="ldap://localhost:389",
                check_method="simple_bind",
                bind_dn="cn=admin,dc=test,dc=local",
                critical=True,
            )

    def test_search_without_base_dn_raises(self) -> None:
        with pytest.raises(ValueError, match="search method requires base_dn"):
            ldap_check(
                "ldap-svc",
                url="ldap://localhost:389",
                check_method="search",
                critical=True,
            )

    def test_start_tls_with_ldaps_raises(self) -> None:
        with pytest.raises(ValueError, match="startTLS is incompatible with ldaps://"):
            ldap_check(
                "ldap-svc",
                url="ldaps://localhost:636",
                start_tls=True,
                critical=True,
            )

    def test_simple_bind_valid(self) -> None:
        spec = ldap_check(
            "ldap-svc",
            url="ldap://localhost:389",
            check_method="simple_bind",
            bind_dn="cn=admin,dc=test,dc=local",
            bind_password="password",
            critical=True,
        )
        assert spec.checker._check_method == LdapCheckMethod.SIMPLE_BIND
        assert spec.checker._bind_dn == "cn=admin,dc=test,dc=local"

    def test_search_valid(self) -> None:
        spec = ldap_check(
            "ldap-svc",
            url="ldap://localhost:389",
            check_method="search",
            base_dn="dc=test,dc=local",
            search_filter="(uid=testuser)",
            search_scope="sub",
            critical=True,
        )
        assert spec.checker._check_method == LdapCheckMethod.SEARCH
        assert spec.checker._base_dn == "dc=test,dc=local"
        assert spec.checker._search_filter == "(uid=testuser)"
        assert spec.checker._search_scope == LdapSearchScope.SUB

    def test_start_tls(self) -> None:
        spec = ldap_check(
            "ldap-svc",
            url="ldap://localhost:389",
            start_tls=True,
            tls_skip_verify=True,
            critical=True,
        )
        assert spec.checker._start_tls is True
        assert spec.checker._tls_skip_verify is True

    def test_host_port_fallback(self) -> None:
        spec = ldap_check(
            "ldap-svc",
            host="ldap.example.com",
            port="389",
            critical=True,
        )
        assert len(spec.endpoints) == 1
        assert spec.endpoints[0].host == "ldap.example.com"
        assert spec.endpoints[0].port == "389"

    def test_labels_passed(self) -> None:
        spec = ldap_check(
            "ldap-svc",
            url="ldap://localhost:389",
            critical=True,
            labels={"env": "prod"},
        )
        assert spec.labels == {"env": "prod"}

    def test_pool_mode(self) -> None:
        mock_client = MagicMock()
        spec = ldap_check(
            "ldap-svc",
            url="ldap://localhost:389",
            client=mock_client,
            critical=True,
        )
        assert spec.checker._client is mock_client


# --- Parser tests ---


class TestParserLdapScheme:
    def test_ldap_scheme_mapping(self) -> None:
        from dephealth.parser import parse_url

        result = parse_url("ldap://ldap.example.com:389")
        assert len(result) == 1
        assert result[0].host == "ldap.example.com"
        assert result[0].port == "389"
        assert result[0].conn_type.value == "ldap"

    def test_ldaps_scheme_mapping(self) -> None:
        from dephealth.parser import parse_url

        result = parse_url("ldaps://ldap.example.com:636")
        assert len(result) == 1
        assert result[0].host == "ldap.example.com"
        assert result[0].port == "636"
        assert result[0].conn_type.value == "ldap"

    def test_ldap_default_port(self) -> None:
        from dephealth.parser import parse_url

        result = parse_url("ldap://ldap.example.com")
        assert result[0].port == "389"

    def test_ldaps_default_port(self) -> None:
        from dephealth.parser import parse_url

        result = parse_url("ldaps://ldap.example.com")
        assert result[0].port == "636"


# --- DependencyType tests ---


class TestDependencyTypeLdap:
    def test_ldap_type_exists(self) -> None:
        from dephealth.dependency import DependencyType

        assert DependencyType.LDAP == "ldap"
        assert DependencyType.LDAP.value == "ldap"
