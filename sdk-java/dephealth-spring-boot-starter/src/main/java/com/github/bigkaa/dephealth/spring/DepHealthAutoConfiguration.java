package com.github.bigkaa.dephealth.spring;

import com.github.bigkaa.dephealth.DepHealth;
import com.github.bigkaa.dephealth.DependencyType;

import io.micrometer.core.instrument.MeterRegistry;

import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.context.annotation.Bean;

/**
 * Автоконфигурация dephealth: создаёт bean DepHealth из свойств application.yml.
 */
@AutoConfiguration
@ConditionalOnClass(DepHealth.class)
@EnableConfigurationProperties(DepHealthProperties.class)
public class DepHealthAutoConfiguration {

    @Bean
    @ConditionalOnMissingBean
    public DepHealth depHealth(DepHealthProperties properties, MeterRegistry meterRegistry) {
        DepHealth.Builder builder = DepHealth.builder(meterRegistry);

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

    @Bean
    @ConditionalOnMissingBean
    public DepHealthLifecycle depHealthLifecycle(DepHealth depHealth) {
        return new DepHealthLifecycle(depHealth);
    }

    @Bean
    @ConditionalOnMissingBean
    public DepHealthIndicator depHealthIndicator(DepHealth depHealth) {
        return new DepHealthIndicator(depHealth);
    }

    @Bean
    @ConditionalOnMissingBean
    public DependenciesEndpoint dependenciesEndpoint(DepHealth depHealth) {
        return new DependenciesEndpoint(depHealth);
    }

    private void configureDependency(DepHealth.DependencyBuilder d,
                                     DepHealthProperties.DependencyProperties props) {
        // Подключение
        if (props.getUrl() != null) {
            d.url(props.getUrl());
        }
        if (props.getHost() != null) {
            d.host(props.getHost());
        }
        if (props.getPort() != null) {
            d.port(props.getPort());
        }

        // Общее
        d.critical(props.isCritical());
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

        // gRPC
        if (props.getServiceName() != null) {
            d.grpcServiceName(props.getServiceName());
        }
        if (props.getTls() != null) {
            d.grpcTls(props.getTls());
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
    }
}
