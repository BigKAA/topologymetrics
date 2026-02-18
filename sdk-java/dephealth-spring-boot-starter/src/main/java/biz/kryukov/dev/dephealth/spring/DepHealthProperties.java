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
 *   group: billing-team
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
    private String group;
    private Duration interval;
    private Duration timeout;
    private Map<String, DependencyProperties> dependencies = new LinkedHashMap<>();

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getGroup() {
        return group;
    }

    public void setGroup(String group) {
        this.group = group;
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

        // HTTP auth
        private Map<String, String> httpHeaders;
        private String httpBearerToken;
        private String httpBasicUsername;
        private String httpBasicPassword;

        // gRPC
        private String serviceName;

        // gRPC auth
        private Map<String, String> grpcMetadata;
        private String grpcBearerToken;
        private String grpcBasicUsername;
        private String grpcBasicPassword;

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

        public Map<String, String> getHttpHeaders() {
            return httpHeaders;
        }

        public void setHttpHeaders(Map<String, String> httpHeaders) {
            this.httpHeaders = httpHeaders;
        }

        public String getHttpBearerToken() {
            return httpBearerToken;
        }

        public void setHttpBearerToken(String httpBearerToken) {
            this.httpBearerToken = httpBearerToken;
        }

        public String getHttpBasicUsername() {
            return httpBasicUsername;
        }

        public void setHttpBasicUsername(String httpBasicUsername) {
            this.httpBasicUsername = httpBasicUsername;
        }

        public String getHttpBasicPassword() {
            return httpBasicPassword;
        }

        public void setHttpBasicPassword(String httpBasicPassword) {
            this.httpBasicPassword = httpBasicPassword;
        }

        public String getServiceName() {
            return serviceName;
        }

        public void setServiceName(String serviceName) {
            this.serviceName = serviceName;
        }

        public Map<String, String> getGrpcMetadata() {
            return grpcMetadata;
        }

        public void setGrpcMetadata(Map<String, String> grpcMetadata) {
            this.grpcMetadata = grpcMetadata;
        }

        public String getGrpcBearerToken() {
            return grpcBearerToken;
        }

        public void setGrpcBearerToken(String grpcBearerToken) {
            this.grpcBearerToken = grpcBearerToken;
        }

        public String getGrpcBasicUsername() {
            return grpcBasicUsername;
        }

        public void setGrpcBasicUsername(String grpcBasicUsername) {
            this.grpcBasicUsername = grpcBasicUsername;
        }

        public String getGrpcBasicPassword() {
            return grpcBasicPassword;
        }

        public void setGrpcBasicPassword(String grpcBasicPassword) {
            this.grpcBasicPassword = grpcBasicPassword;
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
