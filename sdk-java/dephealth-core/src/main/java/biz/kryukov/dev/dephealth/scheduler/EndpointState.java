package biz.kryukov.dev.dephealth.scheduler;

/**
 * Thread-safe состояние эндпоинта: healthy/unhealthy, счётчики последовательных успехов/неудач.
 */
public final class EndpointState {

    private Boolean healthy;         // null = UNKNOWN
    private int consecutiveFailures;
    private int consecutiveSuccesses;

    public synchronized Boolean healthy() {
        return healthy;
    }

    public synchronized void recordSuccess(int successThreshold) {
        consecutiveFailures = 0;
        consecutiveSuccesses++;

        if (healthy == null) {
            // Первая проверка — немедленный переход
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
            // Первая проверка — немедленный переход
            healthy = false;
            return;
        }

        if (healthy && consecutiveFailures >= failureThreshold) {
            healthy = false;
        }
    }
}
