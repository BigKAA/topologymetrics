"""Health check for LDAP servers.

Supports four check methods: anonymous_bind, simple_bind, root_dse, search.
Supports LDAP (plain), LDAPS (TLS), and StartTLS connections.
"""

from __future__ import annotations

import asyncio
import contextlib
import functools
import ssl
from enum import StrEnum
from typing import Any

from dephealth.checker import (
    CheckAuthError,
    CheckConnectionRefusedError,
    CheckDnsError,
    CheckTimeoutError,
    CheckTlsError,
    UnhealthyError,
)
from dephealth.dependency import Endpoint


class LdapCheckMethod(StrEnum):
    """LDAP check method."""

    ANONYMOUS_BIND = "anonymous_bind"
    SIMPLE_BIND = "simple_bind"
    ROOT_DSE = "root_dse"
    SEARCH = "search"


class LdapSearchScope(StrEnum):
    """LDAP search scope."""

    BASE = "base"
    ONE = "one"
    SUB = "sub"


_SCOPE_MAP = {
    LdapSearchScope.BASE: "BASE",
    LdapSearchScope.ONE: "LEVEL",
    LdapSearchScope.SUB: "SUBTREE",
}


class LdapChecker:
    """Health check for an LDAP server.

    Supports two modes:
    - Standalone: creates a new LDAP connection per check
    - Pool: uses an existing ldap3 Connection
    """

    def __init__(
        self,
        timeout: float = 5.0,
        check_method: LdapCheckMethod = LdapCheckMethod.ROOT_DSE,
        bind_dn: str = "",
        bind_password: str = "",
        base_dn: str = "",
        search_filter: str = "(objectClass=*)",
        search_scope: LdapSearchScope = LdapSearchScope.BASE,
        use_tls: bool = False,
        start_tls: bool = False,
        tls_skip_verify: bool = False,
        client: Any = None,  # noqa: ANN401
    ) -> None:
        self._timeout = timeout
        self._check_method = check_method
        self._bind_dn = bind_dn
        self._bind_password = bind_password
        self._base_dn = base_dn
        self._search_filter = search_filter
        self._search_scope = search_scope
        self._use_tls = use_tls
        self._start_tls = start_tls
        self._tls_skip_verify = tls_skip_verify
        self._client = client

    async def check(self, endpoint: Endpoint) -> None:
        """Perform an LDAP health check against the given endpoint."""
        try:
            import ldap3
        except ImportError:
            msg = "ldap3 is required for LDAP checker"
            raise ImportError(msg) from None

        if self._client is not None:
            await self._check_with_client(self._client, endpoint)
            return

        await self._check_standalone(endpoint, ldap3)

    async def _check_standalone(self, endpoint: Endpoint, ldap3: Any) -> None:  # noqa: ANN401
        """Create a new connection and perform the check."""
        loop = asyncio.get_running_loop()
        try:
            await asyncio.wait_for(
                loop.run_in_executor(
                    None,
                    functools.partial(self._check_sync, endpoint, ldap3),
                ),
                timeout=self._timeout,
            )
        except TimeoutError as exc:
            msg = f"LDAP connection to {endpoint.host}:{endpoint.port} timed out"
            raise CheckTimeoutError(msg) from exc

    def _check_sync(self, endpoint: Endpoint, ldap3: Any) -> None:  # noqa: ANN401
        """Synchronous LDAP check (runs in executor)."""
        target = f"{endpoint.host}:{endpoint.port}"
        tls_obj = None

        if self._use_tls or self._start_tls:
            tls_obj = ldap3.Tls(
                validate=ssl.CERT_NONE if self._tls_skip_verify else ssl.CERT_REQUIRED,
            )

        server = ldap3.Server(
            endpoint.host,
            port=int(endpoint.port),
            use_ssl=self._use_tls,
            tls=tls_obj,
            connect_timeout=self._timeout,
            get_info=ldap3.NONE,
        )

        conn = None
        try:
            conn = ldap3.Connection(
                server,
                auto_bind=False,
                raise_exceptions=True,
                receive_timeout=self._timeout,
            )
            conn.open()

            if self._start_tls:
                conn.start_tls()

            self._execute_check(conn, ldap3, target)
        except ldap3.core.exceptions.LDAPInvalidCredentialsResult as e:
            raise CheckAuthError(f"LDAP auth error at {target}: {e}") from e
        except ldap3.core.exceptions.LDAPInsufficientAccessRightsResult as e:
            raise CheckAuthError(f"LDAP access denied at {target}: {e}") from e
        except ldap3.core.exceptions.LDAPSocketOpenError as e:
            _classify_socket_error(e, target)
        except ldap3.core.exceptions.LDAPSocketSendError as e:
            msg = f"LDAP connection to {target} failed: {e}"
            raise CheckConnectionRefusedError(msg) from e
        except (
            ldap3.core.exceptions.LDAPBusyResult,
            ldap3.core.exceptions.LDAPUnavailableResult,
            ldap3.core.exceptions.LDAPUnwillingToPerformResult,
        ) as e:
            msg = f"LDAP server {target} unhealthy: {e}"
            raise UnhealthyError(msg) from e
        except (
            CheckAuthError,
            CheckConnectionRefusedError,
            CheckDnsError,
            CheckTlsError,
            CheckTimeoutError,
            UnhealthyError,
        ):
            raise
        except ldap3.core.exceptions.LDAPExceptionError as e:
            msg = f"LDAP error at {target}: {e}"
            raise CheckConnectionRefusedError(msg) from e
        finally:
            if conn is not None:
                with contextlib.suppress(Exception):
                    conn.unbind()

    def _execute_check(self, conn: Any, ldap3: Any, target: str) -> None:  # noqa: ANN401
        """Execute the configured check method on the connection."""
        if self._check_method == LdapCheckMethod.ANONYMOUS_BIND:
            conn.bind()
        elif self._check_method == LdapCheckMethod.SIMPLE_BIND:
            conn.user = self._bind_dn
            conn.password = self._bind_password
            conn.authentication = ldap3.SIMPLE
            conn.bind()
        elif self._check_method == LdapCheckMethod.ROOT_DSE:
            conn.search(
                search_base="",
                search_filter="(objectClass=*)",
                search_scope=ldap3.BASE,
                attributes=["namingContexts", "subschemaSubentry"],
                size_limit=1,
            )
        elif self._check_method == LdapCheckMethod.SEARCH:
            scope = getattr(ldap3, _SCOPE_MAP[self._search_scope])
            conn.search(
                search_base=self._base_dn,
                search_filter=self._search_filter,
                search_scope=scope,
                attributes=["dn"],
                size_limit=1,
            )

    async def _check_with_client(self, client: Any, endpoint: Endpoint) -> None:  # noqa: ANN401
        """Use an existing ldap3 Connection."""
        try:
            import ldap3
        except ImportError:
            msg = "ldap3 is required for LDAP checker"
            raise ImportError(msg) from None

        target = f"{endpoint.host}:{endpoint.port}"
        loop = asyncio.get_running_loop()
        try:
            await loop.run_in_executor(
                None,
                functools.partial(self._execute_check, client, ldap3, target),
            )
        except Exception:
            raise

    def checker_type(self) -> str:
        """Return the checker type."""
        return "ldap"


def _classify_socket_error(err: Exception, target: str) -> None:
    """Re-raise LDAPSocketOpenError with appropriate classification."""
    msg = str(err)
    lower = msg.lower()

    if "tls" in lower or "ssl" in lower or "certificate" in lower:
        raise CheckTlsError(f"LDAP TLS error at {target}: {err}") from err
    if "name or service not known" in lower or "getaddrinfo" in lower or "nodename" in lower:
        raise CheckDnsError(f"LDAP DNS error for {target}: {err}") from err
    if "refused" in lower:
        raise CheckConnectionRefusedError(f"LDAP connection to {target} refused: {err}") from err
    if "timed out" in lower or "timeout" in lower:
        raise CheckTimeoutError(f"LDAP connection to {target} timed out: {err}") from err

    raise CheckConnectionRefusedError(f"LDAP connection to {target} failed: {err}") from err
