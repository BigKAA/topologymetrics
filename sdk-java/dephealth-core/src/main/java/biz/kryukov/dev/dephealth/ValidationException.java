package biz.kryukov.dev.dephealth;

/**
 * Ошибка валидации параметров.
 */
public class ValidationException extends DepHealthException {

    public ValidationException(String message) {
        super(message);
    }
}
