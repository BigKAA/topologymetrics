package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;
import biz.kryukov.dev.dephealth.ValidationException;

import javax.net.ssl.SNIHostName;
import javax.net.ssl.SSLParameters;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.Base64;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * HTTP health checker — performs a GET request to healthPath and expects 2xx.
 */
public final class HttpHealthChecker implements HealthChecker {

    private static final String DEFAULT_HEALTH_PATH = "/health";
    private static final String USER_AGENT = "dephealth/0.8.1";

    // Allow setting the Host header via HttpClient.
    // Java HttpClient restricts the Host header by default; this system property
    // must be set before the first request to enable Host header override.
    static {
        String existing = System.getProperty("jdk.httpclient.allowRestrictedHeaders");
        if (existing == null || existing.isEmpty()) {
            System.setProperty("jdk.httpclient.allowRestrictedHeaders", "host");
        } else if (!existing.toLowerCase().contains("host")) {
            System.setProperty("jdk.httpclient.allowRestrictedHeaders", existing + ",host");
        }
    }

    private final String healthPath;
    private final boolean tlsEnabled;
    private final String hostHeader;
    private final Map<String, String> headers;
    private final HttpClient client;

    private HttpHealthChecker(Builder builder) {
        this.healthPath = builder.healthPath;
        this.tlsEnabled = builder.tlsEnabled;
        this.hostHeader = builder.hostHeader;
        this.headers = Collections.unmodifiableMap(new LinkedHashMap<>(builder.resolvedHeaders));

        HttpClient.Builder clientBuilder = HttpClient.newBuilder()
                .followRedirects(HttpClient.Redirect.NORMAL);
        if (builder.tlsEnabled && builder.tlsSkipVerify) {
            clientBuilder.sslContext(InsecureSslContext.create());
        }
        // Set TLS SNI when hostHeader is configured and TLS is enabled.
        if (builder.tlsEnabled && builder.hostHeader != null && !builder.hostHeader.isEmpty()) {
            SSLParameters sslParams = new SSLParameters();
            sslParams.setServerNames(List.of(new SNIHostName(builder.hostHeader)));
            clientBuilder.sslParameters(sslParams);
        }
        this.client = clientBuilder.build();
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        String scheme = tlsEnabled ? "https" : "http";
        String host = endpoint.host();
        // IPv6
        if (host.contains(":")) {
            host = "[" + host + "]";
        }
        URI uri = URI.create(scheme + "://" + host + ":" + endpoint.port() + healthPath);

        HttpRequest.Builder requestBuilder = HttpRequest.newBuilder()
                .uri(uri)
                .timeout(timeout)
                .header("User-Agent", USER_AGENT)
                .GET();

        // Apply custom headers after User-Agent so they can override it.
        for (Map.Entry<String, String> entry : headers.entrySet()) {
            requestBuilder.header(entry.getKey(), entry.getValue());
        }

        // Override Host header if configured (used for ingress/gateway routing by IP).
        if (hostHeader != null && !hostHeader.isEmpty()) {
            requestBuilder.header("Host", hostHeader);
        }

        HttpRequest request = requestBuilder.build();

        HttpResponse<Void> response = client.send(request, HttpResponse.BodyHandlers.discarding());

        int status = response.statusCode();
        if (status < 200 || status >= 300) {
            // HTTP 401/403 → auth_error.
            if (status == 401 || status == 403) {
                throw new CheckAuthException(
                        "HTTP health check failed: status " + status);
            }
            throw new UnhealthyException(
                    "HTTP health check failed: status " + status, "http_" + status);
        }
    }

    @Override
    public void close() {
        client.close();
    }

    @Override
    public DependencyType type() {
        return DependencyType.HTTP;
    }

    /** Returns the configured health check path. */
    public String healthPath() {
        return healthPath;
    }

    /** Returns whether TLS is enabled. */
    public boolean tlsEnabled() {
        return tlsEnabled;
    }

