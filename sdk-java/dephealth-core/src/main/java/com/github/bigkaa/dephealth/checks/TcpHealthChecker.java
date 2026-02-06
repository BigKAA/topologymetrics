package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
import com.github.bigkaa.dephealth.HealthChecker;

import java.net.InetSocketAddress;
import java.net.Socket;
import java.time.Duration;

/**
 * TCP health checker — устанавливает TCP-соединение к эндпоинту.
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
