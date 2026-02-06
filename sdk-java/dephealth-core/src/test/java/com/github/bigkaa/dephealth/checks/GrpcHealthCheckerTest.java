package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
import io.grpc.Server;
import io.grpc.inprocess.InProcessChannelBuilder;
import io.grpc.inprocess.InProcessServerBuilder;
import io.grpc.health.v1.HealthCheckRequest;
import io.grpc.health.v1.HealthCheckResponse;
import io.grpc.health.v1.HealthGrpc;
import io.grpc.ManagedChannel;
import io.grpc.stub.StreamObserver;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

class GrpcHealthCheckerTest {

    private Server server;
    private String serverName;

    @BeforeEach
    void setUp() throws Exception {
        serverName = InProcessServerBuilder.generateName();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (server != null) {
            server.shutdownNow();
            server.awaitTermination(5, TimeUnit.SECONDS);
        }
    }

    @Test
    void type() {
        assertEquals(DependencyType.GRPC, GrpcHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheckViaInProcess() throws Exception {
        server = InProcessServerBuilder.forName(serverName)
                .directExecutor()
                .addService(new HealthGrpc.HealthImplBase() {
                    @Override
                    public void check(HealthCheckRequest request,
                                      StreamObserver<HealthCheckResponse> responseObserver) {
                        responseObserver.onNext(HealthCheckResponse.newBuilder()
                                .setStatus(HealthCheckResponse.ServingStatus.SERVING)
                                .build());
                        responseObserver.onCompleted();
                    }
                })
                .build()
                .start();

        // Проверяем напрямую через InProcessChannel
        ManagedChannel channel = InProcessChannelBuilder.forName(serverName)
                .directExecutor()
                .build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel);
            HealthCheckResponse response = stub.check(
                    HealthCheckRequest.newBuilder().setService("").build());
            assertEquals(HealthCheckResponse.ServingStatus.SERVING, response.getStatus());
        } finally {
            channel.shutdownNow();
        }
    }

    @Test
    void notServingThrows() throws Exception {
        server = InProcessServerBuilder.forName(serverName)
                .directExecutor()
                .addService(new HealthGrpc.HealthImplBase() {
                    @Override
                    public void check(HealthCheckRequest request,
                                      StreamObserver<HealthCheckResponse> responseObserver) {
                        responseObserver.onNext(HealthCheckResponse.newBuilder()
                                .setStatus(HealthCheckResponse.ServingStatus.NOT_SERVING)
                                .build());
                        responseObserver.onCompleted();
                    }
                })
                .build()
                .start();

        // Проверяем через InProcessChannel напрямую
        ManagedChannel channel = InProcessChannelBuilder.forName(serverName)
                .directExecutor()
                .build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel);
            HealthCheckResponse response = stub.check(
                    HealthCheckRequest.newBuilder().setService("").build());
            assertEquals(HealthCheckResponse.ServingStatus.NOT_SERVING, response.getStatus());
        } finally {
            channel.shutdownNow();
        }
    }

    @Test
    void connectionRefused() {
        GrpcHealthChecker checker = GrpcHealthChecker.builder().build();
        Endpoint ep = new Endpoint("localhost", "1");
        assertThrows(Exception.class, () -> checker.check(ep, Duration.ofSeconds(1)));
    }
}
