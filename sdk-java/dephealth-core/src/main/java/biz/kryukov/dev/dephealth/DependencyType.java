package biz.kryukov.dev.dephealth;

/**
 * Supported dependency types, corresponding to the {@code type} Prometheus label value.
 */
public enum DependencyType {
    HTTP("http"),
    GRPC("grpc"),
    TCP("tcp"),
    POSTGRES("postgres"),
    MYSQL("mysql"),
    REDIS("redis"),
    AMQP("amqp"),
    KAFKA("kafka"),
    LDAP("ldap");

    private final String label;

    DependencyType(String label) {
        this.label = label;
    }

    /** Returns the string representation for the Prometheus type label. */
    public String label() {
        return label;
    }

    /** Finds a type by its string representation (case-insensitive). */
    public static DependencyType fromLabel(String label) {
        for (DependencyType t : values()) {
            if (t.label.equalsIgnoreCase(label)) {
                return t;
            }
        }
        throw new IllegalArgumentException("Unknown dependency type: " + label);
    }
}
