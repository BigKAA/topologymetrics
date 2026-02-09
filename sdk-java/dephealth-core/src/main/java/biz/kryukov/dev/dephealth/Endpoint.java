package biz.kryukov.dev.dephealth;

import java.util.Collections;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Эндпоинт зависимости (хост + порт + метки). Immutable.
 */
public final class Endpoint {

    /** Паттерн имени метки: начинается с буквы или _, далее буквы, цифры, _. */
    private static final Pattern LABEL_NAME_PATTERN =
            Pattern.compile("^[a-zA-Z_][a-zA-Z0-9_]*$");

    /** Метки, зарезервированные SDK — нельзя переопределять через WithLabel/label(). */
    public static final Set<String> RESERVED_LABELS = Set.of(
            "name", "dependency", "type", "host", "port", "critical"
    );

    private final String host;
    private final String port;
    private final Map<String, String> metadata;

    public Endpoint(String host, String port) {
        this(host, port, Map.of());
    }

    public Endpoint(String host, String port, Map<String, String> metadata) {
        this.host = Objects.requireNonNull(host, "host");
        this.port = Objects.requireNonNull(port, "port");
        this.metadata = metadata == null ? Map.of() : Collections.unmodifiableMap(metadata);
    }

    public String host() {
        return host;
    }

    public String port() {
        return port;
    }

    public int portAsInt() {
        return Integer.parseInt(port);
    }

    /**
     * Возвращает произвольные метки (labels) эндпоинта.
     *
     * @return неизменяемая карта меток
     */
    public Map<String, String> labels() {
        return metadata;
    }

    /**
     * @deprecated Используйте {@link #labels()}.
     */
    @Deprecated
    public Map<String, String> metadata() {
        return metadata;
    }

    /**
     * Проверяет имя метки на соответствие паттерну Prometheus.
     *
     * @param name имя метки
     * @throws ValidationException если имя не соответствует паттерну или является зарезервированным
     */
    public static void validateLabelName(String name) {
        Objects.requireNonNull(name, "label name");
        if (!LABEL_NAME_PATTERN.matcher(name).matches()) {
            throw new ValidationException(
                    "label name must match " + LABEL_NAME_PATTERN.pattern()
                            + ", got '" + name + "'");
        }
        if (RESERVED_LABELS.contains(name)) {
            throw new ValidationException(
                    "label name '" + name + "' is reserved and cannot be used as a custom label");
        }
    }

    /**
     * Валидирует все метки из карты: имена и зарезервированность.
     *
     * @param labels карта меток
     * @throws ValidationException если какое-либо имя невалидно
     */
    public static void validateLabels(Map<String, String> labels) {
        if (labels == null) {
            return;
        }
        for (String key : labels.keySet()) {
            validateLabelName(key);
        }
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) {
            return true;
        }
        if (!(o instanceof Endpoint ep)) {
            return false;
        }
        return host.equals(ep.host) && port.equals(ep.port);
    }

    @Override
    public int hashCode() {
        return Objects.hash(host, port);
    }

    @Override
    public String toString() {
        return host + ":" + port;
    }
}
