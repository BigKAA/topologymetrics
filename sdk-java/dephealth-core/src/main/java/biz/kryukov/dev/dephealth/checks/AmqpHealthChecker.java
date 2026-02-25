package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;

import com.rabbitmq.client.Connection;
import com.rabbitmq.client.ConnectionFactory;

import java.time.Duration;

/**
 * AMQP health checker — open/close RabbitMQ connection.
 */
public final class AmqpHealthChecker implements HealthChecker {

    private final String username;
    private final String password;
    private final String virtualHost;
    private final String amqpUrl;

    private AmqpHealthChecker(Builder builder) {
        this.username = builder.username;
        this.password = builder.password;
        this.virtualHost = builder.virtualHost;
        this.amqpUrl = builder.amqpUrl;
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        int timeoutMs = (int) timeout.toMillis();

        ConnectionFactory factory = new ConnectionFactory();

        if (amqpUrl != null && !amqpUrl.isEmpty()) {
            factory.setUri(amqpUrl);
        } else {
            factory.setHost(endpoint.host());
            factory.setPort(endpoint.portAsInt());
            if (username != null) {
                factory.setUsername(username);
            }
            if (password != null) {
                factory.setPassword(password);
            }
            if (virtualHost != null) {
                factory.setVirtualHost(virtualHost);
            }
        }

        factory.setConnectionTimeout(timeoutMs);
        factory.setHandshakeTimeout(timeoutMs);

        try (Connection conn = factory.newConnection("dephealth-check")) {
            if (!conn.isOpen()) {
                throw new UnhealthyException("AMQP connection is not open");
            }
        } catch (biz.kryukov.dev.dephealth.CheckException e) {
            throw e;
        } catch (Exception e) {
            throw classifyAmqpError(e);
        }
    }

    private static Exception classifyAmqpError(Exception e) {
        String msg = e.getMessage();
        if (msg != null && (msg.contains("403") || msg.contains("ACCESS_REFUSED"))) {
            return new CheckAuthException("AMQP auth error: " + msg, e);
        }
        return e;
    }

    @Override
    public DependencyType type() {
        return DependencyType.AMQP;
    }

    /** Creates a new builder with default settings. */
    public static Builder builder() {
        return new Builder();
    }

    /** Builder for {@link AmqpHealthChecker}. */
    public static final class Builder {
        private String username;
        private String password;
        private String virtualHost;
        private String amqpUrl;

        private Builder() {}

        /** Sets the AMQP username. */
        public Builder username(String username) {
            this.username = username;
            return this;
        }

        /** Sets the AMQP password. */
        public Builder password(String password) {
            this.password = password;
            return this;
        }

        /** Sets the AMQP virtual host. */
        public Builder virtualHost(String virtualHost) {
            this.virtualHost = virtualHost;
            return this;
        }

        /** Sets the full AMQP connection URL. */
        public Builder amqpUrl(String amqpUrl) {
            this.amqpUrl = amqpUrl;
            return this;
        }

        /** Builds the checker. */
        public AmqpHealthChecker build() {
            return new AmqpHealthChecker(this);
        }
    }
}
