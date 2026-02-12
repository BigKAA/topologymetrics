package biz.kryukov.dev.dephealth.spring;

import biz.kryukov.dev.dephealth.DepHealth;
import org.springframework.boot.actuate.endpoint.annotation.Endpoint;
import org.springframework.boot.actuate.endpoint.annotation.ReadOperation;

import java.util.Map;

/**
 * Actuator endpoint /actuator/dependencies — JSON with dependency health statuses.
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
