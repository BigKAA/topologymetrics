package biz.kryukov.dev.dephealth;

import java.util.Map;

/**
 * Default ports for various dependency types and URL schemes.
 */
public final class DefaultPorts {

    private DefaultPorts() {}

    private static final Map<String, String> SCHEME_TO_PORT = Map.ofEntries(
            Map.entry("postgres", "5432"),
            Map.entry("postgresql", "5432"),
            Map.entry("mysql", "3306"),
            Map.entry("redis", "6379"),
            Map.entry("rediss", "6379"),
            Map.entry("amqp", "5672"),
            Map.entry("amqps", "5671"),
            Map.entry("http", "80"),
            Map.entry("https", "443"),
            Map.entry("grpc", "443"),
            Map.entry("kafka", "9092"),
            Map.entry("ldap", "389"),
            Map.entry("ldaps", "636")
    );

    private static final Map<DependencyType, String> TYPE_TO_PORT = Map.ofEntries(
            Map.entry(DependencyType.POSTGRES, "5432"),
            Map.entry(DependencyType.MYSQL, "3306"),
            Map.entry(DependencyType.REDIS, "6379"),
            Map.entry(DependencyType.AMQP, "5672"),
            Map.entry(DependencyType.HTTP, "80"),
            Map.entry(DependencyType.GRPC, "443"),
            Map.entry(DependencyType.KAFKA, "9092"),
            Map.entry(DependencyType.LDAP, "389")
    );

    /** Returns the default port for a URL scheme (lowercase). */
    public static String forScheme(String scheme) {
        return SCHEME_TO_PORT.get(scheme.toLowerCase());
    }

    /** Returns the default port for a dependency type. */
    public static String forType(DependencyType type) {
        return TYPE_TO_PORT.get(type);
    }
}
