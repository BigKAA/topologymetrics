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
    void builderWithUrl() {
        DepHealth dh = DepHealth.builder(registry)
                .dependency("test-http", DependencyType.HTTP, d -> d
                        .url("http://localhost:8080"))
                .build();
        assertNotNull(dh);
    }

    @Test
    void builderWithParams() {
        DepHealth dh = DepHealth.builder(registry)
                .dependency("test-tcp", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("8080"))
                .build();
        assertNotNull(dh);
    }

    @Test
    void noDependenciesThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder(registry).build());
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

        DepHealth dh = DepHealth.builder(registry)
                .checkInterval(Duration.ofSeconds(1))
                .dependency("test", DependencyType.HTTP, mockChecker, d -> d
                        .host("localhost")
                        .port("8080"))
                .build();

        dh.start();
        assertTrue(latch.await(3, TimeUnit.SECONDS));
        Thread.sleep(200);

        Map<String, Boolean> health = dh.health();
        assertFalse(health.isEmpty());

        dh.stop();
    }

    @Test
    void globalIntervalUsed() {
        DepHealth dh = DepHealth.builder(registry)
                .checkInterval(Duration.ofSeconds(30))
                .dependency("test", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("80"))
                .build();
        assertNotNull(dh);
    }

    @Test
    void perDependencyIntervalOverridesGlobal() {
        DepHealth dh = DepHealth.builder(registry)
                .checkInterval(Duration.ofSeconds(30))
                .dependency("test", DependencyType.TCP, d -> d
                        .host("localhost")
                        .port("80")
                        .interval(Duration.ofSeconds(10)))
                .build();
        assertNotNull(dh);
    }

    @Test
    void jdbcUrlParsing() {
        DepHealth dh = DepHealth.builder(registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .jdbcUrl("jdbc:postgresql://localhost:5432/db"))
                .build();
        assertNotNull(dh);
    }

    @Test
    void noEndpointConfigThrows() {
        assertThrows(ConfigurationException.class, () ->
                DepHealth.builder(registry)
                        .dependency("test", DependencyType.HTTP, d -> {})
                        .build());
    }
}
