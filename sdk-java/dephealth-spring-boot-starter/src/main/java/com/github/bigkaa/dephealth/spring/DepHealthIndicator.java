package com.github.bigkaa.dephealth.spring;

import com.github.bigkaa.dephealth.DepHealth;
import org.springframework.boot.actuate.health.Health;
import org.springframework.boot.actuate.health.HealthIndicator;

import java.util.Map;

/**
 * Spring Boot Actuator HealthIndicator: отображает состояние зависимостей в /actuator/health.
 */
public class DepHealthIndicator implements HealthIndicator {

    private final DepHealth depHealth;

    public DepHealthIndicator(DepHealth depHealth) {
        this.depHealth = depHealth;
    }

    @Override
    public Health health() {
        Map<String, Boolean> states = depHealth.health();

        boolean allHealthy = states.values().stream().allMatch(Boolean::booleanValue);

        Health.Builder builder = allHealthy ? Health.up() : Health.down();

        states.forEach((key, healthy) ->
                builder.withDetail(key, healthy ? "UP" : "DOWN"));

        return builder.build();
    }
}
