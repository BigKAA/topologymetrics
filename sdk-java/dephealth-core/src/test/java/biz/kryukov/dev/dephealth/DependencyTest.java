package biz.kryukov.dev.dephealth;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class DependencyTest {

    @Test
    void validDependency() {
        Endpoint ep = new Endpoint("localhost", "5432");
        Dependency dep = Dependency.builder("postgres-main", DependencyType.POSTGRES)
                .endpoint(ep)
                .critical(true)
                .build();

        assertEquals("postgres-main", dep.name());
        assertEquals(DependencyType.POSTGRES, dep.type());
        assertTrue(dep.critical());
        assertEquals(1, dep.endpoints().size());
        assertEquals(ep, dep.endpoints().get(0));
        assertNotNull(dep.config());
    }

    @Test
    void criticalFalse() {
        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "80"))
                .critical(false)
                .build();

        assertFalse(dep.critical());
    }

    @Test
    void missingCriticalThrows() {
        assertThrows(ValidationException.class, () ->
                Dependency.builder("test", DependencyType.HTTP)
                        .endpoint(new Endpoint("localhost", "80"))
                        .build());
    }

    @Test
    void multipleEndpoints() {
        List<Endpoint> eps = List.of(
                new Endpoint("host1", "9092"),
                new Endpoint("host2", "9092")
        );
        Dependency dep = Dependency.builder("kafka-main", DependencyType.KAFKA)
                .endpoints(eps)
                .critical(true)
                .build();

        assertEquals(2, dep.endpoints().size());
    }

    @ParameterizedTest
    @ValueSource(strings = {"a", "my-service", "redis-cache-01", "a1b2c3"})
    void validNames(String name) {
        assertDoesNotThrow(() ->
                Dependency.builder(name, DependencyType.HTTP)
                        .endpoint(new Endpoint("localhost", "80"))
                        .critical(true)
                        .build());
    }

    @ParameterizedTest
    @ValueSource(strings = {"", "A", "1start", "-start", "has_underscore", "has.dot", "HAS-UPPER"})
    void invalidNames(String name) {
        assertThrows(ValidationException.class, () ->
                Dependency.builder(name, DependencyType.HTTP)
                        .endpoint(new Endpoint("localhost", "80"))
                        .critical(true)
                        .build());
    }

    @Test
    void nameTooLong() {
        String longName = "a" + "b".repeat(63);
        assertThrows(ValidationException.class, () ->
                Dependency.builder(longName, DependencyType.HTTP)
                        .endpoint(new Endpoint("localhost", "80"))
                        .critical(true)
                        .build());
    }

    @Test
    void nameMaxLength() {
        String name63 = "a" + "b".repeat(62);
        assertDoesNotThrow(() ->
                Dependency.builder(name63, DependencyType.HTTP)
                        .endpoint(new Endpoint("localhost", "80"))
                        .critical(true)
                        .build());
    }

    @Test
    void noEndpointsThrows() {
        assertThrows(ValidationException.class, () ->
                Dependency.builder("test", DependencyType.HTTP)
                        .critical(true)
                        .build());
    }

    @Test
    void endpointsAreImmutable() {
        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "80"))
                .critical(true)
                .build();

        assertThrows(UnsupportedOperationException.class, () ->
                dep.endpoints().add(new Endpoint("other", "8080")));
    }

    @Test
    void customConfig() {
        CheckConfig cfg = CheckConfig.builder()
                .interval(java.time.Duration.ofSeconds(30))
                .timeout(java.time.Duration.ofSeconds(10))
                .build();

        Dependency dep = Dependency.builder("test", DependencyType.HTTP)
                .endpoint(new Endpoint("localhost", "80"))
                .critical(true)
                .config(cfg)
                .build();

        assertEquals(cfg, dep.config());
    }

    @Test
    void boolToYesNo() {
        assertEquals("yes", Dependency.boolToYesNo(true));
        assertEquals("no", Dependency.boolToYesNo(false));
    }

    @Test
    void endpointLabelsValidated() {
        // reserved label in endpoint -> validation error
        Endpoint ep = new Endpoint("localhost", "80", Map.of("host", "bad"));
        assertThrows(ValidationException.class, () ->
                Dependency.builder("test", DependencyType.HTTP)
                        .endpoint(ep)
                        .critical(true)
                        .build());
    }

    @Test
    void endpointCustomLabelsValid() {
        Endpoint ep = new Endpoint("localhost", "80", Map.of("region", "us-east"));
        assertDoesNotThrow(() ->
                Dependency.builder("test", DependencyType.HTTP)
                        .endpoint(ep)
                        .critical(true)
                        .build());
    }
}
