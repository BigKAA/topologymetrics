package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.CheckConfig;
import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.EndpointNotFoundException;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.metrics.MetricsExporter;

import io.micrometer.core.instrument.Meter;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Collection;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.*;

class CheckSchedulerTest {

    private static final CheckConfig FAST_CONFIG = CheckConfig.builder()
            .interval(Duration.ofSeconds(1))
            .timeout(Duration.ofMillis(500))
            .initialDelay(Duration.ZERO)
            .build();

    private SimpleMeterRegistry registry;
    private MetricsExporter metrics;
    private CheckScheduler scheduler;

    @BeforeEach
    void setUp() {
        registry = new SimpleMeterRegistry();
        metrics = new MetricsExporter(registry, "test-app", "test-group");
        scheduler = new CheckScheduler(metrics, CheckConfig.defaults());
    }

    @Test
    void startAndStop() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);

        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };

        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "8080"))
                .critical(true)
                .config(CheckConfig.builder()
                        .interval(Duration.ofSeconds(1))
                        .timeout(Duration.ofMillis(500))
                        .initialDelay(Duration.ZERO)
                        .build())
                .build();

        scheduler.addDependency(dep, checker);
        scheduler.start();

        assertTrue(latch.await(3, TimeUnit.SECONDS));

        scheduler.stop();
    }

    @Test
    void healthReturnsStates() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);
        AtomicInteger callCount = new AtomicInteger(0);

        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                callCount.incrementAndGet();
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };

        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "8080"))
                .critical(true)
                .config(CheckConfig.builder()
                        .interval(Duration.ofSeconds(1))
                        .timeout(Duration.ofMillis(500))
                        .initialDelay(Duration.ZERO)
                        .build())
                .build();

        scheduler.addDependency(dep, checker);
        scheduler.start();

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        // Allow time to update state
        Thread.sleep(100);

        Map<String, Boolean> health = scheduler.health();
        assertTrue(health.containsKey("test:localhost:8080"));
        assertTrue(health.get("test:localhost:8080"));

        scheduler.stop();
    }

    @Test
    void failedCheckMarksUnhealthy() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);

        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) throws Exception {
                latch.countDown();
                throw new Exception("Connection refused");
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };

        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "8080"))
                .critical(true)
                .config(CheckConfig.builder()
                        .interval(Duration.ofSeconds(1))
                        .timeout(Duration.ofMillis(500))
                        .initialDelay(Duration.ZERO)
                        .build())
                .build();

        scheduler.addDependency(dep, checker);
        scheduler.start();

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        Map<String, Boolean> health = scheduler.health();
        assertTrue(health.containsKey("test:localhost:8080"));
        assertFalse(health.get("test:localhost:8080"));

        scheduler.stop();
    }

    @Test
    void doubleStartThrows() {
        Dependency dep = Dependency.builder("test", DependencyType.TCP)
                .endpoint(new Endpoint("localhost", "80"))
                .critical(true)
                .config(CheckConfig.builder()
                        .interval(Duration.ofSeconds(10))
                        .timeout(Duration.ofSeconds(5))
                        .initialDelay(Duration.ZERO)
                        .build())
                .build();

        scheduler.addDependency(dep, new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {}

            @Override
            public DependencyType type() {
                return DependencyType.TCP;
            }
        });
        scheduler.start();
        assertThrows(IllegalStateException.class, scheduler::start);
        scheduler.stop();
    }

    @Test
    void emptyHealthBeforeStart() {
        assertTrue(scheduler.health().isEmpty());
    }

    // ---- Dynamic endpoint tests ----

    /**
     * Helper: creates a CheckScheduler with fast globalConfig (initialDelay=0, interval=1s).
     */
    private CheckScheduler fastScheduler() {
        return new CheckScheduler(metrics, FAST_CONFIG);
    }

    private HealthChecker successChecker(CountDownLatch latch) {
        return new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };
    }

    private HealthChecker failChecker(CountDownLatch latch) {
        return new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) throws Exception {
                latch.countDown();
                throw new Exception("Connection refused");
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };
    }

    private HealthChecker noopChecker() {
        return new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {}

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };
    }

    @Test
    void testAddEndpoint() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, successChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        Map<String, Boolean> health = sched.health();
        assertTrue(health.containsKey("dyn:dynhost:9090"));
        assertTrue(health.get("dyn:dynhost:9090"));

        sched.stop();
    }

    @Test
    void testAddEndpoint_Idempotent() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        HealthChecker checker = successChecker(latch);

        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, checker);
        // Second add with same key — should not throw
        sched.addEndpoint("dyn", DependencyType.HTTP, false, ep, noopChecker());

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        // Only one entry
        Map<String, Boolean> health = sched.health();
        assertEquals(1, health.size());
        assertTrue(health.containsKey("dyn:dynhost:9090"));

        sched.stop();
    }

    @Test
    void testAddEndpoint_BeforeStart() {
        CheckScheduler sched = fastScheduler();
        Endpoint ep = new Endpoint("dynhost", "9090");

        assertThrows(IllegalStateException.class, () ->
                sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_AfterStop() {
        CheckScheduler sched = fastScheduler();
        sched.start();
        sched.stop();

        Endpoint ep = new Endpoint("dynhost", "9090");
        assertThrows(IllegalStateException.class, () ->
                sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_Metrics() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, successChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        // Verify health gauge registered with correct tags
        assertNotNull(registry.find("app_dependency_health")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .gauge());

        // Verify latency summary registered
        assertNotNull(registry.find("app_dependency_latency_seconds")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .summary());

        // Verify status gauges registered (at least one of the 8)
        assertFalse(registry.find("app_dependency_status")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .gauges().isEmpty());

        sched.stop();
    }

    @Test
    void testRemoveEndpoint() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, successChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);
        assertTrue(sched.health().containsKey("dyn:dynhost:9090"));

        sched.removeEndpoint("dyn", "dynhost", "9090");

        assertFalse(sched.health().containsKey("dyn:dynhost:9090"));

        sched.stop();
    }

    @Test
    void testRemoveEndpoint_Idempotent() {
        CheckScheduler sched = fastScheduler();
        sched.start();

        // Removing non-existent endpoint — no error
        sched.removeEndpoint("nonexistent", "nohost", "1234");

        sched.stop();
    }

    @Test
    void testRemoveEndpoint_MetricsDeleted() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, successChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        // Metrics should exist before removal
        assertNotNull(registry.find("app_dependency_health")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .gauge());

        sched.removeEndpoint("dyn", "dynhost", "9090");

        // All metric series should be removed
        assertNull(registry.find("app_dependency_health")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .gauge());

        assertNull(registry.find("app_dependency_latency_seconds")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .summary());

        assertTrue(registry.find("app_dependency_status")
                .tag("dependency", "dyn")
                .tag("host", "dynhost")
                .tag("port", "9090")
                .gauges().isEmpty());

        sched.stop();
    }

    @Test
    void testRemoveEndpoint_BeforeStart() {
        CheckScheduler sched = fastScheduler();

        assertThrows(IllegalStateException.class, () ->
                sched.removeEndpoint("dyn", "dynhost", "9090"));
    }

    @Test
    void testUpdateEndpoint() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        // Add initial endpoint
        CountDownLatch addLatch = new CountDownLatch(1);
        Endpoint oldEp = new Endpoint("oldhost", "8080");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, oldEp, successChecker(addLatch));

        assertTrue(addLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);
        assertTrue(sched.health().containsKey("dyn:oldhost:8080"));

        // Update to new endpoint
        CountDownLatch updateLatch = new CountDownLatch(1);
        Endpoint newEp = new Endpoint("newhost", "9090");
        sched.updateEndpoint("dyn", "oldhost", "8080", newEp, successChecker(updateLatch));

        assertTrue(updateLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        Map<String, Boolean> health = sched.health();
        assertFalse(health.containsKey("dyn:oldhost:8080"));
        assertTrue(health.containsKey("dyn:newhost:9090"));
        assertTrue(health.get("dyn:newhost:9090"));

        sched.stop();
    }

    @Test
    void testUpdateEndpoint_NotFound() {
        CheckScheduler sched = fastScheduler();
        sched.start();

        Endpoint newEp = new Endpoint("newhost", "9090");
        assertThrows(EndpointNotFoundException.class, () ->
                sched.updateEndpoint("dyn", "nohost", "1234", newEp, noopChecker()));

        sched.stop();
    }

    @Test
    void testUpdateEndpoint_MetricsSwap() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        // Add and wait for first check
        CountDownLatch addLatch = new CountDownLatch(1);
        Endpoint oldEp = new Endpoint("oldhost", "8080");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, oldEp, successChecker(addLatch));

        assertTrue(addLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        // Old metrics present
        assertNotNull(registry.find("app_dependency_health")
                .tag("host", "oldhost").tag("port", "8080").gauge());

        // Update endpoint
        CountDownLatch updateLatch = new CountDownLatch(1);
        Endpoint newEp = new Endpoint("newhost", "9090");
        sched.updateEndpoint("dyn", "oldhost", "8080", newEp, successChecker(updateLatch));

        assertTrue(updateLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(100);

        // Old metrics deleted
        assertNull(registry.find("app_dependency_health")
                .tag("host", "oldhost").tag("port", "8080").gauge());

        // New metrics present
        assertNotNull(registry.find("app_dependency_health")
                .tag("host", "newhost").tag("port", "9090").gauge());

        sched.stop();
    }

    @Test
    void testStopAfterDynamicAdd() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        sched.addEndpoint("dyn", DependencyType.HTTP, true, ep, successChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));

        // Stop should complete cleanly without exceptions
        sched.stop();
    }

    @Test
    void testConcurrentAddRemoveHealth() throws Exception {
        CheckScheduler sched = fastScheduler();
        sched.start();

        int threadCount = 8;
        int opsPerThread = 50;
        ExecutorService pool = Executors.newFixedThreadPool(threadCount);
        CountDownLatch startGate = new CountDownLatch(1);
        CountDownLatch doneLatch = new CountDownLatch(threadCount);
        AtomicBoolean failed = new AtomicBoolean(false);

        for (int t = 0; t < threadCount; t++) {
            final int threadId = t;
            pool.submit(() -> {
                try {
                    startGate.await();
                    for (int i = 0; i < opsPerThread; i++) {
                        String host = "host-" + threadId + "-" + i;
                        Endpoint ep = new Endpoint(host, "8080");
                        sched.addEndpoint("conc", DependencyType.TCP, false,
                                ep, noopChecker());
                        sched.health();
                        sched.healthDetails();
                        sched.removeEndpoint("conc", host, "8080");
                    }
                } catch (Exception e) {
                    failed.set(true);
                } finally {
                    doneLatch.countDown();
                }
            });
        }

        startGate.countDown();
        assertTrue(doneLatch.await(30, TimeUnit.SECONDS));
        assertFalse(failed.get(), "Concurrent operations should not throw exceptions");

        pool.shutdown();
        sched.stop();
    }
}
