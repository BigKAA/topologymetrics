package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.CheckConfig;
import biz.kryukov.dev.dephealth.CheckResult;
import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.Endpoint;
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
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

/**
 * Scheduler for periodic dependency health checks.
 */
public final class CheckScheduler {

    private static final Logger LOG = LoggerFactory.getLogger(CheckScheduler.class);

    private final MetricsExporter metrics;
    private final Logger logger;
    private final List<ScheduledDep> deps = new ArrayList<>();
    private final Map<String, EndpointState> states = new LinkedHashMap<>();

    private ScheduledExecutorService executor;
    private final List<ScheduledFuture<?>> futures = new ArrayList<>();
    private volatile boolean started;
    private volatile boolean stopped;

    public CheckScheduler(MetricsExporter metrics) {
        this(metrics, LOG);
    }

    public CheckScheduler(MetricsExporter metrics, Logger logger) {
        this.metrics = metrics;
        this.logger = logger;
    }

    /**
     * Registers a dependency for periodic checking.
     */
    public void addDependency(Dependency dependency, HealthChecker checker) {
        if (started) {
            throw new IllegalStateException("Cannot add dependency after scheduler started");
        }
        deps.add(new ScheduledDep(dependency, checker));
        for (Endpoint ep : dependency.endpoints()) {
            String key = stateKey(dependency.name(), ep);
            states.put(key, new EndpointState());
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

        if (deps.isEmpty()) {
            logger.info("dephealth: scheduler started, 0 dependencies, 0 endpoints");
            return;
        }

        int threadCount = Math.max(1, deps.stream()
                .mapToInt(d -> d.dependency.endpoints().size())
                .sum());

        executor = Executors.newScheduledThreadPool(threadCount, r -> {
            Thread t = new Thread(r, "dephealth-scheduler");
            t.setDaemon(true);
            return t;
        });

        for (ScheduledDep dep : deps) {
            for (Endpoint ep : dep.dependency.endpoints()) {
                CheckConfig config = dep.dependency.config();
                long initialDelay = config.initialDelay().toMillis();
                long interval = config.interval().toMillis();

                ScheduledFuture<?> future = executor.scheduleAtFixedRate(
                        () -> runCheck(dep.dependency, dep.checker, ep, config),
                        initialDelay,
                        interval,
                        TimeUnit.MILLISECONDS
                );
                futures.add(future);
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

        for (ScheduledFuture<?> f : futures) {
            f.cancel(false);
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

    private void runCheck(Dependency dep, HealthChecker checker, Endpoint ep,
                          CheckConfig config) {
        String key = stateKey(dep.name(), ep);
        EndpointState state = states.get(key);
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

    private static String stateKey(String name, Endpoint ep) {
        return name + ":" + ep.host() + ":" + ep.port();
    }

    private record ScheduledDep(Dependency dependency, HealthChecker checker) {}
}
