package biz.kryukov.dev.dephealth;

/**
 * TLS/SSL error.
 */
public class CheckTlsException extends CheckException {

    public CheckTlsException(String message) {
        super(message, StatusCategory.TLS_ERROR, "tls_error");
    }

    public CheckTlsException(String message, Throwable cause) {
        super(message, cause, StatusCategory.TLS_ERROR, "tls_error");
    }
}
