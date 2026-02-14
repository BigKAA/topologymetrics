package biz.kryukov.dev.dephealth;

/**
 * Connection error (refused, unreachable).
 */
public class CheckConnectionException extends CheckException {

    public CheckConnectionException(String message) {
        super(message, StatusCategory.CONNECTION_ERROR, "connection_refused");
    }

    public CheckConnectionException(String message, Throwable cause) {
        super(message, cause, StatusCategory.CONNECTION_ERROR, "connection_refused");
    }
}
