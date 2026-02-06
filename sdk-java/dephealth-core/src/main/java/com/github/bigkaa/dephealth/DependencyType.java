package com.github.bigkaa.dephealth;

/**
 * Тип зависимости.
 */
public enum DependencyType {
    HTTP("http"),
    GRPC("grpc"),
    TCP("tcp"),
    POSTGRES("postgres"),
    MYSQL("mysql"),
    REDIS("redis"),
    AMQP("amqp"),
    KAFKA("kafka");

    private final String label;

    DependencyType(String label) {
        this.label = label;
    }

    /** Возвращает строковое представление для Prometheus-метки type. */
    public String label() {
        return label;
    }

    /** Находит тип по строковому представлению (case-insensitive). */
    public static DependencyType fromLabel(String label) {
        for (DependencyType t : values()) {
            if (t.label.equalsIgnoreCase(label)) {
                return t;
            }
        }
        throw new IllegalArgumentException("Unknown dependency type: " + label);
    }
}
