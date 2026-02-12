package biz.kryukov.dev.dephealth;

/**
 * Parameter validation error.
 */
public class ValidationException extends DepHealthException {

    public ValidationException(String message) {
        super(message);
    }
}
