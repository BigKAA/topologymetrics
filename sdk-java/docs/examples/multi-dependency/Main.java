// Example: multiple dependencies with pool integration, custom labels,
// a Kubernetes readiness probe, and a JSON health details endpoint.
package com.example.demo;

import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.EndpointStatus;
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

import com.sun.net.httpserver.HttpServer;
import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import io.micrometer.prometheusmetrics.PrometheusConfig;
import io.micrometer.prometheusmetrics.PrometheusMeterRegistry;
import redis.clients.jedis.JedisPool;
import redis.clients.jedis.JedisPoolConfig;

import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.time.Duration;
import java.util.Map;
import java.util.StringJoiner;

public class Main {

    public static void main(String[] args) throws Exception {
        // Create connection pools.
        var hikariConfig = new HikariConfig();
        hikariConfig.setJdbcUrl("jdbc:postgresql://pg.db:5432/orders");
        hikariConfig.setUsername("app");
        hikariConfig.setPassword("secret");
        var dataSource = new HikariDataSource(hikariConfig);

        var jedisPool = new JedisPool(new JedisPoolConfig(), "redis.cache", 6379,
                2000, "redis-pass");

        // Create a Prometheus registry.
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        // Build dephealth with multiple dependencies and global options.
        var dh = DepHealth.builder("order-service", "backend", registry)
                .checkInterval(Duration.ofSeconds(10))
                .timeout(Duration.ofSeconds(3))

                // PostgreSQL — pool integration via pre-built checker.
                .dependency("postgres-main", DependencyType.POSTGRES,
                        PostgresHealthChecker.builder().dataSource(dataSource).build(),
                        d -> d
                                .host("pg.db").port("5432")
                                .critical(true)
                                .label("env", "production"))

                // Redis — pool integration via pre-built checker.
                .dependency("redis-cache", DependencyType.REDIS,
                        RedisHealthChecker.builder().jedisPool(jedisPool).build(),
                        d -> d
                                .host("redis.cache").port("6379")
                                .critical(false)
                                .label("env", "production"))

                // HTTP with Bearer auth.
                .dependency("auth-service", DependencyType.HTTP, d -> d
                        .url("https://auth.internal:8443")
                        .httpHealthPath("/healthz")
                        .httpBearerToken("my-service-token")
                        .critical(true)
                        .label("env", "production"))

                // gRPC dependency.
                .dependency("recommendation-grpc", DependencyType.GRPC, d -> d
                        .host("recommend.internal").port("9090")
                        .grpcServiceName("recommendation.v1.Recommender")
                        .critical(false)
                        .label("env", "production"))

                // Kafka brokers.
                .dependency("events-kafka", DependencyType.KAFKA, d -> d
                        .url("kafka://kafka-0.broker:9092,kafka-1.broker:9092,kafka-2.broker:9092")
                        .critical(true)
                        .label("env", "production"))

                .build();

        dh.start();

        // Expose HTTP endpoints.
        var server = HttpServer.create(new InetSocketAddress(8080), 0);

        // Prometheus metrics.
        server.createContext("/metrics", exchange -> {
            String body = registry.scrape();
            exchange.getResponseHeaders().set("Content-Type", "text/plain; charset=utf-8");
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        // Kubernetes readiness probe: 200 if all critical deps healthy, 503 otherwise.
        server.createContext("/readyz", exchange -> {
            Map<String, Boolean> health = dh.health();
            boolean ready = health.values().stream().allMatch(ok -> ok != null && ok);
            int code = ready ? 200 : 503;
            String body = ready ? "ok" : "not ready";
            exchange.sendResponseHeaders(code, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        // Debug endpoint: detailed JSON health status.
        server.createContext("/healthz", exchange -> {
            Map<String, EndpointStatus> details = dh.healthDetails();
            var json = new StringJoiner(",", "{", "}");
            details.forEach((key, status) -> json.add(
                    "\"" + key + "\":{\"healthy\":" + status.healthy()
                            + ",\"latency_ms\":" + status.latencyMillis()
                            + ",\"status\":\"" + status.status()
                            + "\",\"detail\":\"" + status.detail() + "\"}"));
            String body = json.toString();
            exchange.getResponseHeaders().set("Content-Type", "application/json");
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });

        server.start();
        System.out.println("Listening on http://localhost:8080");

        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            dh.stop();
            dataSource.close();
            jedisPool.close();
            server.stop(0);
        }));

        Thread.currentThread().join();
    }
}
