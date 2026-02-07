package biz.kryukov.dev.dephealth.spring;

import biz.kryukov.dev.dephealth.DepHealth;
import org.junit.jupiter.api.Test;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.Status;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;

class DepHealthIndicatorTest {

    @Test
    void allHealthyReturnsUp() {
        DepHealth depHealth = mock(DepHealth.class);
        when(depHealth.health()).thenReturn(Map.of(
                "postgres:localhost:5432", true,
                "redis:localhost:6379", true
        ));

        DepHealthIndicator indicator = new DepHealthIndicator(depHealth);
        Health health = indicator.health();

        assertEquals(Status.UP, health.getStatus());
        assertEquals("UP", health.getDetails().get("postgres:localhost:5432"));
        assertEquals("UP", health.getDetails().get("redis:localhost:6379"));
    }

    @Test
    void anyUnhealthyReturnsDown() {
        DepHealth depHealth = mock(DepHealth.class);
        when(depHealth.health()).thenReturn(Map.of(
                "postgres:localhost:5432", true,
                "redis:localhost:6379", false
        ));

        DepHealthIndicator indicator = new DepHealthIndicator(depHealth);
        Health health = indicator.health();

        assertEquals(Status.DOWN, health.getStatus());
        assertEquals("DOWN", health.getDetails().get("redis:localhost:6379"));
    }

    @Test
    void emptyHealthReturnsUp() {
        DepHealth depHealth = mock(DepHealth.class);
        when(depHealth.health()).thenReturn(Map.of());

        DepHealthIndicator indicator = new DepHealthIndicator(depHealth);
        Health health = indicator.health();

        assertEquals(Status.UP, health.getStatus());
    }
}
