package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.EndpointStatus;
import biz.kryukov.dev.dephealth.StatusCategory;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.concurrent.ScheduledFuture;

/**
 * Thread-safe endpoint state: healthy/unhealthy, consecutive success/failure counters,
 * and fields for the HealthDetails() API.
 */
public final class EndpointState {

    private Boolean healthy;         // null = UNKNOWN
    private int consecutiveFailures;
    private int consecutiveSuccesses;

    // Dynamic fields for HealthDetails() API.
    private String lastStatus;
    private String lastDetail;
    private Duration lastLatency;
    private Instant lastCheckedAt;

    // Scheduled future for cancellation (used by dynamic endpoint management).
    private ScheduledFuture<?> future;

    // Static fields set at state creation time.
    private String depName;
    private DependencyType depType;
    private String host;
    private String port;
    private boolean critical;
    private Map<String, String> labels;

    public synchronized Boolean healthy() {
        return healthy;
    }

    public synchronized void recordSuccess(int successThreshold) {
        consecutiveFailures = 0;
        consecutiveSuccesses++;

        if (healthy == null) {
            // First check — immediate transition
            healthy = true;
            return;
        }

        if (!healthy && consecutiveSuccesses >= successThreshold) {
            healthy = true;
        }
    }

    public synchronized void recordFailure(int failureThreshold) {
        consecutiveSuccesses = 0;
        consecutiveFailures++;

        if (healthy == null) {
            // First check — immediate transition
            healthy = false;
            return;
        }

        if (healthy && consecutiveFailures >= failureThreshold) {
            healthy = false;
        }
    }

    /**
     * Sets the static fields for this endpoint (called once during registration).
     */
    synchronized void setStaticFields(String depName, DependencyType depType, String host,
                                       String port, boolean critical,
                                       Map<String, String> labels) {
        this.depName = depName;
        this.depType = depType;
        this.host = host;
        this.port = port;
        this.critical = critical;
        this.labels = labels == null ? Map.of() : Map.copyOf(labels);
        // Set UNKNOWN defaults for dynamic fields.
        this.lastStatus = StatusCategory.UNKNOWN;
        this.lastDetail = StatusCategory.UNKNOWN;
        this.lastLatency = Duration.ZERO;
        this.lastCheckedAt = null;
    }

    /**
     * Stores the classification results from a health check.
     */
    synchronized void storeCheckResult(String status, String detail, Duration latency) {
        this.lastStatus = status;
        this.lastDetail = detail;
        this.lastLatency = latency;
        this.lastCheckedAt = Instant.now();
    }

    /**
     * Returns the scheduled future for this endpoint's periodic check.
     */
    synchronized ScheduledFuture<?> future() {
        return future;
    }

    /**
     * Sets the scheduled future for this endpoint's periodic check.
     */
    synchronized void setFuture(ScheduledFuture<?> future) {
        this.future = future;
    }

    /**
     * Creates an {@link EndpointStatus} snapshot of the current state.
     */
    synchronized EndpointStatus toEndpointStatus() {
        return new EndpointStatus(
                healthy,
                lastStatus,
                lastDetail,
                lastLatency,
                depType,
                depName,
                host,
                port,
                critical,
                lastCheckedAt,
                labels
        );
    }
}
