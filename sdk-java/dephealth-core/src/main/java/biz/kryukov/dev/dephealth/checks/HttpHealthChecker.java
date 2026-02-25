package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;
import biz.kryukov.dev.dephealth.ValidationException;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.Base64;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * HTTP health checker — performs a GET request to healthPath and expects 2xx.
 */
public final class HttpHealthChecker implements HealthChecker {

    private static final String DEFAULT_HEALTH_PATH = "/health";
    private static final String USER_AGENT = "dephealth/0.5.0";

    private final String healthPath;
    private final boolean tlsEnabled;
    private final boolean tlsSkipVerify;
    private final Map<String, String> headers;

    private HttpHealthChecker(Builder builder) {
        this.healthPath = builder.healthPath;
        this.tlsEnabled = builder.tlsEnabled;
        this.tlsSkipVerify = builder.tlsSkipVerify;
        this.headers = Collections.unmodifiableMap(new LinkedHashMap<>(builder.resolvedHeaders));
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

        HttpClient.Builder clientBuilder = HttpClient.newBuilder()
                .connectTimeout(timeout)
                .followRedirects(HttpClient.Redirect.NORMAL);

        if (tlsEnabled && tlsSkipVerify) {
            clientBuilder.sslContext(InsecureSslContext.create());
        }

        HttpClient client = clientBuilder.build();

        HttpRequest.Builder requestBuilder = HttpRequest.newBuilder()
                .uri(uri)
                .timeout(timeout)
                .header("User-Agent", USER_AGENT)
                .GET();

        // Apply custom headers after User-Agent so they can override it.
        for (Map.Entry<String, String> entry : headers.entrySet()) {
            requestBuilder.header(entry.getKey(), entry.getValue());
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

        /** Builds the checker, validating auth configuration. */
        public HttpHealthChecker build() {
            validateAuthConflicts();
            buildResolvedHeaders();
            return new HttpHealthChecker(this);
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
