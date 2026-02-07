package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;

import org.apache.kafka.clients.admin.AdminClient;
import org.apache.kafka.clients.admin.AdminClientConfig;
import org.apache.kafka.common.Node;

import java.time.Duration;
import java.util.Collection;
import java.util.Properties;
import java.util.concurrent.TimeUnit;

/**
 * Kafka health checker — AdminClient describeCluster().nodes().
 */
public final class KafkaHealthChecker implements HealthChecker {

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        Properties props = new Properties();
        props.put(AdminClientConfig.BOOTSTRAP_SERVERS_CONFIG,
                endpoint.host() + ":" + endpoint.port());
        props.put(AdminClientConfig.REQUEST_TIMEOUT_MS_CONFIG, (int) timeout.toMillis());
        props.put(AdminClientConfig.DEFAULT_API_TIMEOUT_MS_CONFIG, (int) timeout.toMillis());

        try (AdminClient client = AdminClient.create(props)) {
            Collection<Node> nodes = client.describeCluster().nodes()
                    .get(timeout.toMillis(), TimeUnit.MILLISECONDS);
            if (nodes.isEmpty()) {
                throw new Exception("Kafka cluster has no nodes");
            }
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.KAFKA;
    }
}
