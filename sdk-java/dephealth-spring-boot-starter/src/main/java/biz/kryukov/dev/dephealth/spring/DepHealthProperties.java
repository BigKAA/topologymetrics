package biz.kryukov.dev.dephealth.spring;

import org.springframework.boot.context.properties.ConfigurationProperties;

import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Configuration for dephealth via application.yml / application.properties.
 *
 * <pre>
 * dephealth:
 *   name: order-api
 *   interval: 15s
 *   timeout: 5s
 *   dependencies:
 *     postgres-main:
 *       type: postgres
 *       url: postgres://localhost:5432/db
 *       critical: true
 *       labels:
 *         region: us-east-1
 * </pre>
 */
@ConfigurationProperties(prefix = "dephealth")
public class DepHealthProperties {

    private String name;
    private Duration interval;
    private Duration timeout;
    private Map<String, DependencyProperties> dependencies = new LinkedHashMap<>();

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public Duration getInterval() {
        return interval;
    }

    public void setInterval(Duration interval) {
        this.interval = interval;
    }

    public Duration getTimeout() {
        return timeout;
    }

    public void setTimeout(Duration timeout) {
        this.timeout = timeout;
    }

    public Map<String, DependencyProperties> getDependencies() {
        return dependencies;
    }

    public void setDependencies(Map<String, DependencyProperties> dependencies) {
        this.dependencies = dependencies;
    }

    public static class DependencyProperties {
        private String type;
        private String url;
        private String host;
        private String port;
        private Boolean critical;
        private Duration interval;
        private Duration timeout;
        private Map<String, String> labels = new LinkedHashMap<>();

        // HTTP
        private String healthPath;
        private Boolean tls;
        private Boolean tlsSkipVerify;

        // gRPC
        private String serviceName;

        // DB
        private String username;
        private String password;
        private String database;
        private String query;

        // Redis
        private String redisPassword;
        private Integer redisDb;

        // AMQP
        private String amqpUrl;
        private String amqpUsername;
        private String amqpPassword;
        private String virtualHost;

        public String getType() {
            return type;
        }

        public void setType(String type) {
            this.type = type;
        }

        public String getUrl() {
            return url;
        }

        public void setUrl(String url) {
            this.url = url;
        }

        public String getHost() {
            return host;
        }

        public void setHost(String host) {
            this.host = host;
        }

        public String getPort() {
            return port;
        }

        public void setPort(String port) {
            this.port = port;
        }

        public Boolean getCritical() {
            return critical;
        }

        public void setCritical(Boolean critical) {
            this.critical = critical;
        }

        public Map<String, String> getLabels() {
            return labels;
        }

        public void setLabels(Map<String, String> labels) {
            this.labels = labels;
        }

        public Duration getInterval() {
            return interval;
        }

        public void setInterval(Duration interval) {
            this.interval = interval;
        }

        public Duration getTimeout() {
            return timeout;
        }

        public void setTimeout(Duration timeout) {
            this.timeout = timeout;
        }

        public String getHealthPath() {
            return healthPath;
        }

        public void setHealthPath(String healthPath) {
            this.healthPath = healthPath;
        }

        public Boolean getTls() {
            return tls;
        }

        public void setTls(Boolean tls) {
            this.tls = tls;
        }

        public Boolean getTlsSkipVerify() {
            return tlsSkipVerify;
        }

        public void setTlsSkipVerify(Boolean tlsSkipVerify) {
            this.tlsSkipVerify = tlsSkipVerify;
        }

        public String getServiceName() {
            return serviceName;
        }

        public void setServiceName(String serviceName) {
            this.serviceName = serviceName;
        }

        public String getUsername() {
            return username;
        }

        public void setUsername(String username) {
            this.username = username;
        }

        public String getPassword() {
            return password;
        }

        public void setPassword(String password) {
            this.password = password;
        }

        public String getDatabase() {
            return database;
        }

        public void setDatabase(String database) {
            this.database = database;
        }

        public String getQuery() {
            return query;
        }

        public void setQuery(String query) {
            this.query = query;
        }

        public String getRedisPassword() {
            return redisPassword;
        }

        public void setRedisPassword(String redisPassword) {
            this.redisPassword = redisPassword;
        }

        public Integer getRedisDb() {
            return redisDb;
        }

        public void setRedisDb(Integer redisDb) {
            this.redisDb = redisDb;
        }

        public String getAmqpUrl() {
            return amqpUrl;
        }

        public void setAmqpUrl(String amqpUrl) {
            this.amqpUrl = amqpUrl;
        }

        public String getAmqpUsername() {
            return amqpUsername;
        }

        public void setAmqpUsername(String amqpUsername) {
            this.amqpUsername = amqpUsername;
        }

        public String getAmqpPassword() {
            return amqpPassword;
        }

        public void setAmqpPassword(String amqpPassword) {
            this.amqpPassword = amqpPassword;
        }

        public String getVirtualHost() {
            return virtualHost;
        }

        public void setVirtualHost(String virtualHost) {
            this.virtualHost = virtualHost;
        }
    }
}
