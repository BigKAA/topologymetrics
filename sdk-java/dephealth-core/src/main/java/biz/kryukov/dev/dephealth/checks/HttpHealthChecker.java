package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

/**
 * HTTP health checker — выполняет GET-запрос к healthPath и ожидает 2xx.
 */
public final class HttpHealthChecker implements HealthChecker {

    private static final String DEFAULT_HEALTH_PATH = "/health";
    private static final String USER_AGENT = "dephealth/0.2.1";

    private final String healthPath;
    private final boolean tlsEnabled;
    private final boolean tlsSkipVerify;

    private HttpHealthChecker(Builder builder) {
        this.healthPath = builder.healthPath;
        this.tlsEnabled = builder.tlsEnabled;
        this.tlsSkipVerify = builder.tlsSkipVerify;
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

        HttpRequest request = HttpRequest.newBuilder()
                .uri(uri)
                .timeout(timeout)
                .header("User-Agent", USER_AGENT)
                .GET()
                .build();

        HttpResponse<Void> response = client.send(request, HttpResponse.BodyHandlers.discarding());

        int status = response.statusCode();
        if (status < 200 || status >= 300) {
            throw new Exception("HTTP health check failed: status " + status);
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.HTTP;
    }

    public String healthPath() {
        return healthPath;
    }

    public boolean tlsEnabled() {
        return tlsEnabled;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String healthPath = DEFAULT_HEALTH_PATH;
        private boolean tlsEnabled;
        private boolean tlsSkipVerify;

        private Builder() {}

        public Builder healthPath(String healthPath) {
            this.healthPath = healthPath;
            return this;
        }

        public Builder tlsEnabled(boolean tlsEnabled) {
            this.tlsEnabled = tlsEnabled;
            return this;
        }

        public Builder tlsSkipVerify(boolean tlsSkipVerify) {
            this.tlsSkipVerify = tlsSkipVerify;
            return this;
        }

        public HttpHealthChecker build() {
            return new HttpHealthChecker(this);
        }
    }
}
