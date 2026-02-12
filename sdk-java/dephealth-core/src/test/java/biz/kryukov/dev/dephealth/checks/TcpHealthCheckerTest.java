package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import org.junit.jupiter.api.Test;

import java.net.ServerSocket;
import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

class TcpHealthCheckerTest {

    @Test
    void type() {
        assertEquals(DependencyType.TCP, new TcpHealthChecker().type());
    }

    @Test
    void successfulConnection() throws Exception {
        try (ServerSocket server = new ServerSocket(0)) {
            int port = server.getLocalPort();
            Endpoint ep = new Endpoint("localhost", String.valueOf(port));
            TcpHealthChecker checker = new TcpHealthChecker();
            assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(2)));
        }
    }

    @Test
    void connectionRefused() {
        // Use a port that nothing is listening on
        Endpoint ep = new Endpoint("localhost", "1");
        TcpHealthChecker checker = new TcpHealthChecker();
        assertThrows(Exception.class, () -> checker.check(ep, Duration.ofSeconds(1)));
    }
}
