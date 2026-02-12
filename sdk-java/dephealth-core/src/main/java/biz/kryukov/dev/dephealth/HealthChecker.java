package biz.kryukov.dev.dephealth;

import java.time.Duration;

/**
 * Dependency health check interface.
 *
 * <p>Implementations must be thread-safe.</p>
 */
public interface HealthChecker {

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
}
