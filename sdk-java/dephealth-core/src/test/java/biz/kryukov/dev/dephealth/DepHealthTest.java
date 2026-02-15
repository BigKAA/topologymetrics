package biz.kryukov.dev.dephealth;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

class DepHealthTest {

    private SimpleMeterRegistry registry;

    @BeforeEach
    void setUp() {
        registry = new SimpleMeterRegistry();
    }

    @Test
    void builderWithNameAndUrl() {
        DepHealth dh = DepHealth.builder("test-app", registry)
                .dependency("test-http", DependencyType.HTTP, d -> d
                        .url("http://localhost:8080")
                        .critical(true))
                .build();
        assertNotNull(dh);
    }

    @Test
    void builderWithParams() {
        DepHealth dh = DepHealth.builder("test-app", registry)
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
                DepHealth.builder("", registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void nullNameThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder(null, registry)
                        .dependency("test", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true))
                        .build());
    }

    @Test
    void invalidNameThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("INVALID_NAME", registry));
    }

    @Test
    void noDependenciesAllowed() {
        DepHealth dh = DepHealth.builder("test-app", registry).build();
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
                DepHealth.builder("test-app", registry)
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

        DepHealth dh = DepHealth.builder("test-app", registry)
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

        DepHealth dh = DepHealth.builder("test-app", registry)
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
        DepHealth dh = DepHealth.builder("test-app", registry)
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
        DepHealth dh = DepHealth.builder("test-app", registry)
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
        DepHealth dh = DepHealth.builder("test-app", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .jdbcUrl("jdbc:postgresql://localhost:5432/db")
                        .critical(true))
                .build();
        assertNotNull(dh);
    }

    @Test
    void noEndpointConfigThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder("test-app", registry)
                        .dependency("test", DependencyType.HTTP, d -> d.critical(true))
                        .build());
    }

    @Test
    void withLabel() {
        DepHealth dh = DepHealth.builder("test-app", registry)
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
                DepHealth.builder("test-app", registry)
                        .dependency("test-http", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true)
                                .label("host", "bad")));
    }

    @Test
    void invalidLabelNameThrows() {
        assertThrows(ValidationException.class, () ->
                DepHealth.builder("test-app", registry)
                        .dependency("test-http", DependencyType.HTTP, d -> d
                                .url("http://localhost:8080")
                                .critical(true)
                                .label("123invalid", "bad")));
    }
}
