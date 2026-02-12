package biz.kryukov.dev.dephealth;

/**
 * Base exception for the dephealth SDK.
 */
public class DepHealthException extends RuntimeException {

    public DepHealthException(String message) {
        super(message);
    }

    public DepHealthException(String message, Throwable cause) {
        super(message, cause);
    }
}
