package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.CheckConfig;
import biz.kryukov.dev.dephealth.CheckResult;
import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.EndpointStatus;
import biz.kryukov.dev.dephealth.ErrorClassifier;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.metrics.MetricsExporter;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.TimeUnit;

/**
 * Scheduler for periodic dependency health checks.
 *
 * <p>Supports both static dependencies (registered before start via {@link #addDependency})
 * and dynamic endpoints (added/removed/updated at runtime after start).
 */
public final class CheckScheduler {

    private static final Logger LOG = LoggerFactory.getLogger(CheckScheduler.class);
    private static final int MIN_CORE_POOL_SIZE = 1;

    private final MetricsExporter metrics;
    private final CheckConfig globalConfig;
    private final Logger logger;
    private final List<ScheduledDep> deps = new ArrayList<>();
    private final Map<String, EndpointState> states = new ConcurrentHashMap<>();

    private ScheduledThreadPoolExecutor executor;
    private volatile boolean started;
    private volatile boolean stopped;

    public CheckScheduler(MetricsExporter metrics, CheckConfig globalConfig) {
        this(metrics, globalConfig, LOG);
    }

    public CheckScheduler(MetricsExporter metrics, CheckConfig globalConfig, Logger logger) {
        this.metrics = metrics;
        this.globalConfig = globalConfig;
        this.logger = logger;
    }

    /**
     * Returns the global check configuration used for dynamic endpoints.
     */
    public CheckConfig globalConfig() {
        return globalConfig;
    }

    /**
     * Registers a dependency for periodic checking (before start).
     */
    public void addDependency(Dependency dependency, HealthChecker checker) {
        if (started) {
            throw new IllegalStateException("Cannot add dependency after scheduler started");
        }
        deps.add(new ScheduledDep(dependency, checker));
        for (Endpoint ep : dependency.endpoints()) {
            String key = stateKey(dependency.name(), ep);
            EndpointState state = new EndpointState();
            state.setStaticFields(dependency.name(), dependency.type(),
                    ep.host(), ep.port(), dependency.critical(), ep.labels());
            states.put(key, state);
        }
    }

    /**
     * Starts periodic health checks.
     */
    public synchronized void start() {
        if (started) {
            throw new IllegalStateException("Scheduler already started");
        }
        if (stopped) {
            throw new IllegalStateException("Scheduler already stopped");
        }
        started = true;

        int threadCount = Math.max(MIN_CORE_POOL_SIZE, deps.stream()
                .mapToInt(d -> d.dependency.endpoints().size())
                .sum());

        executor = new ScheduledThreadPoolExecutor(threadCount, r -> {
            Thread t = new Thread(r, "dephealth-scheduler");
            t.setDaemon(true);
            return t;
        });

        if (deps.isEmpty()) {
            logger.info("dephealth: scheduler started, 0 dependencies, 0 endpoints");
            return;
        }

        for (ScheduledDep dep : deps) {
            for (Endpoint ep : dep.dependency.endpoints()) {
                CheckConfig config = dep.dependency.config();
                String key = stateKey(dep.dependency.name(), ep);
                EndpointState state = states.get(key);

                ScheduledFuture<?> future = executor.scheduleAtFixedRate(
                        () -> runCheck(dep.dependency, dep.checker, ep, config),
                        config.initialDelay().toMillis(),
                        config.interval().toMillis(),
                        TimeUnit.MILLISECONDS
                );
                state.setFuture(future);
            }
        }

        logger.info("dephealth: scheduler started, {} dependencies, {} endpoints",
                deps.size(), states.size());
    }

    /**
     * Stops all health checks.
     */
    public synchronized void stop() {
        if (!started || stopped) {
            return;
        }
        stopped = true;

        for (EndpointState state : states.values()) {
            ScheduledFuture<?> f = state.future();
            if (f != null) {
                f.cancel(false);
            }
        }
        if (executor != null) {
            executor.shutdown();
            try {
                if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                    executor.shutdownNow();
                }
            } catch (InterruptedException e) {
                executor.shutdownNow();
                Thread.currentThread().interrupt();
            }
        }

