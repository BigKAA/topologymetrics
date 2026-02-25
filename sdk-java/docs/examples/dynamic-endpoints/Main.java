// Example: dynamic endpoint management via a REST API.
// Endpoints can be added, removed, and updated at runtime.
package com.example.demo;

import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import io.micrometer.prometheusmetrics.PrometheusConfig;
import io.micrometer.prometheusmetrics.PrometheusMeterRegistry;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.util.Map;

public class Main {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    public static void main(String[] args) throws Exception {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        // Start with one static HTTP dependency.
        var dh = DepHealth.builder("gateway", "platform", registry)
                .dependency("users-api", DependencyType.HTTP, d -> d
                        .url("http://users.internal:8080")
                        .critical(true))
                .build();

        dh.start();

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

        // GET /health — current health status.
        server.createContext("/health", exchange -> {
            writeJson(exchange, dh.healthDetails());
        });

        // POST /endpoints — add a new monitored endpoint.
        // Body: {"name":"billing-api","host":"billing.internal","port":"8080","critical":true}
        server.createContext("/endpoints", exchange -> {
            if (!"POST".equals(exchange.getRequestMethod())) {
                sendError(exchange, 405, "method not allowed");
                return;
            }
            var req = MAPPER.readValue(exchange.getRequestBody(), AddRequest.class);
            var ep = new Endpoint(req.host, req.port);
            var checker = HttpHealthChecker.builder().build();
            try {
                dh.addEndpoint(req.name, DependencyType.HTTP, req.critical, ep, checker);
                writeJson(exchange, Map.of("status", "added"));
            } catch (Exception e) {
                sendError(exchange, 400, e.getMessage());
            }
        });

        // DELETE /endpoints/delete?name=billing-api&host=billing.internal&port=8080
        server.createContext("/endpoints/delete", exchange -> {
            if (!"DELETE".equals(exchange.getRequestMethod())) {
                sendError(exchange, 405, "method not allowed");
                return;
            }
            var params = parseQuery(exchange.getRequestURI().getQuery());
            try {
                dh.removeEndpoint(params.get("name"), params.get("host"), params.get("port"));
                writeJson(exchange, Map.of("status", "removed"));
            } catch (Exception e) {
                sendError(exchange, 400, e.getMessage());
            }
        });

        // PUT /endpoints/update — update an existing endpoint's target.
        // Body: {"name":"billing-api","old_host":"billing.internal","old_port":"8080",
        //        "new_host":"billing-v2.internal","new_port":"8080"}
        server.createContext("/endpoints/update", exchange -> {
            if (!"PUT".equals(exchange.getRequestMethod())) {
                sendError(exchange, 405, "method not allowed");
                return;
            }
            var req = MAPPER.readValue(exchange.getRequestBody(), UpdateRequest.class);
            var newEp = new Endpoint(req.newHost, req.newPort);
            var checker = HttpHealthChecker.builder().build();
            try {
                dh.updateEndpoint(req.name, req.oldHost, req.oldPort, newEp, checker);
                writeJson(exchange, Map.of("status", "updated"));
            } catch (Exception e) {
                sendError(exchange, 400, e.getMessage());
            }
        });

        server.start();
        System.out.println("Listening on http://localhost:8080");

        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            dh.stop();
            server.stop(0);
        }));

        Thread.currentThread().join();
    }

    record AddRequest(String name, String host, String port, boolean critical) {}

    record UpdateRequest(String name,
                         String oldHost, String oldPort,
                         String newHost, String newPort) {}

    private static void writeJson(HttpExchange exchange, Object obj) throws IOException {
        String body = MAPPER.writeValueAsString(obj);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(200, body.length());
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(body.getBytes());
        }
    }

    private static void sendError(HttpExchange exchange, int code, String msg) throws IOException {
        exchange.sendResponseHeaders(code, msg.length());
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(msg.getBytes());
        }
    }

    private static Map<String, String> parseQuery(String query) {
        if (query == null) {
            return Map.of();
        }
        var map = new java.util.HashMap<String, String>();
        for (String pair : query.split("&")) {
            String[] kv = pair.split("=", 2);
            map.put(kv[0], kv.length > 1 ? kv[1] : "");
        }
        return map;
    }
}
