package biz.kryukov.dev.dephealth;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

class DepHealthTest {

    private SimpleMeterRegistry registry;

    /** Tracks a started DepHealth so AfterEach can stop it. */
    private DepHealth activeDh;

    @BeforeEach
    void setUp() {
        registry = new SimpleMeterRegistry();
    }

    @AfterEach
    void tearDown() {
        if (activeDh != null) {
            activeDh.stop();
            activeDh = null;
        }
    }

    @Test
    void builderWithNameAndUrl() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("test-http", DependencyType.HTTP, d -> d
                        .url("http://localhost:8080")
                        .critical(true))
                .build();
        assertNotNull(dh);
    }

    @Test
    void builderWithParams() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("test-tcp", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("8080")
                        .critical(false))
                .build();
        assertNotNull(dh);
    }

    @Test
    void missingNameThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("", "test-group", registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void nullNameThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder(null, "test-group", registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void invalidNameThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("INVALID_NAME", "test-group", registry));
    }

    @Test
    void missingGroupThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("test-app", "", registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void nullGroupThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("test-app", null, registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void invalidGroupThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("test-app", "INVALID_GROUP", registry));
    }

    @Test
    void noDependenciesAllowed() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry).build();
        assertNotNull(dh);

        // health() returns an empty collection
        Map<String, Boolean> health = dh.health();
        assertTrue(health.isEmpty());

        // start()/stop() — no-op
        dh.start();
        dh.stop();
    }

    @Test
    void missingCriticalThrows() {
        // critical not set -> error on build (Dependency.validate)
        assertThrows(ValidationException.class, () ->
                DepHealth.builder("test-app", "test-group", registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080"))
                        .build());
    }

    @Test
    void startStopCycle() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);

        HealthChecker mockChecker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };

        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(1))
                .dependency("test", DependencyType.HTTP, mockChecker, d -> d
                        .host("localhost")
                        .port("8080")
                        .critical(true))
                .build();

        dh.start();
        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);

        Map<String, Boolean> health = dh.health();
        assertFalse(health.isEmpty());

        dh.stop();
    }

    @Test
    void healthDetailsFacade() throws Exception {
        CountDownLatch latch = new CountDownLatch(1);

        HealthChecker mockChecker = new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {
                latch.countDown();
            }

            @Override
            public DependencyType type() {
                return DependencyType.HTTP;
            }
        };

        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(1))
                .dependency("test", DependencyType.HTTP, mockChecker, d -> d
                        .host("localhost")
                        .port("8080")
                        .critical(true))
                .build();

        // Before start — endpoints exist but with UNKNOWN status.
        Map<String, EndpointStatus> detailsBefore = dh.healthDetails();
        assertFalse(detailsBefore.isEmpty());
        EndpointStatus unknown = detailsBefore.get("test:localhost:8080");
        assertNotNull(unknown);
        assertNull(unknown.healthy());
        assertEquals(StatusCategory.UNKNOWN, unknown.status());

        dh.start();
        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);

        Map<String, EndpointStatus> details = dh.healthDetails();
        assertFalse(details.isEmpty());
        EndpointStatus es = details.get("test:localhost:8080");
        assertNotNull(es);
        assertEquals(Boolean.TRUE, es.healthy());
        assertEquals(StatusCategory.OK, es.status());

        dh.stop();
    }

    @Test
    void globalIntervalUsed() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(30))
                .dependency("test", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("80")
                        .critical(true))
                .build();
        assertNotNull(dh);
    }

    @Test
    void perDependencyIntervalOverridesGlobal() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(30))
                .dependency("test", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("80")
                        .critical(true)
                        .interval(Duration.ofSeconds(10)))
                .build();
        assertNotNull(dh);
    }

    @Test
    void jdbcUrlParsing() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .jdbcUrl("jdbc:postgresql://localhost:5432/db")
                        .critical(true))
                .build();
        assertNotNull(dh);
    }

    @Test
    void noEndpointConfigThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("test-app", "test-group", registry)
                        .dependency("test", DependencyType.HTTP, d -> d.critical(true))
                        .build());
    }

    @Test
    void withLabel() {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("test-http", DependencyType.HTTP, d -> d
                        .url("http://localhost:8080")
                        .critical(true)
                        .label("region", "us-east"))
                .build();
        assertNotNull(dh);
    }

    @Test
    void reservedLabelThrows() {
        assertThrows(ValidationException.class, () ->
                DepHealth.builder("test-app", "test-group", registry)
                        .dependency("test-http", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true)
                                .label("host", "bad")));
    }

    @Test
    void invalidLabelNameThrows() {
        assertThrows(ValidationException.class, () ->
                DepHealth.builder("test-app", "test-group", registry)
                        .dependency("test-http", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true)
                                .label("123invalid", "bad")));
    }

    // ---- Dynamic endpoint facade tests (Phase 5) ----

    private HealthChecker mockChecker(CountDownLatch latch) {
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

    private HealthChecker noopChecker() {
        return new HealthChecker() {
            @Override
            public void check(Endpoint endpoint, Duration timeout) {}

            @Override
            public DependencyType type() {
                return DependencyType.TCP;
            }
        };
    }

    private DepHealth fastDepHealth() {
        return DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(1))
                .build();
    }

    @Test
    void testAddEndpoint() throws Exception {
        activeDh = fastDepHealth();
        activeDh.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        activeDh.addEndpoint("dyn-svc", DependencyType.HTTP, true, ep, mockChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);

        Map<String, Boolean> health = activeDh.health();
        assertTrue(health.containsKey("dyn-svc:dynhost:9090"));
        assertTrue(health.get("dyn-svc:dynhost:9090"));
    }

    @Test
    void testAddEndpoint_InvalidName() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint ep = new Endpoint("host", "8080");
        assertThrows(ValidationException.class, () ->
                activeDh.addEndpoint("INVALID_NAME", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_InvalidType() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint ep = new Endpoint("host", "8080");
        assertThrows(ValidationException.class, () ->
                activeDh.addEndpoint("valid-name", null, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_MissingHost() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint ep = new Endpoint("", "8080");
        assertThrows(ValidationException.class, () ->
                activeDh.addEndpoint("valid-name", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_MissingPort() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint ep = new Endpoint("host", "");
        assertThrows(ValidationException.class, () ->
                activeDh.addEndpoint("valid-name", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testAddEndpoint_ReservedLabel() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint ep = new Endpoint("host", "8080", Map.of("host", "bad"));
        assertThrows(ValidationException.class, () ->
                activeDh.addEndpoint("valid-name", DependencyType.HTTP, true, ep, noopChecker()));
    }

    @Test
    void testRemoveEndpoint() throws Exception {
        activeDh = fastDepHealth();
        activeDh.start();

        CountDownLatch latch = new CountDownLatch(1);
        Endpoint ep = new Endpoint("dynhost", "9090");
        activeDh.addEndpoint("dyn-svc", DependencyType.HTTP, true, ep, mockChecker(latch));

        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);
        assertTrue(activeDh.health().containsKey("dyn-svc:dynhost:9090"));

        activeDh.removeEndpoint("dyn-svc", "dynhost", "9090");
        assertFalse(activeDh.health().containsKey("dyn-svc:dynhost:9090"));
    }

    @Test
    void testUpdateEndpoint() throws Exception {
        activeDh = fastDepHealth();
        activeDh.start();

        // Add initial endpoint
        CountDownLatch addLatch = new CountDownLatch(1);
        Endpoint oldEp = new Endpoint("oldhost", "8080");
        activeDh.addEndpoint("dyn-svc", DependencyType.HTTP, true, oldEp, mockChecker(addLatch));

        assertTrue(addLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);
        assertTrue(activeDh.health().containsKey("dyn-svc:oldhost:8080"));

        // Update to new endpoint
        CountDownLatch updateLatch = new CountDownLatch(1);
        Endpoint newEp = new Endpoint("newhost", "9090");
        activeDh.updateEndpoint("dyn-svc", "oldhost", "8080", newEp, mockChecker(updateLatch));

        assertTrue(updateLatch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);

        Map<String, Boolean> health = activeDh.health();
        assertFalse(health.containsKey("dyn-svc:oldhost:8080"));
        assertTrue(health.containsKey("dyn-svc:newhost:9090"));
        assertTrue(health.get("dyn-svc:newhost:9090"));
    }

    @Test
    void testUpdateEndpoint_MissingNewHost() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint newEp = new Endpoint("", "9090");
        assertThrows(ValidationException.class, () ->
                activeDh.updateEndpoint("dyn-svc", "oldhost", "8080", newEp, noopChecker()));
    }

    @Test
    void testUpdateEndpoint_NotFound() {
        activeDh = fastDepHealth();
        activeDh.start();

        Endpoint newEp = new Endpoint("newhost", "9090");
        assertThrows(EndpointNotFoundException.class, () ->
                activeDh.updateEndpoint("dyn-svc", "nohost", "1234", newEp, noopChecker()));
    }

    @Test
    void testAddEndpoint_InheritsGlobalConfig() throws Exception {
        // Build DepHealth with a custom global interval of 2s
        activeDh = DepHealth.builder("test-app", "test-group", registry)
                .checkInterval(Duration.ofSeconds(2))
                .build();
        activeDh.start();

        // Add a dynamic endpoint; it should use the global 2s interval.
        // We verify by checking that the checker is called within ~2s (not 15s default).
        CountDownLatch latch = new CountDownLatch(2);
        Endpoint ep = new Endpoint("dynhost", "9090");
        activeDh.addEndpoint("dyn-svc", DependencyType.HTTP, true, ep, mockChecker(latch));

        // Two invocations within 5s proves interval is ~2s, not 15s default
        assertTrue(latch.await(5, TimeUnit.SECONDS),
                "dynamic endpoint should use global interval (2s), not default (15s)");
    }
}