        logger.info("dephealth: scheduler stopped");
    }

    /**
     * Returns the current health status of all endpoints.
     */
    public Map<String, Boolean> health() {
        Map<String, Boolean> result = new LinkedHashMap<>();
        for (Map.Entry<String, EndpointState> entry : states.entrySet()) {
            Boolean healthy = entry.getValue().healthy();
            if (healthy != null) {
                result.put(entry.getKey(), healthy);
            }
        }
        return result;
    }

    /**
     * Returns the detailed health status of all endpoints (including UNKNOWN).
     */
    public Map<String, EndpointStatus> healthDetails() {
        Map<String, EndpointStatus> result = new LinkedHashMap<>();
        for (Map.Entry<String, EndpointState> entry : states.entrySet()) {
            result.put(entry.getKey(), entry.getValue().toEndpointStatus());
        }
        return result;
    }

    private void runCheck(Dependency dep, HealthChecker checker, Endpoint ep,
                          CheckConfig config) {
        String key = stateKey(dep.name(), ep);
        EndpointState state = states.get(key);
        if (state == null) {
            return;
        }
        long startNs = System.nanoTime();

        try {
            safeCheck(checker, ep, config.timeout());
            long durationNs = System.nanoTime() - startNs;
            Duration duration = Duration.ofNanos(durationNs);

            Boolean wasBefore = state.healthy();
            state.recordSuccess(config.successThreshold());

            metrics.setHealth(dep, ep, 1.0);
            metrics.observeLatency(dep, ep, duration);

            // Classify success.
            CheckResult result = ErrorClassifier.classify(null);
            metrics.setStatus(dep, ep, result.category());
            metrics.setStatusDetail(dep, ep, result.detail());

            // Store classification results for HealthDetails() API.
            state.storeCheckResult(result.category(), result.detail(), duration);

            if (wasBefore != null && !wasBefore && Boolean.TRUE.equals(state.healthy())) {
                logger.info("dephealth: {} [{}] recovered", dep.name(), ep);
            }
        } catch (Exception e) {
            long durationNs = System.nanoTime() - startNs;
            Duration duration = Duration.ofNanos(durationNs);

            Boolean wasBefore = state.healthy();
            state.recordFailure(config.failureThreshold());

            metrics.setHealth(dep, ep, 0.0);
            metrics.observeLatency(dep, ep, duration);

            // Classify error.
            CheckResult result = ErrorClassifier.classify(e);
            metrics.setStatus(dep, ep, result.category());
            metrics.setStatusDetail(dep, ep, result.detail());

            // Store classification results for HealthDetails() API.
            state.storeCheckResult(result.category(), result.detail(), duration);

            if (wasBefore == null || wasBefore) {
                String msg = e.getMessage() != null ? e.getMessage()
                        : e.getClass().getName();
                Throwable cause = e.getCause();
                if (cause != null) {
                    msg += " (cause: " + (cause.getMessage() != null
                            ? cause.getMessage() : cause.getClass().getName()) + ")";
                }
                logger.warn("dephealth: {} [{}] check failed: {}",
                        dep.name(), ep, msg);
            }
            if (wasBefore != null && wasBefore && Boolean.FALSE.equals(state.healthy())) {
                logger.error("dephealth: {} [{}] became unhealthy", dep.name(), ep);
            }
        }
    }

    private void safeCheck(HealthChecker checker, Endpoint ep, Duration timeout) throws Exception {
        try {
            checker.check(ep, timeout);
        } catch (Exception e) {
            throw e;
        } catch (Throwable t) {
            logger.error("dephealth: panic in health checker", t);
            throw new RuntimeException("panic in health checker: " + t.getMessage(), t);
        }
    }

    static String stateKey(String name, Endpoint ep) {
        return name + ":" + ep.host() + ":" + ep.port();
    }

    static String stateKey(String name, String host, String port) {
        return name + ":" + host + ":" + port;
    }

    private record ScheduledDep(Dependency dependency, HealthChecker checker) {}
}
