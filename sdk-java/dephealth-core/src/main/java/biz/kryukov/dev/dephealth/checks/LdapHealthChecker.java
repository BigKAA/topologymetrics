package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.CheckConnectionException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;
import biz.kryukov.dev.dephealth.ValidationException;

import com.unboundid.ldap.sdk.ExtendedResult;
import com.unboundid.ldap.sdk.LDAPConnection;
import com.unboundid.ldap.sdk.LDAPConnectionOptions;
import com.unboundid.ldap.sdk.LDAPException;
import com.unboundid.ldap.sdk.ResultCode;
import com.unboundid.ldap.sdk.SearchRequest;
import com.unboundid.ldap.sdk.SearchResult;
import com.unboundid.ldap.sdk.SearchScope;
import com.unboundid.ldap.sdk.extensions.StartTLSExtendedRequest;

import java.net.ConnectException;
import java.time.Duration;
import javax.net.ssl.SSLContext;
import javax.net.ssl.SSLSocketFactory;

/**
 * LDAP health checker — supports anonymous bind, simple bind, RootDSE query, search.
 *
 * <p>Supports LDAP (plain), LDAPS (TLS), and StartTLS connections.
 * Two modes: standalone (creates a new connection) or pool (uses existing connection).</p>
 */
public final class LdapHealthChecker implements HealthChecker {

    /** LDAP check method. */
    public enum CheckMethod {
        ANONYMOUS_BIND,
        SIMPLE_BIND,
        ROOT_DSE,
        SEARCH
    }

    /** LDAP search scope. */
    public enum LdapSearchScope {
        BASE,
        ONE,
        SUB
    }

    private final CheckMethod checkMethod;
    private final String bindDN;
    private final String bindPassword;
    private final String baseDN;
    private final String searchFilter;
    private final LdapSearchScope searchScope;
    private final boolean useTLS;
    private final boolean startTLS;
    private final boolean tlsSkipVerify;
    private final LDAPConnection connection;

    private LdapHealthChecker(Builder builder) {
        this.checkMethod = builder.checkMethod;
        this.bindDN = builder.bindDN;
        this.bindPassword = builder.bindPassword;
        this.baseDN = builder.baseDN;
        this.searchFilter = builder.searchFilter;
        this.searchScope = builder.searchScope;
        this.useTLS = builder.useTLS;
        this.startTLS = builder.startTLS;
        this.tlsSkipVerify = builder.tlsSkipVerify;
        this.connection = builder.connection;
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        if (connection != null) {
            checkWithConnection(connection);
        } else {
            checkStandalone(endpoint, timeout);
        }
    }

    private void checkStandalone(Endpoint endpoint, Duration timeout) throws Exception {
        int timeoutMs = (int) timeout.toMillis();

        LDAPConnectionOptions options = new LDAPConnectionOptions();
        options.setConnectTimeoutMillis(timeoutMs);
        options.setResponseTimeoutMillis(timeoutMs);
        options.setFollowReferrals(false);

        LDAPConnection conn = null;
        try {
            if (useTLS) {
                SSLSocketFactory sslFactory = getSslSocketFactory();
                conn = new LDAPConnection(sslFactory, options,
                        endpoint.host(), endpoint.portAsInt());
            } else {
                conn = new LDAPConnection(options,
                        endpoint.host(), endpoint.portAsInt());
            }

            if (startTLS) {
                SSLContext sslContext = tlsSkipVerify
                        ? InsecureSslContext.create()
                        : SSLContext.getDefault();
                ExtendedResult startTlsResult = conn.processExtendedOperation(
                        new StartTLSExtendedRequest(sslContext));
                if (startTlsResult.getResultCode() != ResultCode.SUCCESS) {
                    throw new LDAPException(startTlsResult.getResultCode(),
                            "StartTLS failed: " + startTlsResult.getDiagnosticMessage());
                }
            }

            checkWithConnection(conn);
        } catch (biz.kryukov.dev.dephealth.CheckException e) {
            throw e;
        } catch (LDAPException e) {
            throw classifyLdapError(e);
        } catch (Exception e) {
            throw classifyGenericError(e);
        } finally {
            if (conn != null) {
                conn.close();
            }
        }
    }

    private void checkWithConnection(LDAPConnection conn) throws Exception {
        try {
            switch (checkMethod) {
                case ANONYMOUS_BIND -> conn.bind("", "");
                case SIMPLE_BIND -> conn.bind(bindDN, bindPassword);
                case ROOT_DSE -> searchRootDSE(conn);
                case SEARCH -> searchWithConfig(conn);
                default -> searchRootDSE(conn);
            }
        } catch (LDAPException e) {
            throw classifyLdapError(e);
        }
    }

    private void searchRootDSE(LDAPConnection conn) throws LDAPException {
        SearchRequest req = new SearchRequest(
                "", SearchScope.BASE, "(objectClass=*)",
                "namingContexts", "subschemaSubentry");
        req.setSizeLimit(1);
        SearchResult result = conn.search(req);
        if (result.getResultCode() != ResultCode.SUCCESS) {
            throw new LDAPException(result.getResultCode(),
                    "RootDSE query failed: " + result.getDiagnosticMessage());
        }
    }

    private void searchWithConfig(LDAPConnection conn) throws LDAPException {
        // Bind before search if credentials are provided.
        if (bindDN != null && !bindDN.isEmpty()) {
            conn.bind(bindDN, bindPassword);
        }

        String filter = searchFilter;
        if (filter == null || filter.isEmpty()) {
            filter = "(objectClass=*)";
        }

        SearchScope scope = toUnboundidScope(searchScope);
        SearchRequest req = new SearchRequest(baseDN, scope, filter, "dn");
        req.setSizeLimit(1);
        SearchResult result = conn.search(req);
        if (result.getResultCode() != ResultCode.SUCCESS) {
            throw new LDAPException(result.getResultCode(),
                    "Search failed: " + result.getDiagnosticMessage());
        }
    }

