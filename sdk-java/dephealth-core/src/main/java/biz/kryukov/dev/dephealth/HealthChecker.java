package biz.kryukov.dev.dephealth;

import java.time.Duration;

/**
 * Dependency health check interface.
 *
 * <p>Implementations must be thread-safe. Stateful checkers (e.g. those caching
 * connections or clients) should override {@link #close()} to release resources.</p>
 */
public interface HealthChecker extends AutoCloseable {

    /**
     * Performs a health check on the endpoint.
     *
     * @param endpoint endpoint to check
     * @param timeout  maximum wait time
     * @throws Exception if the dependency is unhealthy or the check failed
     */
    void check(Endpoint endpoint, Duration timeout) throws Exception;

    /** Returns the dependency type. */
    DependencyType type();

    /**
     * Releases resources held by this checker (connections, clients, channels).
     * Default implementation is a no-op for stateless checkers.
     */
    @Override
    default void close() {
        // no-op by default
    }
}
