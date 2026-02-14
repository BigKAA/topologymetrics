package biz.kryukov.dev.dephealth;

/**
 * Dependency is reachable but unhealthy.
 */
public class UnhealthyException extends CheckException {

    public UnhealthyException(String message) {
        super(message, StatusCategory.UNHEALTHY, "unhealthy");
    }

    public UnhealthyException(String message, String detail) {
        super(message, StatusCategory.UNHEALTHY, detail);
    }

    public UnhealthyException(String message, String detail, Throwable cause) {
        super(message, cause, StatusCategory.UNHEALTHY, detail);
    }
}