    /** Creates a new builder with default settings. */
    public static Builder builder() {
        return new Builder();
    }

    /** Builder for {@link HttpHealthChecker}. */
    public static final class Builder {
        private String healthPath = DEFAULT_HEALTH_PATH;
        private boolean tlsEnabled;
        private boolean tlsSkipVerify;
        private String hostHeader;
        private Map<String, String> customHeaders = new LinkedHashMap<>();
        private String bearerToken;
        private String basicAuthUsername;
        private String basicAuthPassword;
        private final Map<String, String> resolvedHeaders = new LinkedHashMap<>();

        private Builder() {}

        /** Sets the health check path (default: {@code /health}). */
        public Builder healthPath(String healthPath) {
            this.healthPath = healthPath;
            return this;
        }

        /** Enables or disables TLS. */
        public Builder tlsEnabled(boolean tlsEnabled) {
            this.tlsEnabled = tlsEnabled;
            return this;
        }

        /** Skips TLS certificate verification. */
        public Builder tlsSkipVerify(boolean tlsSkipVerify) {
            this.tlsSkipVerify = tlsSkipVerify;
            return this;
        }

        /**
         * Sets the HTTP Host header override for health check requests.
         * Used when connecting by IP through ingress/gateway for Host-based routing.
         * When TLS is enabled, also sets TLS SNI (ServerName) to the same value.
         * Does NOT affect the "host" metric label.
         */
        public Builder hostHeader(String hostHeader) {
            this.hostHeader = hostHeader;
            return this;
        }

        /** Sets custom HTTP headers. */
        public Builder headers(Map<String, String> headers) {
            this.customHeaders = new LinkedHashMap<>(headers);
            return this;
        }

        /** Sets a Bearer token for authentication. */
        public Builder bearerToken(String token) {
            this.bearerToken = token;
            return this;
        }

        /** Sets Basic authentication credentials. */
        public Builder basicAuth(String username, String password) {
            this.basicAuthUsername = username;
            this.basicAuthPassword = password;
            return this;
        }

        /** Builds the checker, validating configuration. */
        public HttpHealthChecker build() {
            validateAuthConflicts();
            validateHostHeaderConflicts();
            buildResolvedHeaders();
            return new HttpHealthChecker(this);
        }

        private void validateHostHeaderConflicts() {
            if (hostHeader == null || hostHeader.isEmpty()) {
                return;
            }
            for (String key : customHeaders.keySet()) {
                if (key.equalsIgnoreCase("Host")) {
                    throw new ValidationException(
                            "conflicting Host header: specify only one of "
                                    + "hostHeader or Host in headers");
                }
            }
        }

        private void validateAuthConflicts() {
            int methods = 0;
            if (bearerToken != null && !bearerToken.isEmpty()) {
                methods++;
            }
            if (basicAuthUsername != null && !basicAuthUsername.isEmpty()) {
                methods++;
            }
            for (String key : customHeaders.keySet()) {
                if (key.equalsIgnoreCase("Authorization")) {
                    methods++;
                    break;
                }
            }
            if (methods > 1) {
                throw new ValidationException(
                        "conflicting auth methods: specify only one of "
                                + "bearerToken, basicAuth, or Authorization header");
            }
        }

        private void buildResolvedHeaders() {
            resolvedHeaders.putAll(customHeaders);
            if (bearerToken != null && !bearerToken.isEmpty()) {
                resolvedHeaders.put("Authorization", "Bearer " + bearerToken);
            }
            if (basicAuthUsername != null && !basicAuthUsername.isEmpty()) {
                String credentials = basicAuthUsername + ":"
                        + (basicAuthPassword != null ? basicAuthPassword : "");
                String encoded = Base64.getEncoder()
                        .encodeToString(credentials.getBytes(StandardCharsets.UTF_8));
                resolvedHeaders.put("Authorization", "Basic " + encoded);
            }
        }
    }
}