    private static SearchScope toUnboundidScope(LdapSearchScope scope) {
        return switch (scope) {
            case ONE -> SearchScope.ONE;
            case SUB -> SearchScope.SUB;
            default -> SearchScope.BASE;
        };
    }

    private SSLSocketFactory getSslSocketFactory() {
        if (tlsSkipVerify) {
            return InsecureSslContext.create().getSocketFactory();
        }
        try {
            return SSLContext.getDefault().getSocketFactory();
        } catch (java.security.NoSuchAlgorithmException e) {
            throw new RuntimeException("Failed to create default SSL context", e);
        }
    }

    private static Exception classifyLdapError(LDAPException e) {
        ResultCode rc = e.getResultCode();

        // Auth errors: Invalid Credentials (49), Insufficient Access Rights (50).
        if (rc == ResultCode.INVALID_CREDENTIALS
                || rc == ResultCode.INSUFFICIENT_ACCESS_RIGHTS) {
            return new CheckAuthException("LDAP auth error: " + e.getMessage(), e);
        }

        // Server unavailable: Busy (51), Unavailable (52), Unwilling To Perform (53).
        if (rc == ResultCode.BUSY
                || rc == ResultCode.UNAVAILABLE
                || rc == ResultCode.UNWILLING_TO_PERFORM) {
            return new UnhealthyException("LDAP server unhealthy: " + e.getMessage(),
                    "unhealthy", e);
        }

        // Connection error.
        if (rc == ResultCode.CONNECT_ERROR || rc == ResultCode.SERVER_DOWN) {
            return new CheckConnectionException("LDAP connection error: " + e.getMessage(), e);
        }

        return classifyGenericError(e);
    }

    private static Exception classifyGenericError(Exception e) {
        // Connection refused.
        if (hasConnectionRefused(e)) {
            return new CheckConnectionException("LDAP connection refused: " + e.getMessage(), e);
        }

        String msg = e.getMessage();

        // TLS errors.
        if (msg != null && (msg.contains("tls:") || msg.contains("x509:")
                || msg.contains("certificate") || msg.contains("SSL")
                || msg.contains("handshake"))) {
            return new biz.kryukov.dev.dephealth.CheckException(
                    "LDAP TLS error: " + msg, e,
                    biz.kryukov.dev.dephealth.StatusCategory.TLS_ERROR, "tls_error");
        }

        // Message-based fallback for connection errors.
        if (msg != null && (msg.contains("Connection refused")
                || msg.contains("connect timed out")
                || msg.contains("Failed to connect"))) {
            return new CheckConnectionException("LDAP connection error: " + msg, e);
        }

        return e;
    }

    private static boolean hasConnectionRefused(Throwable e) {
        while (e != null) {
            if (e instanceof ConnectException) {
                return true;
            }
            e = e.getCause();
        }
        return false;
    }

    @Override
    public DependencyType type() {
        return DependencyType.LDAP;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private CheckMethod checkMethod = CheckMethod.ROOT_DSE;
        private String bindDN = "";
        private String bindPassword = "";
        private String baseDN = "";
        private String searchFilter = "(objectClass=*)";
        private LdapSearchScope searchScope = LdapSearchScope.BASE;
        private boolean useTLS;
        private boolean startTLS;
        private boolean tlsSkipVerify;
        private LDAPConnection connection;

        private Builder() {}

        public Builder checkMethod(CheckMethod checkMethod) {
            this.checkMethod = checkMethod;
            return this;
        }

        public Builder bindDN(String bindDN) {
            this.bindDN = bindDN;
            return this;
        }

        public Builder bindPassword(String bindPassword) {
            this.bindPassword = bindPassword;
            return this;
        }

        public Builder baseDN(String baseDN) {
            this.baseDN = baseDN;
            return this;
        }

        public Builder searchFilter(String searchFilter) {
            this.searchFilter = searchFilter;
            return this;
        }

        public Builder searchScope(LdapSearchScope searchScope) {
            this.searchScope = searchScope;
            return this;
        }

        public Builder useTLS(boolean useTLS) {
            this.useTLS = useTLS;
            return this;
        }

        public Builder startTLS(boolean startTLS) {
            this.startTLS = startTLS;
            return this;
        }

        public Builder tlsSkipVerify(boolean tlsSkipVerify) {
            this.tlsSkipVerify = tlsSkipVerify;
            return this;
        }

        public Builder connection(LDAPConnection connection) {
            this.connection = connection;
            return this;
        }

        public LdapHealthChecker build() {
            validate();
            return new LdapHealthChecker(this);
        }

        private void validate() {
            if (checkMethod == CheckMethod.SIMPLE_BIND) {
                if (bindDN == null || bindDN.isEmpty()
                        || bindPassword == null || bindPassword.isEmpty()) {
                    throw new ValidationException(
                            "LDAP simple_bind requires bindDN and bindPassword");
                }
            }
            if (checkMethod == CheckMethod.SEARCH) {
                if (baseDN == null || baseDN.isEmpty()) {
                    throw new ValidationException(
                            "LDAP search requires baseDN");
                }
            }
            if (startTLS && useTLS) {
                throw new ValidationException(
                        "startTLS and ldaps:// are incompatible");
            }
        }
    }
}
