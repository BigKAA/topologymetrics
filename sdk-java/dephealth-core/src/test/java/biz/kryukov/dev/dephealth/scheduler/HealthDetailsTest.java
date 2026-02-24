package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.CheckConfig;
import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.EndpointStatus;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.StatusCategory;
import biz.kryukov.dev.dephealth.metrics.MetricsExporter;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

class HealthDetailsTest {

    private MetricsExporter metrics;
    private CheckScheduler scheduler;

    @BeforeEach
    void setUp() {
        metrics = new MetricsExporter(new SimpleMeterRegistry(), "test-app", "test-group");
        scheduler = new CheckScheduler(metrics, CheckConfig.defaults());
    }

    @Test
    void emptyBeforeAddingDependencies() {
        assertTrue(scheduler.healthDetails().isEmpty());
    }

    @Test
    void unknownStateBeforeFirstCheck() throws Exception {
        CountDownLatch started = new CountDownLatch(1);
        // Checker that blocks forever.
        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) throws InterruptedException {
                started.countDown();
                Thread.sleep(60_000);
            }

            @Override
            public DependencyType type() {
                return DependencyType.POSTGRES;
            }
        };

        Dependency dep = Dependency.builder("test-dep", DependencyType.POSTGRES)
                .endpoint(new Endpoint("pg.svc", "5432", Map.of("role", "primary")))
                .critical(true)
                .config(CheckConfig.builder()
                        .interval(Duration.ofSeconds(10))
                        .timeout(Duration.ofSeconds(5))
                        .initialDelay(Duration.ZERO)
                        .build())
                .build();

        scheduler.addDependency(dep, checker);
        scheduler.start();

        // Wait for check goroutine to start (but not complete).
        assertTrue(started.await(3, TimeUnit.SECONDS));
        Thread.sleep(50);

        Map<String, EndpointStatus> details = scheduler.healthDetails();
        assertEquals(1, details.size());

        String key = "test-dep:pg.svc:5432";
        EndpointStatus es = details.get(key);
        assertNotNull(es);

        // UNKNOWN state checks.
        assertNull(es.healthy());
        assertEquals(StatusCategory.UNKNOWN, es.status());
        assertEquals(StatusCategory.UNKNOWN, es.detail());
        assertEquals(Duration.ZERO, es.latency());
        assertNull(es.lastCheckedAt());

        // Static fields should be populated.
        assertEquals(DependencyType.POSTGRES, es.type());
        assertEquals("test-dep", es.name());
        assertEquals("pg.svc", es.host());
        assertEquals("5432", es.port());
        assertTrue(es.critical());
        assertEquals("primary", es.labels().get("role"));

        scheduler.stop();
    }

    @Test
    void healthyEndpoint() throws Exception {
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

        Dependency dep = Dependency.builder("api-gw", DependencyType.HTTP)
                .endpoint(new Endpoint("api.svc", "8080"))
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

        Map<String, EndpointStatus> details = scheduler.healthDetails();
        String key = "api-gw:api.svc:8080";
        EndpointStatus es = details.get(key);
        assertNotNull(es);

        assertEquals(Boolean.TRUE, es.healthy());
        assertEquals(StatusCategory.OK, es.status());
        assertEquals(StatusCategory.OK, es.detail());
        assertTrue(es.latency().toNanos() > 0);
        assertNotNull(es.lastCheckedAt());
        assertEquals(DependencyType.HTTP, es.type());
        assertEquals("api-gw", es.name());
        assertTrue(es.critical());

        scheduler.stop();
    }

    @Test
    void unhealthyEndpoint() throws Exception {
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

        Dependency dep = Dependency.builder("api-gw", DependencyType.HTTP)
                .endpoint(new Endpoint("api.svc", "8080"))
                .critical(false)
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

        Map<String, EndpointStatus> details = scheduler.healthDetails();
        String key = "api-gw:api.svc:8080";
        EndpointStatus es = details.get(key);
        assertNotNull(es);

        assertEquals(Boolean.FALSE, es.healthy());
        assertEquals(StatusCategory.ERROR, es.status());
        assertEquals("error", es.detail());
        assertTrue(es.latency().toNanos() > 0);
        assertNotNull(es.lastCheckedAt());
        assertFalse(es.critical());

        scheduler.stop();
    }

    @Test
    void keysMatchHealth() throws Exception {
        CountDownLatch latch = new CountDownLatch(2);
        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.TCP;
            }
        };

        Dependency dep = Dependency.builder("multi-ep", DependencyType.TCP)
                .endpoints(java.util.List.of(
                        new Endpoint("host-1", "1111"),
                        new Endpoint("host-2", "2222")
                ))
                .critical(false)
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
        Map<String, EndpointStatus> details = scheduler.healthDetails();

        // All keys from health() must be present in healthDetails().
        for (String key : health.keySet()) {
            assertTrue(details.containsKey(key),
                    "key " + key + " in health() but not in healthDetails()");
        }

        // healthDetails() includes all endpoints (same or more than health()).
        assertTrue(details.size() >= health.size());

        scheduler.stop();
    }

    @Test
    void concurrentAccess() throws Exception {
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
        Thread.sleep(100);

        // Launch concurrent readers.
        int threadCount = 10;
        int iterations = 100;
        CountDownLatch done = new CountDownLatch(threadCount);
        for (int i = 0; i < threadCount; i++) {
            new Thread(() -> {
                try {
                    for (int j = 0; j < iterations; j++) {
                        Map<String, EndpointStatus> details = scheduler.healthDetails();
                        assertNotNull(details);
                        assertFalse(details.isEmpty());
                    }
                } finally {
                    done.countDown();
                }
            }).start();
        }

        assertTrue(done.await(5, TimeUnit.SECONDS));
        scheduler.stop();
    }

    @Test
    void afterStop() throws Exception {
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
        Thread.sleep(100);
        scheduler.stop();

        // After stop, healthDetails should return last known state.
        Map<String, EndpointStatus> details = scheduler.healthDetails();
        assertFalse(details.isEmpty());

        String key = "test:localhost:8080";
        EndpointStatus es = details.get(key);
        assertNotNull(es);
        assertEquals(Boolean.TRUE, es.healthy());
    }

    @Test
    void labelsEmptyWhenNotSet() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);
        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.TCP;
            }
        };

        Dependency dep = Dependency.builder("test", DependencyType.TCP)
                .endpoint(new Endpoint("localhost", "9090"))
                .critical(false)
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

        Map<String, EndpointStatus> details = scheduler.healthDetails();
        EndpointStatus es = details.get("test:localhost:9090");
        assertNotNull(es);

        // Labels should be empty map, not null.
        assertNotNull(es.labels());
        assertTrue(es.labels().isEmpty());

        scheduler.stop();
    }

    @Test
    void resultMapIsIndependent() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);
        HealthChecker checker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.TCP;
            }
        };

        Dependency dep = Dependency.builder("test", DependencyType.TCP)
                .endpoint(new Endpoint("localhost", "9090"))
                .critical(false)
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

        Map<String, EndpointStatus> details = scheduler.healthDetails();
        String key = "test:localhost:9090";

        // Modify the returned map — should not affect internal state.
        details.remove(key);

        // Get fresh details — should be unaffected.
        Map<String, EndpointStatus> details2 = scheduler.healthDetails();
        assertTrue(details2.containsKey(key));
        assertEquals("test", details2.get(key).name());

        scheduler.stop();
    }

    @Test
    void latencyMillis() {
        EndpointStatus es = new EndpointStatus(
                true, StatusCategory.OK, "ok", Duration.ofNanos(2_500_000),
                DependencyType.HTTP, "test", "localhost", "8080",
                true, null, Map.of()
        );
        assertEquals(2.5, es.latencyMillis(), 0.001);
    }
}
