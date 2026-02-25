package biz.kryukov.dev.dephealth.spring;

import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker;

import io.micrometer.core.instrument.MeterRegistry;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;

/**
 * Auto-configuration for dephealth: creates a DepHealth bean from application.yml properties.
 */
@AutoConfiguration
@ConditionalOnClass(DepHealth.class)
@EnableConfigurationProperties(DepHealthProperties.class)
public class DepHealthAutoConfiguration {

    /**
     * Creates a {@link DepHealth} bean configured from application properties.
     *
     * @param properties    dephealth configuration properties
     * @param meterRegistry Micrometer meter registry
     * @return configured DepHealth instance
     */
    @Bean
    @ConditionalOnMissingBean
    public DepHealth depHealth(DepHealthProperties properties, MeterRegistry meterRegistry) {
        DepHealth.Builder builder = DepHealth.builder(
                properties.getName(), properties.getGroup(), meterRegistry);

        if (properties.getInterval() != null) {
            builder.checkInterval(properties.getInterval());
        }
        if (properties.getTimeout() != null) {
            builder.timeout(properties.getTimeout());
        }

        properties.getDependencies().forEach((name, depProps) -> {
            DependencyType type = DependencyType.fromLabel(depProps.getType());
            builder.dependency(name, type, d -> configureDependency(d, depProps));
        });

        return builder.build();
    }

    /** Creates a lifecycle bean for automatic start/stop of health checks. */
    @Bean
    @ConditionalOnMissingBean
    public DepHealthLifecycle depHealthLifecycle(DepHealth depHealth) {
        return new DepHealthLifecycle(depHealth);
    }

    /** Creates a Spring Boot Actuator HealthIndicator for dependency health. */
    @Bean
    @ConditionalOnMissingBean
    public DepHealthIndicator depHealthIndicator(DepHealth depHealth) {
        return new DepHealthIndicator(depHealth);
    }

    /** Creates an Actuator endpoint exposing dependency health at {@code /actuator/dependencies}. */
    @Bean
    @ConditionalOnMissingBean
    public DependenciesEndpoint dependenciesEndpoint(DepHealth depHealth) {
        return new DependenciesEndpoint(depHealth);
    }

    private void configureDependency(DepHealth.DependencyBuilder d,
                                     DepHealthProperties.DependencyProperties props) {
        // Connection
        if (props.getUrl() != null) {
            d.url(props.getUrl());
        }
        if (props.getHost() != null) {
            d.host(props.getHost());
        }
        if (props.getPort() != null) {
            d.port(props.getPort());
        }

        // General
        if (props.getCritical() != null) {
            d.critical(props.getCritical());
        }

        // Custom labels
        if (props.getLabels() != null) {
            props.getLabels().forEach(d::label);
        }
        if (props.getInterval() != null) {
            d.interval(props.getInterval());
        }
        if (props.getTimeout() != null) {
            d.timeout(props.getTimeout());
        }

        // HTTP
        if (props.getHealthPath() != null) {
            d.httpHealthPath(props.getHealthPath());
        }
        if (props.getTls() != null) {
            d.httpTls(props.getTls());
        }
        if (props.getTlsSkipVerify() != null) {
            d.httpTlsSkipVerify(props.getTlsSkipVerify());
        }

        // HTTP auth
        if (props.getHttpHeaders() != null) {
            d.httpHeaders(props.getHttpHeaders());
        }
        if (props.getHttpBearerToken() != null) {
            d.httpBearerToken(props.getHttpBearerToken());
        }
        if (props.getHttpBasicUsername() != null) {
            d.httpBasicAuth(props.getHttpBasicUsername(), props.getHttpBasicPassword());
        }

        // gRPC
        if (props.getServiceName() != null) {
            d.grpcServiceName(props.getServiceName());
        }
        if (props.getTls() != null) {
            d.grpcTls(props.getTls());
        }

        // gRPC auth
        if (props.getGrpcMetadata() != null) {
            d.grpcMetadata(props.getGrpcMetadata());
        }
        if (props.getGrpcBearerToken() != null) {
            d.grpcBearerToken(props.getGrpcBearerToken());
        }
        if (props.getGrpcBasicUsername() != null) {
            d.grpcBasicAuth(props.getGrpcBasicUsername(), props.getGrpcBasicPassword());
        }

        // DB
        if (props.getUsername() != null) {
            d.dbUsername(props.getUsername());
        }
        if (props.getPassword() != null) {
            d.dbPassword(props.getPassword());
        }
        if (props.getDatabase() != null) {
            d.dbDatabase(props.getDatabase());
        }
        if (props.getQuery() != null) {
            d.dbQuery(props.getQuery());
        }

        // Redis
        if (props.getRedisPassword() != null) {
            d.redisPassword(props.getRedisPassword());
        }
        if (props.getRedisDb() != null) {
            d.redisDb(props.getRedisDb());
        }

        // AMQP
        if (props.getAmqpUrl() != null) {
            d.amqpUrl(props.getAmqpUrl());
        }
        if (props.getAmqpUsername() != null) {
            d.amqpUsername(props.getAmqpUsername());
        }
        if (props.getAmqpPassword() != null) {
            d.amqpPassword(props.getAmqpPassword());
        }
        if (props.getVirtualHost() != null) {
            d.amqpVirtualHost(props.getVirtualHost());
        }

        // LDAP
        if (props.getLdapCheckMethod() != null) {
            d.ldapCheckMethod(LdapHealthChecker.CheckMethod.valueOf(
                    props.getLdapCheckMethod().toUpperCase()));
        }
        if (props.getLdapBindDn() != null) {
            d.ldapBindDN(props.getLdapBindDn());
        }
        if (props.getLdapBindPassword() != null) {
            d.ldapBindPassword(props.getLdapBindPassword());
        }
        if (props.getLdapBaseDn() != null) {
            d.ldapBaseDN(props.getLdapBaseDn());
        }
        if (props.getLdapSearchFilter() != null) {
            d.ldapSearchFilter(props.getLdapSearchFilter());
        }
        if (props.getLdapSearchScope() != null) {
            d.ldapSearchScope(LdapHealthChecker.LdapSearchScope.valueOf(
                    props.getLdapSearchScope().toUpperCase()));
        }
        if (props.getLdapStartTls() != null) {
            d.ldapStartTLS(props.getLdapStartTls());
        }
        if (props.getLdapTlsSkipVerify() != null) {
            d.ldapTlsSkipVerify(props.getLdapTlsSkipVerify());
        }
    }
}
