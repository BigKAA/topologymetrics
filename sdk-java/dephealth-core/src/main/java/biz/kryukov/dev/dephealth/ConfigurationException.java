package biz.kryukov.dev.dephealth;

/**
 * Ошибка конфигурации SDK.
 */
public class ConfigurationException extends DepHealthException {

    public ConfigurationException(String message) {
        super(message);
    }

    public ConfigurationException(String message, Throwable cause) {
        super(message, cause);
    }
}
