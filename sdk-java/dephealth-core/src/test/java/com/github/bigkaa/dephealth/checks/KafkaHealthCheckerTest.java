package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

class KafkaHealthCheckerTest {

    @Test
    void type() {
        assertEquals(DependencyType.KAFKA, new KafkaHealthChecker().type());
    }

    @Test
    void connectionRefused() {
        KafkaHealthChecker checker = new KafkaHealthChecker();
        Endpoint ep = new Endpoint("localhost", "1");
        assertThrows(Exception.class, () -> checker.check(ep, Duration.ofSeconds(2)));
    }
}
