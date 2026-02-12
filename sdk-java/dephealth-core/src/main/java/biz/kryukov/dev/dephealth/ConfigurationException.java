package biz.kryukov.dev.dephealth;

/**
 * SDK configuration error.
 */
public class ConfigurationException extends DepHealthException {

    public ConfigurationException(String message) {
        super(message);
    }

    public ConfigurationException(String message, Throwable cause) {
        super(message, cause);
    }
}
