package biz.kryukov.dev.dephealth;

import java.util.Collections;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Dependency endpoint (host + port + labels). Immutable.
 */
public final class Endpoint {

    /** Label name pattern: starts with a letter or _, followed by letters, digits, _. */
    private static final Pattern LABEL_NAME_PATTERN =
            Pattern.compile("^[a-zA-Z_][a-zA-Z0-9_]*$");

    /** Labels reserved by the SDK — cannot be overridden via WithLabel/label(). */
    public static final Set<String> RESERVED_LABELS = Set.of(
            "name", "group", "dependency", "type", "host", "port", "critical"
    );

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

    /**
     * Returns custom labels of the endpoint.
     *
     * @return unmodifiable map of labels
     */
    public Map<String, String> labels() {
        return metadata;
    }

    /**
     * @deprecated Use {@link #labels()} instead.
     */
    @Deprecated
    public Map<String, String> metadata() {
        return metadata;
    }

    /**
     * Validates the label name against the Prometheus naming pattern.
     *
     * @param name label name
     * @throws ValidationException if the name does not match the pattern or is reserved
     */
    public static void validateLabelName(String name) {
        Objects.requireNonNull(name, "label name");
        if (!LABEL_NAME_PATTERN.matcher(name).matches()) {
            throw new ValidationException(
                    "label name must match " + LABEL_NAME_PATTERN.pattern()
                            + ", got '" + name + "'");
        }
        if (RESERVED_LABELS.contains(name)) {
            throw new ValidationException(
                    "label name '" + name + "' is reserved and cannot be used as a custom label");
        }
    }

    /**
     * Validates all labels in the map: names and reserved status.
     *
     * @param labels label map
     * @throws ValidationException if any name is invalid
     */
    public static void validateLabels(Map<String, String> labels) {
        if (labels == null) {
            return;
        }
        for (String key : labels.keySet()) {
            validateLabelName(key);
        }
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
