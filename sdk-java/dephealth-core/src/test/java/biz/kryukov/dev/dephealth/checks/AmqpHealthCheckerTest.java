package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

class AmqpHealthCheckerTest {

    @Test
    void type() {
        assertEquals(DependencyType.AMQP, AmqpHealthChecker.builder().build().type());
    }

    @Test
    void connectionRefused() {
        AmqpHealthChecker checker = AmqpHealthChecker.builder()
                .username("guest")
                .password("guest")
                .build();

        Endpoint ep = new Endpoint("localhost", "1");
        assertThrows(Exception.class, () -> checker.check(ep, Duration.ofSeconds(1)));
    }

    @Test
    void builderSetsFields() {
        // Just verify that the builder does not throw exceptions
        AmqpHealthChecker checker = AmqpHealthChecker.builder()
                .username("user")
                .password("pass")
                .virtualHost("/test")
                .build();
        assertNotNull(checker);
    }

    @Test
    void builderWithUrl() {
        AmqpHealthChecker checker = AmqpHealthChecker.builder()
                .amqpUrl("amqp://user:pass@localhost:5672/vhost")
                .build();
        assertNotNull(checker);
    }
}
