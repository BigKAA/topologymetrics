package biz.kryukov.dev.dephealth;

/**
 * Check timed out.
 */
public class CheckTimeoutException extends CheckException {

    public CheckTimeoutException(String message) {
        super(message, StatusCategory.TIMEOUT, "timeout");
    }

    public CheckTimeoutException(String message, Throwable cause) {
        super(message, cause, StatusCategory.TIMEOUT, "timeout");
    }
}
