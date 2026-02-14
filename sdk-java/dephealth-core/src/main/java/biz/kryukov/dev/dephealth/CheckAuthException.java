package biz.kryukov.dev.dephealth;

/**
 * Authentication/authorization failure.
 */
public class CheckAuthException extends CheckException {

    public CheckAuthException(String message) {
        super(message, StatusCategory.AUTH_ERROR, "auth_error");
    }

    public CheckAuthException(String message, Throwable cause) {
        super(message, cause, StatusCategory.AUTH_ERROR, "auth_error");
    }
}
