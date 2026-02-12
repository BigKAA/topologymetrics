package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;

import java.net.InetSocketAddress;
import java.net.Socket;
import java.time.Duration;

/**
 * TCP health checker — establishes a TCP connection to the endpoint.
 */
public final class TcpHealthChecker implements HealthChecker {

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        int timeoutMs = (int) timeout.toMillis();
        try (Socket socket = new Socket()) {
            socket.connect(
                    new InetSocketAddress(endpoint.host(), endpoint.portAsInt()),
                    timeoutMs);
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.TCP;
    }
}
