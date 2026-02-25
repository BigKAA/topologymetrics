package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.ValidationException;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.CheckMethod;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.LdapSearchScope;

import com.unboundid.ldap.sdk.LDAPConnection;
import com.unboundid.ldap.sdk.LDAPException;
import com.unboundid.ldap.sdk.LDAPSearchException;
import com.unboundid.ldap.sdk.BindResult;
import com.unboundid.ldap.sdk.LDAPResult;
import com.unboundid.ldap.sdk.ResultCode;
import com.unboundid.ldap.sdk.SearchResult;
import com.unboundid.ldap.sdk.SearchResultEntry;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.time.Duration;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertDoesNotThrow;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class LdapHealthCheckerTest {

    @Mock
    private LDAPConnection ldapConnection;

    private static final Endpoint EP = new Endpoint("localhost", "389");
    private static final Duration TIMEOUT = Duration.ofSeconds(5);

    @Test
    void type() {
        assertEquals(DependencyType.LDAP, LdapHealthChecker.builder().build().type());
    }

    // --- RootDSE (default) ---

    @Test
    void rootDseCheckSucceeds() throws Exception {
        SearchResult searchResult = new SearchResult(1, ResultCode.SUCCESS,
                null, null, null, List.of(), List.of(), 0, 0, null);
        when(ldapConnection.search(any(com.unboundid.ldap.sdk.SearchRequest.class)))
                .thenReturn(searchResult);

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.ROOT_DSE)
                .connection(ldapConnection)
                .build();

        assertDoesNotThrow(() -> checker.check(EP, TIMEOUT));
        verify(ldapConnection).search(any(com.unboundid.ldap.sdk.SearchRequest.class));
    }

    // --- Anonymous Bind ---

    @Test
    void anonymousBindSucceeds() throws Exception {
        BindResult bindResult = new BindResult(1, ResultCode.SUCCESS, null, null, null, null);
        when(ldapConnection.bind(eq(""), eq(""))).thenReturn(bindResult);

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.ANONYMOUS_BIND)
                .connection(ldapConnection)
                .build();

        assertDoesNotThrow(() -> checker.check(EP, TIMEOUT));
        verify(ldapConnection).bind("", "");
    }

    // --- Simple Bind ---

    @Test
    void simpleBindSucceeds() throws Exception {
        BindResult bindResult = new BindResult(1, ResultCode.SUCCESS, null, null, null, null);
        when(ldapConnection.bind(eq("cn=admin,dc=test,dc=local"), eq("password")))
                .thenReturn(bindResult);

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.SIMPLE_BIND)
                .bindDN("cn=admin,dc=test,dc=local")
                .bindPassword("password")
                .connection(ldapConnection)
                .build();

        assertDoesNotThrow(() -> checker.check(EP, TIMEOUT));
        verify(ldapConnection).bind("cn=admin,dc=test,dc=local", "password");
    }

    @Test
    void simpleBindInvalidCredentialsThrowsAuthException() throws Exception {
        when(ldapConnection.bind(any(String.class), any(String.class)))
                .thenThrow(new LDAPException(ResultCode.INVALID_CREDENTIALS,
                        "Invalid credentials"));

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.SIMPLE_BIND)
                .bindDN("cn=admin,dc=test,dc=local")
                .bindPassword("wrong")
                .connection(ldapConnection)
                .build();

        Exception ex = assertThrows(Exception.class, () -> checker.check(EP, TIMEOUT));
        assertInstanceOf(CheckAuthException.class, ex);
    }

    @Test
    void simpleBindInsufficientAccessThrowsAuthException() throws Exception {
        when(ldapConnection.bind(any(String.class), any(String.class)))
                .thenThrow(new LDAPException(ResultCode.INSUFFICIENT_ACCESS_RIGHTS,
                        "Insufficient access"));

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.SIMPLE_BIND)
                .bindDN("cn=user,dc=test,dc=local")
                .bindPassword("pass")
                .connection(ldapConnection)
                .build();

        Exception ex = assertThrows(Exception.class, () -> checker.check(EP, TIMEOUT));
        assertInstanceOf(CheckAuthException.class, ex);
    }

    // --- Search ---

    @Test
    void searchSucceeds() throws Exception {
        SearchResult searchResult = new SearchResult(1, ResultCode.SUCCESS,
                null, null, null, List.of(), List.of(), 1, 0, null);
        when(ldapConnection.search(any(com.unboundid.ldap.sdk.SearchRequest.class)))
                .thenReturn(searchResult);

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.SEARCH)
                .baseDN("dc=test,dc=local")
                .searchFilter("(uid=testuser)")
                .searchScope(LdapSearchScope.SUB)
                .connection(ldapConnection)
                .build();

        assertDoesNotThrow(() -> checker.check(EP, TIMEOUT));
        verify(ldapConnection).search(any(com.unboundid.ldap.sdk.SearchRequest.class));
    }

    // --- Validation ---

    @Test
    void simpleBindWithoutBindDNThrowsValidationException() {
        assertThrows(ValidationException.class, () ->
                LdapHealthChecker.builder()
                        .checkMethod(CheckMethod.SIMPLE_BIND)
                        .build());
    }

    @Test
    void simpleBindWithoutPasswordThrowsValidationException() {
        assertThrows(ValidationException.class, () ->
                LdapHealthChecker.builder()
                        .checkMethod(CheckMethod.SIMPLE_BIND)
                        .bindDN("cn=admin,dc=test,dc=local")
                        .build());
    }

    @Test
    void searchWithoutBaseDNThrowsValidationException() {
        assertThrows(ValidationException.class, () ->
                LdapHealthChecker.builder()
                        .checkMethod(CheckMethod.SEARCH)
                        .build());
    }

    @Test
    void startTlsWithLdapsThrowsValidationException() {
        assertThrows(ValidationException.class, () ->
                LdapHealthChecker.builder()
                        .useTLS(true)
                        .startTLS(true)
                        .build());
    }

    // --- Error classification ---

    @Test
    void busyServerThrowsUnhealthyException() throws Exception {
        when(ldapConnection.search(any(com.unboundid.ldap.sdk.SearchRequest.class)))
                .thenThrow(new LDAPSearchException(ResultCode.BUSY, "Server is busy"));

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.ROOT_DSE)
                .connection(ldapConnection)
                .build();

        Exception ex = assertThrows(Exception.class, () -> checker.check(EP, TIMEOUT));
        assertInstanceOf(biz.kryukov.dev.dephealth.UnhealthyException.class, ex);
    }

    @Test
    void connectionRefusedStandalone() {
        LdapHealthChecker checker = LdapHealthChecker.builder()
                .checkMethod(CheckMethod.ROOT_DSE)
                .build();

        Endpoint badEp = new Endpoint("localhost", "19999");
        Exception ex = assertThrows(Exception.class,
                () -> checker.check(badEp, Duration.ofSeconds(2)));
        assertInstanceOf(biz.kryukov.dev.dephealth.CheckConnectionException.class, ex);
    }

    // --- Builder defaults ---

    @Test
    void defaultCheckMethodIsRootDSE() throws Exception {
        SearchResult searchResult = new SearchResult(1, ResultCode.SUCCESS,
                null, null, null, List.of(), List.of(), 0, 0, null);
        when(ldapConnection.search(any(com.unboundid.ldap.sdk.SearchRequest.class)))
                .thenReturn(searchResult);

        LdapHealthChecker checker = LdapHealthChecker.builder()
                .connection(ldapConnection)
                .build();

        assertDoesNotThrow(() -> checker.check(EP, TIMEOUT));
    }
}
