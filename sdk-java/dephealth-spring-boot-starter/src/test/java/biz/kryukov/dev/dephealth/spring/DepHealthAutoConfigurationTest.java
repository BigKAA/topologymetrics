package biz.kryukov.dev.dephealth.spring;

import biz.kryukov.dev.dephealth.DepHealth;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;
import org.junit.jupiter.api.Test;
import org.springframework.boot.autoconfigure.AutoConfigurations;
import org.springframework.boot.test.context.runner.ApplicationContextRunner;

import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

class DepHealthAutoConfigurationTest {

    private final ApplicationContextRunner contextRunner = new ApplicationContextRunner()
            .withConfiguration(AutoConfigurations.of(DepHealthAutoConfiguration.class))
            .withBean(SimpleMeterRegistry.class);

    @Test
    void createsDepHealthBean() {
        contextRunner
                .withPropertyValues(
                        "dephealth.dependencies.test-http.type=http",
                        "dephealth.dependencies.test-http.url=http://localhost:8080"
                )
                .run(context -> {
                    assertTrue(context.containsBean("depHealth"));
                    assertNotNull(context.getBean(DepHealth.class));
                });
    }

    @Test
    void createsLifecycleBean() {
        contextRunner
                .withPropertyValues(
                        "dephealth.dependencies.test-http.type=http",
                        "dephealth.dependencies.test-http.url=http://localhost:8080"
                )
                .run(context -> {
                    assertTrue(context.containsBean("depHealthLifecycle"));
                });
    }

    @Test
    void createsHealthIndicatorBean() {
        contextRunner
                .withPropertyValues(
                        "dephealth.dependencies.test-http.type=http",
                        "dephealth.dependencies.test-http.url=http://localhost:8080"
                )
                .run(context -> {
                    assertTrue(context.containsBean("depHealthIndicator"));
                });
    }

    @Test
    void createsDependenciesEndpointBean() {
        contextRunner
                .withPropertyValues(
                        "dephealth.dependencies.test-http.type=http",
                        "dephealth.dependencies.test-http.url=http://localhost:8080"
                )
                .run(context -> {
                    assertTrue(context.containsBean("dependenciesEndpoint"));
                });
    }

    @Test
    void globalIntervalAndTimeout() {
        contextRunner
                .withPropertyValues(
                        "dephealth.interval=30s",
                        "dephealth.timeout=10s",
                        "dephealth.dependencies.test-tcp.type=tcp",
                        "dephealth.dependencies.test-tcp.host=localhost",
                        "dephealth.dependencies.test-tcp.port=8080"
                )
                .run(context -> {
                    assertNotNull(context.getBean(DepHealth.class));
                });
    }

    @Test
    void perDependencyConfig() {
        contextRunner
                .withPropertyValues(
                        "dephealth.dependencies.my-redis.type=redis",
                        "dephealth.dependencies.my-redis.url=redis://localhost:6379",
                        "dephealth.dependencies.my-redis.critical=true",
                        "dephealth.dependencies.my-redis.interval=10s",
                        "dephealth.dependencies.my-redis.timeout=3s"
                )
                .run(context -> {
                    assertNotNull(context.getBean(DepHealth.class));
                });
    }
}
