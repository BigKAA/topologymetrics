package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.StatusCategory;
import biz.kryukov.dev.dephealth.UnhealthyException;
import biz.kryukov.dev.dephealth.ValidationException;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.net.InetSocketAddress;
import java.time.Duration;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

class HttpHealthCheckerTest {

    private HttpServer server;
    private int port;

    @BeforeEach
    void setUp() throws Exception {
        server = HttpServer.create(new InetSocketAddress(0), 0);
        port = server.getAddress().getPort();
    }

    @AfterEach
    void tearDown() {
        if (server != null) {
            server.stop(0);
        }
    }

    @Test
    void type() {
        assertEquals(DependencyType.HTTP, HttpHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheck() throws Exception {
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(200, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
    }

    @Test
    void customHealthPath() throws Exception {
        server.createContext("/custom/health", exchange -> {
            exchange.sendResponseHeaders(200, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder()
                .healthPath("/custom/health")
                .build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
    }

    @Test
    void non2xxThrows() throws Exception {
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(503, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        Exception ex = assertThrows(Exception.class,
                () -> checker.check(ep, Duration.ofSeconds(5)));
        assertTrue(ex.getMessage().contains("503"));
    }

    @Test
    void connectionRefused() {
        // Do not start the server
        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertThrows(Exception.class, () -> checker.check(ep, Duration.ofSeconds(1)));
    }

    @Test
    void defaultHealthPath() {
        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        assertEquals("/health", checker.healthPath());
    }

    @Test
    void tlsNotEnabledByDefault() {
        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        assertFalse(checker.tlsEnabled());
    }

    // --- Auth tests ---

    @Test
    void bearerTokenSendsAuthorizationHeader() throws Exception {
        server.createContext("/health", exchange -> {
            String auth = exchange.getRequestHeaders().getFirst("Authorization");
            if ("Bearer test-token".equals(auth)) {
                exchange.sendResponseHeaders(200, -1);
            } else {
                exchange.sendResponseHeaders(401, -1);
            }
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder()
                .bearerToken("test-token")
                .build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
    }

    @Test
    void basicAuthSendsAuthorizationHeader() throws Exception {
        // Base64("admin:password") = "YWRtaW46cGFzc3dvcmQ="
        server.createContext("/health", exchange -> {
            String auth = exchange.getRequestHeaders().getFirst("Authorization");
            if ("Basic YWRtaW46cGFzc3dvcmQ=".equals(auth)) {
                exchange.sendResponseHeaders(200, -1);
            } else {
                exchange.sendResponseHeaders(401, -1);
            }
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder()
                .basicAuth("admin", "password")
                .build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
    }

    @Test
    void customHeadersSent() throws Exception {
        server.createContext("/health", exchange -> {
            String apiKey = exchange.getRequestHeaders().getFirst("X-API-Key");
            if ("my-key".equals(apiKey)) {
                exchange.sendResponseHeaders(200, -1);
            } else {
                exchange.sendResponseHeaders(403, -1);
            }
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder()
                .headers(Map.of("X-API-Key", "my-key"))
                .build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
    }

    @Test
    void http401ThrowsAuthException() throws Exception {
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(401, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        CheckAuthException ex = assertThrows(CheckAuthException.class,
                () -> checker.check(ep, Duration.ofSeconds(5)));
        assertEquals(StatusCategory.AUTH_ERROR, ex.statusCategory());
        assertEquals("auth_error", ex.statusDetail());
    }

    @Test
    void http403ThrowsAuthException() throws Exception {
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(403, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        CheckAuthException ex = assertThrows(CheckAuthException.class,
                () -> checker.check(ep, Duration.ofSeconds(5)));
        assertEquals(StatusCategory.AUTH_ERROR, ex.statusCategory());
        assertEquals("auth_error", ex.statusDetail());
    }

    @Test
    void http500ThrowsUnhealthyNotAuth() throws Exception {
        server.createContext("/health", exchange -> {
            exchange.sendResponseHeaders(500, -1);
            exchange.close();
        });
        server.start();

        HttpHealthChecker checker = HttpHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", String.valueOf(port));
        UnhealthyException ex = assertThrows(UnhealthyException.class,
                () -> checker.check(ep, Duration.ofSeconds(5)));
        assertEquals(StatusCategory.UNHEALTHY, ex.statusCategory());
        assertEquals("http_500", ex.statusDetail());
    }

    @Test
    void conflictBearerAndBasicAuth() {
        assertThrows(ValidationException.class, () ->
                HttpHealthChecker.builder()
                        .bearerToken("token")
                        .basicAuth("user", "pass")
                        .build());
    }

    @Test
    void conflictBearerAndAuthorizationHeader() {
        assertThrows(ValidationException.class, () ->
                HttpHealthChecker.builder()
                        .bearerToken("token")
                        .headers(Map.of("Authorization", "Custom value"))
                        .build());
    }

    @Test
    void conflictBasicAuthAndAuthorizationHeader() {
        assertThrows(ValidationException.class, () ->
                HttpHealthChecker.builder()
                        .basicAuth("user", "pass")
                        .headers(Map.of("Authorization", "Custom value"))
                        .build());
    }

    @Test
    void noConflictSingleBearerToken() {
        assertDoesNotThrow(() ->
                HttpHealthChecker.builder()
                        .bearerToken("token")
                        .build());
    }

    @Test
    void noConflictSingleBasicAuth() {
        assertDoesNotThrow(() ->
                HttpHealthChecker.builder()
                        .basicAuth("user", "pass")
                        .build());
    }

    @Test
    void noConflictHeadersWithoutAuthorization() {
        assertDoesNotThrow(() ->
                HttpHealthChecker.builder()
                        .bearerToken("token")
                        .headers(Map.of("X-Custom", "value"))
                        .build());
    }

    @Test
    void authorizationHeaderCaseInsensitiveConflict() {
        assertThrows(ValidationException.class, () ->
                HttpHealthChecker.builder()
                        .bearerToken("token")
                        .headers(Map.of("authorization", "Custom value"))
                        .build());
    }
}
