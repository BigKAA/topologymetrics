package com.github.bigkaa.dephealth;

/**
 * Базовое исключение SDK dephealth.
 */
public class DepHealthException extends RuntimeException {

    public DepHealthException(String message) {
        super(message);
    }

    public DepHealthException(String message, Throwable cause) {
        super(message, cause);
    }
}
