// Example: programmatic dephealth API without Spring Boot.
// Configures dependencies via the builder, exposes Prometheus metrics on /metrics.
package com.example.demo;

import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;

import com.sun.net.httpserver.HttpServer;
import io.micrometer.prometheusmetrics.PrometheusConfig;
import io.micrometer.prometheusmetrics.PrometheusMeterRegistry;

import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.time.Duration;

public class Main {

    public static void main(String[] args) throws Exception {
        // Create a Prometheus registry.
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        // Build dephealth with a single HTTP dependency.
        var dh = DepHealth.builder("my-service", "backend", registry)
                .checkInterval(Duration.ofSeconds(15))
                .timeout(Duration.ofSeconds(5))
                .dependency("payment-api", DependencyType.HTTP, d -> d
                        .url("https://payment.internal:8443/health")
                        .critical(true))
                .dependency("redis-cache", DependencyType.REDIS, d -> d
                        .url("redis://redis.cache:6379")
                        .critical(false))
                .build();

        // Start health checks.
        dh.start();

        // Expose Prometheus metrics via JDK HttpServer.
        var server = HttpServer.create(new InetSocketAddress(9090), 0);
        server.createContext("/metrics", exchange -> {
            String body = registry.scrape();
            exchange.getResponseHeaders().set("Content-Type", "text/plain; charset=utf-8");
            exchange.sendResponseHeaders(200, body.length());
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body.getBytes());
            }
        });
        server.start();
        System.out.println("Metrics available at http://localhost:9090/metrics");

        // Wait for shutdown signal.
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            dh.stop();
            server.stop(0);
        }));

        Thread.currentThread().join();
    }
}
