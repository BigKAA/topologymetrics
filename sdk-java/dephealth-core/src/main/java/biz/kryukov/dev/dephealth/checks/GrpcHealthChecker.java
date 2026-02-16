package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;
import biz.kryukov.dev.dephealth.UnhealthyException;
import biz.kryukov.dev.dephealth.ValidationException;

import io.grpc.CallOptions;
import io.grpc.Channel;
import io.grpc.ClientCall;
import io.grpc.ClientInterceptor;
import io.grpc.ForwardingClientCall;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.Metadata;
import io.grpc.MethodDescriptor;
import io.grpc.Status;
import io.grpc.StatusRuntimeException;
import io.grpc.health.v1.HealthCheckRequest;
import io.grpc.health.v1.HealthCheckResponse;
import io.grpc.health.v1.HealthGrpc;

import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.Base64;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * gRPC health checker — uses the gRPC Health Checking Protocol.
 */
public final class GrpcHealthChecker implements HealthChecker {

    private final String serviceName;
    private final boolean tlsEnabled;
    private final boolean tlsSkipVerify;
    private final Map<String, String> metadata;

    private GrpcHealthChecker(Builder builder) {
        this.serviceName = builder.serviceName;
        this.tlsEnabled = builder.tlsEnabled;
        this.tlsSkipVerify = builder.tlsSkipVerify;
        this.metadata = Collections.unmodifiableMap(new LinkedHashMap<>(builder.resolvedMetadata));
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

        // Attach metadata interceptor if configured.
        if (!metadata.isEmpty()) {
            channelBuilder.intercept(new MetadataInterceptor(metadata));
        }

        ManagedChannel channel = channelBuilder.build();
        try {
            HealthGrpc.HealthBlockingStub stub = HealthGrpc.newBlockingStub(channel)
                    .withDeadlineAfter(timeout.toMillis(), TimeUnit.MILLISECONDS);

            HealthCheckRequest request = HealthCheckRequest.newBuilder()
                    .setService(serviceName)
                    .build();

            HealthCheckResponse response;
            try {
                response = stub.check(request);
            } catch (StatusRuntimeException e) {
                // Classify UNAUTHENTICATED and PERMISSION_DENIED as auth_error.
                Status.Code code = e.getStatus().getCode();
                if (code == Status.Code.UNAUTHENTICATED
                        || code == Status.Code.PERMISSION_DENIED) {
                    throw new CheckAuthException(
                            "gRPC health check: " + e.getStatus(), e);
                }
                throw e;
            }

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
        private Map<String, String> customMetadata = new LinkedHashMap<>();
        private String bearerToken;
        private String basicAuthUsername;
        private String basicAuthPassword;
        private final Map<String, String> resolvedMetadata = new LinkedHashMap<>();

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

        public Builder metadata(Map<String, String> metadata) {
            this.customMetadata = new LinkedHashMap<>(metadata);
            return this;
        }

        public Builder bearerToken(String token) {
            this.bearerToken = token;
            return this;
        }

        public Builder basicAuth(String username, String password) {
            this.basicAuthUsername = username;
            this.basicAuthPassword = password;
            return this;
        }

        public GrpcHealthChecker build() {
            validateAuthConflicts();
            buildResolvedMetadata();
            return new GrpcHealthChecker(this);
        }

        private void validateAuthConflicts() {
            int methods = 0;
            if (bearerToken != null && !bearerToken.isEmpty()) {
                methods++;
            }
            if (basicAuthUsername != null && !basicAuthUsername.isEmpty()) {
                methods++;
            }
            for (String key : customMetadata.keySet()) {
                if (key.equalsIgnoreCase("authorization")) {
                    methods++;
                    break;
                }
            }
            if (methods > 1) {
                throw new ValidationException(
                        "conflicting auth methods: specify only one of "
                                + "bearerToken, basicAuth, or authorization metadata");
            }
        }

        private void buildResolvedMetadata() {
            resolvedMetadata.putAll(customMetadata);
            if (bearerToken != null && !bearerToken.isEmpty()) {
                resolvedMetadata.put("authorization", "Bearer " + bearerToken);
            }
            if (basicAuthUsername != null && !basicAuthUsername.isEmpty()) {
                String credentials = basicAuthUsername + ":"
                        + (basicAuthPassword != null ? basicAuthPassword : "");
                String encoded = Base64.getEncoder()
                        .encodeToString(credentials.getBytes(StandardCharsets.UTF_8));
                resolvedMetadata.put("authorization", "Basic " + encoded);
            }
        }
    }

    /**
     * Interceptor that attaches custom metadata to every gRPC call.
     */
    private static final class MetadataInterceptor implements ClientInterceptor {

        private final Map<String, String> extra;

        MetadataInterceptor(Map<String, String> extra) {
            this.extra = extra;
        }

        @Override
        public <ReqT, RespT> ClientCall<ReqT, RespT> interceptCall(
                MethodDescriptor<ReqT, RespT> method,
                CallOptions callOptions,
                Channel next) {
            return new ForwardingClientCall.SimpleForwardingClientCall<>(
                    next.newCall(method, callOptions)) {
                @Override
                public void start(Listener<RespT> responseListener, Metadata headers) {
                    for (Map.Entry<String, String> entry : extra.entrySet()) {
                        Metadata.Key<String> key = Metadata.Key.of(
                                entry.getKey(), Metadata.ASCII_STRING_MARSHALLER);
                        headers.put(key, entry.getValue());
                    }
                    super.start(responseListener, headers);
                }
            };
        }
    }
}
