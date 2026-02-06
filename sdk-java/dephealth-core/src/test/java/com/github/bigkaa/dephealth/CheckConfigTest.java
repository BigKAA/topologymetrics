package com.github.bigkaa.dephealth;

import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

class CheckConfigTest {

    @Test
    void defaultValues() {
        CheckConfig cfg = CheckConfig.defaults();
        assertEquals(Duration.ofSeconds(15), cfg.interval());
        assertEquals(Duration.ofSeconds(5), cfg.timeout());
        assertEquals(Duration.ofSeconds(5), cfg.initialDelay());
        assertEquals(1, cfg.failureThreshold());
        assertEquals(1, cfg.successThreshold());
    }

    @Test
    void customValues() {
        CheckConfig cfg = CheckConfig.builder()
                .interval(Duration.ofSeconds(30))
                .timeout(Duration.ofSeconds(10))
                .initialDelay(Duration.ofSeconds(2))
                .failureThreshold(3)
                .successThreshold(2)
                .build();

        assertEquals(Duration.ofSeconds(30), cfg.interval());
        assertEquals(Duration.ofSeconds(10), cfg.timeout());
        assertEquals(Duration.ofSeconds(2), cfg.initialDelay());
        assertEquals(3, cfg.failureThreshold());
        assertEquals(2, cfg.successThreshold());
    }

    @Test
    void intervalTooSmall() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().interval(Duration.ofMillis(500)).build());
    }

    @Test
    void intervalTooLarge() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().interval(Duration.ofMinutes(11)).build());
    }

    @Test
    void timeoutTooSmall() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().timeout(Duration.ofMillis(50)).build());
    }

    @Test
    void timeoutTooLarge() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().timeout(Duration.ofSeconds(31)).build());
    }

    @Test
    void timeoutMustBeLessThanInterval() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder()
                        .interval(Duration.ofSeconds(5))
                        .timeout(Duration.ofSeconds(5))
                        .build());
    }

    @Test
    void thresholdTooSmall() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().failureThreshold(0).build());
    }

    @Test
    void thresholdTooLarge() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().successThreshold(11).build());
    }

    @Test
    void initialDelayZeroAllowed() {
        CheckConfig cfg = CheckConfig.builder().initialDelay(Duration.ZERO).build();
        assertEquals(Duration.ZERO, cfg.initialDelay());
    }

    @Test
    void initialDelayTooLarge() {
        assertThrows(ValidationException.class, () ->
                CheckConfig.builder().initialDelay(Duration.ofMinutes(6)).build());
    }
}
