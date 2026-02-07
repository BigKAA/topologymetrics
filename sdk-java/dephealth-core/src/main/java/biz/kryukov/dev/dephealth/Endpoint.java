package biz.kryukov.dev.dephealth;

import java.util.Collections;
import java.util.Map;
import java.util.Objects;

/**
 * Эндпоинт зависимости (хост + порт + метаданные). Immutable.
 */
public final class Endpoint {

    private final String host;
    private final String port;
    private final Map<String, String> metadata;

    public Endpoint(String host, String port) {
        this(host, port, Map.of());
    }

    public Endpoint(String host, String port, Map<String, String> metadata) {
        this.host = Objects.requireNonNull(host, "host");
        this.port = Objects.requireNonNull(port, "port");
        this.metadata = metadata == null ? Map.of() : Collections.unmodifiableMap(metadata);
    }

    public String host() {
        return host;
    }

    public String port() {
        return port;
    }

    public int portAsInt() {
        return Integer.parseInt(port);
    }

    public Map<String, String> metadata() {
        return metadata;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (!(o instanceof Endpoint ep)) {
            return false;
        }
        return host.equals(ep.host) && port.equals(ep.port);
    }

    @Override
    public int hashCode() {
        return Objects.hash(host, port);
    }

    @Override
    public String toString() {
        return host + ":" + port;
    }
}
