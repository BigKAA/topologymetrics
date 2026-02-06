package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
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
        // Просто проверяем, что builder не бросает исключений
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
