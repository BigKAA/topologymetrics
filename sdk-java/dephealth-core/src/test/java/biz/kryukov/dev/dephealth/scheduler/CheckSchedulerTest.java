package biz.kryukov.dev.dephealth.scheduler;

import biz.kryukov.dev.dephealth.CheckConfig;
import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.metrics.MetricsExporter;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.Map;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.*;

class CheckSchedulerTest {

    private MetricsExporter metrics;
    private CheckScheduler scheduler;

    @BeforeEach
    void setUp() {
        metrics = new MetricsExporter(new SimpleMeterRegistry(), "test-app", "test-group");
        scheduler = new CheckScheduler(metrics);
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
}
