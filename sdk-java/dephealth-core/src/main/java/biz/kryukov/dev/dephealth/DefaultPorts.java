package biz.kryukov.dev.dephealth;

import java.util.Map;

/**
 * Порты по умолчанию для различных типов зависимостей и схем URL.
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
            Map.entry("kafka", "9092")
    );

    private static final Map<DependencyType, String> TYPE_TO_PORT = Map.of(
            DependencyType.POSTGRES, "5432",
            DependencyType.MYSQL, "3306",
            DependencyType.REDIS, "6379",
            DependencyType.AMQP, "5672",
            DependencyType.HTTP, "80",
            DependencyType.GRPC, "443",
            DependencyType.KAFKA, "9092"
    );

    /** Возвращает порт по умолчанию для схемы URL (lowercase). */
    public static String forScheme(String scheme) {
        return SCHEME_TO_PORT.get(scheme.toLowerCase());
    }

    /** Возвращает порт по умолчанию для типа зависимости. */
    public static String forType(DependencyType type) {
        return TYPE_TO_PORT.get(type);
    }
}
