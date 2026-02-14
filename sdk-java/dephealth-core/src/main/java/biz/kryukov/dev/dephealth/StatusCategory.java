package biz.kryukov.dev.dephealth;

import java.util.List;

/**
 * Status category constants for health check classification.
 */
public final class StatusCategory {

    public static final String OK = "ok";
    public static final String TIMEOUT = "timeout";
    public static final String CONNECTION_ERROR = "connection_error";
    public static final String DNS_ERROR = "dns_error";
    public static final String AUTH_ERROR = "auth_error";
    public static final String TLS_ERROR = "tls_error";
    public static final String UNHEALTHY = "unhealthy";
    public static final String ERROR = "error";

    /** All status category values in specification order. */
    public static final List<String> ALL = List.of(
            OK, TIMEOUT, CONNECTION_ERROR, DNS_ERROR, AUTH_ERROR, TLS_ERROR, UNHEALTHY, ERROR
    );

    private StatusCategory() {}
}
