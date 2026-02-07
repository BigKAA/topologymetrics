package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.net.InetSocketAddress;
import java.time.Duration;

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
        // Не запускаем сервер
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
}
