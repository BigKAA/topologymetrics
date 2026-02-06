package com.github.bigkaa.dephealth.spring;

import com.github.bigkaa.dephealth.DepHealth;
import org.springframework.boot.actuate.endpoint.annotation.Endpoint;
import org.springframework.boot.actuate.endpoint.annotation.ReadOperation;

import java.util.Map;

/**
 * Actuator endpoint /actuator/dependencies — JSON со статусами зависимостей.
 */
@Endpoint(id = "dependencies")
public class DependenciesEndpoint {

    private final DepHealth depHealth;

    public DependenciesEndpoint(DepHealth depHealth) {
        this.depHealth = depHealth;
    }

    @ReadOperation
    public Map<String, Boolean> dependencies() {
        return depHealth.health();
    }
}
