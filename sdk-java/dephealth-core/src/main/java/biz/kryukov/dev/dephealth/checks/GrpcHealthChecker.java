package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.health.v1.HealthCheckRequest;
import io.grpc.health.v1.HealthCheckResponse;
import io.grpc.health.v1.HealthGrpc;

import java.time.Duration;
import java.util.concurrent.TimeUnit;

/**
 * gRPC health checker — uses the gRPC Health Checking Protocol.
 */
public final class GrpcHealthChecker implements HealthChecker {

    private final String serviceName;
    private final boolean tlsEnabled;
    private final boolean tlsSkipVerify;

    private GrpcHealthChecker(Builder builder) {
        this.serviceName = builder.serviceName;
        this.tlsEnabled = builder.tlsEnabled;
        this.tlsSkipVerify = builder.tlsSkipVerify;
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        String target = endpoint.host() + ":" + endpoint.port();

        ManagedChannelBuilder<?> channelBuilder = ManagedChannelBuilder.forTarget(target);

        if (tlsEnabled) {
            channelBuilder.useTransportSecurity();
        } else {
            channelBuilder.usePlaintext();
        }

        ManagedChannel channel = channelBuilder.build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel)
                    .withDeadlineAfter(timeout.toMillis(), TimeUnit.MILLISECONDS);

            HealthCheckRequest request = HealthCheckRequest.newBuilder()
                    .setService(serviceName)
                    .build();

            HealthCheckResponse response = stub.check(request);

            if (response.getStatus() != HealthCheckResponse.ServingStatus.SERVING) {
                String detail = response.getStatus() == HealthCheckResponse.ServingStatus.UNKNOWN
                        ? "grpc_unknown" : "grpc_not_serving";
                throw new UnhealthyException(
                        "gRPC health check: status " + response.getStatus(), detail);
            }
        } finally {
            channel.shutdownNow();
            channel.awaitTermination(1, TimeUnit.SECONDS);
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.GRPC;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String serviceName = "";
        private boolean tlsEnabled;
        private boolean tlsSkipVerify;

        private Builder() {}

        public Builder serviceName(String serviceName) {
            this.serviceName = serviceName;
            return this;
        }

        public Builder tlsEnabled(boolean tlsEnabled) {
            this.tlsEnabled = tlsEnabled;
            return this;
        }

        public Builder tlsSkipVerify(boolean tlsSkipVerify) {
            this.tlsSkipVerify = tlsSkipVerify;
            return this;
        }

        public GrpcHealthChecker build() {
            return new GrpcHealthChecker(this);
        }
    }
}
