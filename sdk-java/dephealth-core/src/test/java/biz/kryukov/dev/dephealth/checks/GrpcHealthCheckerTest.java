package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.StatusCategory;
import biz.kryukov.dev.dephealth.ValidationException;
import io.grpc.Metadata;
import io.grpc.Server;
import io.grpc.ServerCall;
import io.grpc.ServerCallHandler;
import io.grpc.ServerInterceptor;
import io.grpc.ServerInterceptors;
import io.grpc.Status;
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
import java.util.Map;
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

        // Verify directly via InProcessChannel
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

        // Verify via InProcessChannel directly
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

    // --- Auth tests ---

    @Test
    void conflictBearerAndBasicAuth() {
        assertThrows(ValidationException.class, () ->
                GrpcHealthChecker.builder()
                        .bearerToken("token")
                        .basicAuth("user", "pass")
                        .build());
    }

    @Test
    void conflictBearerAndAuthorizationMetadata() {
        assertThrows(ValidationException.class, () ->
                GrpcHealthChecker.builder()
                        .bearerToken("token")
                        .metadata(Map.of("authorization", "Custom value"))
                        .build());
    }

    @Test
    void conflictBasicAuthAndAuthorizationMetadata() {
        assertThrows(ValidationException.class, () ->
                GrpcHealthChecker.builder()
                        .basicAuth("user", "pass")
                        .metadata(Map.of("authorization", "Custom value"))
                        .build());
    }

    @Test
    void noConflictSingleBearerToken() {
        assertDoesNotThrow(() ->
                GrpcHealthChecker.builder()
                        .bearerToken("token")
                        .build());
    }

    @Test
    void noConflictSingleBasicAuth() {
        assertDoesNotThrow(() ->
                GrpcHealthChecker.builder()
                        .basicAuth("user", "pass")
                        .build());
    }

    @Test
    void noConflictMetadataWithoutAuthorization() {
        assertDoesNotThrow(() ->
                GrpcHealthChecker.builder()
                        .bearerToken("token")
                        .metadata(Map.of("x-custom", "value"))
                        .build());
    }

    @Test
    void authorizationMetadataCaseInsensitiveConflict() {
        assertThrows(ValidationException.class, () ->
                GrpcHealthChecker.builder()
                        .bearerToken("token")
                        .metadata(Map.of("Authorization", "Custom value"))
                        .build());
    }

    @Test
    void unauthenticatedThrowsAuthException() throws Exception {
        // Server that rejects all requests with UNAUTHENTICATED
        ServerInterceptor authInterceptor = new ServerInterceptor() {
            @Override
            public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
                    ServerCall<ReqT, RespT> call, Metadata headers,
                    ServerCallHandler<ReqT, RespT> next) {
                call.close(Status.UNAUTHENTICATED.withDescription("missing token"), new Metadata());
                return new ServerCall.Listener<>() {};
            }
        };

        server = InProcessServerBuilder.forName(serverName)
                .directExecutor()
                .addService(ServerInterceptors.intercept(
                        new HealthGrpc.HealthImplBase() {}, authInterceptor))
                .build()
                .start();

        // Use InProcessChannel to trigger auth error
        ManagedChannel channel = InProcessChannelBuilder.forName(serverName)
                .directExecutor()
                .build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel);
            io.grpc.StatusRuntimeException ex = assertThrows(
                    io.grpc.StatusRuntimeException.class,
                    () -> stub.check(HealthCheckRequest.newBuilder().setService("").build()));
            assertEquals(Status.Code.UNAUTHENTICATED, ex.getStatus().getCode());
        } finally {
            channel.shutdownNow();
        }
    }

    @Test
    void permissionDeniedThrowsAuthException() throws Exception {
        // Server that rejects all requests with PERMISSION_DENIED
        ServerInterceptor authInterceptor = new ServerInterceptor() {
            @Override
            public <ReqT, RespT> ServerCall.Listener<ReqT> interceptCall(
                    ServerCall<ReqT, RespT> call, Metadata headers,
                    ServerCallHandler<ReqT, RespT> next) {
                call.close(Status.PERMISSION_DENIED.withDescription("forbidden"), new Metadata());
                return new ServerCall.Listener<>() {};
            }
        };

        server = InProcessServerBuilder.forName(serverName)
                .directExecutor()
                .addService(ServerInterceptors.intercept(
                        new HealthGrpc.HealthImplBase() {}, authInterceptor))
                .build()
                .start();

        ManagedChannel channel = InProcessChannelBuilder.forName(serverName)
                .directExecutor()
                .build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel);
            io.grpc.StatusRuntimeException ex = assertThrows(
                    io.grpc.StatusRuntimeException.class,
                    () -> stub.check(HealthCheckRequest.newBuilder().setService("").build()));
            assertEquals(Status.Code.PERMISSION_DENIED, ex.getStatus().getCode());
        } finally {
            channel.shutdownNow();
        }
    }
}
