package biz.kryukov.dev.dephealth.conformance;

import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.EndpointStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.LinkedHashMap;
import java.util.Map;

@RestController
public class HealthDetailsController {

    private final DepHealth depHealth;

    public HealthDetailsController(DepHealth depHealth) {
        this.depHealth = depHealth;
    }

    @GetMapping("/health-details")
    public Map<String, Map<String, Object>> healthDetails() {
        Map<String, EndpointStatus> details = depHealth.healthDetails();
        Map<String, Map<String, Object>> result = new LinkedHashMap<>();
        for (var entry : details.entrySet()) {
            result.put(entry.getKey(), toMap(entry.getValue()));
        }
        return result;
    }

    private static Map<String, Object> toMap(EndpointStatus es) {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("healthy", es.healthy());
        m.put("status", es.status());
        m.put("detail", es.detail());
        m.put("latency_ms", es.latencyMillis());
        m.put("type", es.type().label());
        m.put("name", es.name());
        m.put("host", es.host());
        m.put("port", es.port());
        m.put("critical", es.critical());
        m.put("last_checked_at", es.lastCheckedAt() != null
                ? es.lastCheckedAt().toString() : null);
        m.put("labels", es.labels());
        return m;
    }
}
