package biz.kryukov.dev.dephealth.conformance;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
public class InfoController {

    @GetMapping("/")
    public Map<String, String> info() {
        return Map.of(
                "service", "dephealth-conformance-java",
                "version", "0.1.0",
                "language", "java"
        );
    }

    @GetMapping("/health")
    public String health() {
        return "OK";
    }
}
