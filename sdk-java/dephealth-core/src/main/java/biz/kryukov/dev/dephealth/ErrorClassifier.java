package biz.kryukov.dev.dephealth;

import java.net.ConnectException;
import java.net.SocketTimeoutException;
import java.net.UnknownHostException;
import java.net.http.HttpTimeoutException;
import javax.net.ssl.SSLException;

/**
 * Classifies check exceptions into {@link CheckResult} with status category and detail.
 *
 * <p>Classification chain:
 * <ol>
 *   <li>{@link CheckException} with statusCategory/statusDetail</li>
 *   <li>Platform exception types (timeout, DNS, connection, TLS)</li>
 *   <li>Wrapped exception cause (recursive)</li>
 *   <li>Fallback: error/error</li>
 * </ol>
 */
public final class ErrorClassifier {

    private ErrorClassifier() {}

    /**
     * Classifies an exception into a CheckResult.
     *
     * @param err the exception to classify, or null for success
     * @return the classification result
     */
    public static CheckResult classify(Exception err) {
        if (err == null) {
            return CheckResult.OK;
        }

        // 1. CheckException with explicit classification.
        if (err instanceof CheckException ce) {
            return new CheckResult(ce.statusCategory(), ce.statusDetail());
        }

        // 2. Platform exception types.
        CheckResult platform = classifyPlatform(err);
        if (platform != null) {
            return platform;
        }

        // 3. Check wrapped cause.
        Throwable cause = err.getCause();
        if (cause instanceof Exception inner && cause != err) {
            CheckResult innerResult = classify(inner);
            if (!StatusCategory.ERROR.equals(innerResult.category())) {
                return innerResult;
            }
        }

        // 4. Fallback.
        return new CheckResult(StatusCategory.ERROR, "error");
    }

    private static CheckResult classifyPlatform(Throwable err) {
        if (err instanceof SocketTimeoutException) {
            return new CheckResult(StatusCategory.TIMEOUT, "timeout");
        }
        if (err instanceof java.util.concurrent.TimeoutException) {
            return new CheckResult(StatusCategory.TIMEOUT, "timeout");
        }
        if (err instanceof HttpTimeoutException) {
            return new CheckResult(StatusCategory.TIMEOUT, "timeout");
        }
        if (err instanceof UnknownHostException) {
            return new CheckResult(StatusCategory.DNS_ERROR, "dns_error");
        }
        if (err instanceof ConnectException) {
            return new CheckResult(StatusCategory.CONNECTION_ERROR, "connection_refused");
        }
        if (err instanceof java.net.NoRouteToHostException) {
            return new CheckResult(StatusCategory.CONNECTION_ERROR, "connection_refused");
        }
        if (err instanceof SSLException) {
            return new CheckResult(StatusCategory.TLS_ERROR, "tls_error");
        }
        return null;
    }
}
