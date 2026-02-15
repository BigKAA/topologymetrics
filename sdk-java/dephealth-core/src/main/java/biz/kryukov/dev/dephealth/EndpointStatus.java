package biz.kryukov.dev.dephealth;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Objects;

/**
 * Detailed health check state for a single endpoint.
 * Returned by {@link DepHealth#healthDetails()}.
 *
 * <p>Contains all 11 fields defined in the specification (section 8).
 * Immutable: all fields are set at creation time.
 */
public final class EndpointStatus {

    private final Boolean healthy;
    private final String status;
    private final String detail;
    private final Duration latency;
    private final DependencyType type;
    private final String name;
    private final String host;
    private final String port;
    private final boolean critical;
    private final Instant lastCheckedAt;
    private final Map<String, String> labels;

    @SuppressWarnings("checkstyle:ParameterNumber")
    public EndpointStatus(Boolean healthy, String status, String detail, Duration latency,
                   DependencyType type, String name, String host, String port,
                   boolean critical, Instant lastCheckedAt, Map<String, String> labels) {
        this.healthy = healthy;
        this.status = Objects.requireNonNull(status, "status");
        this.detail = Objects.requireNonNull(detail, "detail");
        this.latency = Objects.requireNonNull(latency, "latency");
        this.type = Objects.requireNonNull(type, "type");
        this.name = Objects.requireNonNull(name, "name");
        this.host = Objects.requireNonNull(host, "host");
        this.port = Objects.requireNonNull(port, "port");
        this.critical = critical;
        this.lastCheckedAt = lastCheckedAt; // null before first check
        this.labels = labels == null ? Map.of() : Map.copyOf(labels);
    }

    /** Health status: {@code true} = healthy, {@code false} = unhealthy, {@code null} = unknown. */
    public Boolean healthy() {
        return healthy;
    }

    /** Status category (e.g. "ok", "timeout", "unknown"). */
    public String status() {
        return status;
    }

    /** Detailed reason string (e.g. "ok", "timeout", "connection_refused"). */
    public String detail() {
        return detail;
    }

    /** Latency of the last health check. */
    public Duration latency() {
        return latency;
    }

    /** Latency in milliseconds as a floating-point value. */
    public double latencyMillis() {
        return latency.toNanos() / 1_000_000.0;
    }

    /** Dependency type (e.g. HTTP, POSTGRES). */
    public DependencyType type() {
        return type;
    }

    /** Dependency name. */
    public String name() {
        return name;
    }

    /** Endpoint host. */
    public String host() {
        return host;
    }

    /** Endpoint port. */
    public String port() {
        return port;
    }

    /** Whether the dependency is marked as critical. */
    public boolean critical() {
        return critical;
    }

    /** Timestamp of the last check, or {@code null} if not yet checked. */
    public Instant lastCheckedAt() {
        return lastCheckedAt;
    }

    /** Custom labels (unmodifiable). */
    public Map<String, String> labels() {
        return labels;
    }
}
