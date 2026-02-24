package biz.kryukov.dev.dephealth;

import java.util.Collections;
import java.util.List;
import java.util.Objects;
import java.util.regex.Pattern;

/**
 * Dependency descriptor: name, type, endpoints, and check configuration. Immutable.
 */
public final class Dependency {

    private static final Pattern NAME_PATTERN = Pattern.compile("^[a-z][a-z0-9-]*$");
    private static final int MAX_NAME_LENGTH = 63;

    private final String name;
    private final DependencyType type;
    private final boolean critical;
    private final List<Endpoint> endpoints;
    private final CheckConfig config;

    private Dependency(Builder builder) {
        this.name = builder.name;
        this.type = builder.type;
        this.critical = builder.criticalValue;
        this.endpoints = Collections.unmodifiableList(builder.endpoints);
        this.config = builder.config;
    }

    public String name() {
        return name;
    }

    public DependencyType type() {
        return type;
    }

    public boolean critical() {
        return critical;
    }

    public List<Endpoint> endpoints() {
        return endpoints;
    }

    public CheckConfig config() {
        return config;
    }

    /**
     * Converts a boolean to "yes"/"no" string for the critical label.
     */
    public static String boolToYesNo(boolean value) {
        return value ? "yes" : "no";
    }

    /**
     * Validates that the dependency name matches the naming rules.
     *
     * @param name dependency name
     * @throws ValidationException if the name is invalid
     */
    public static void validateName(String name) {
        Objects.requireNonNull(name, "dependency name");
        if (name.isEmpty() || name.length() > MAX_NAME_LENGTH) {
            throw new ValidationException(
                    "dependency name must be 1-" + MAX_NAME_LENGTH + " characters, got '"
                            + name + "' (" + name.length() + " chars)");
        }
        if (!NAME_PATTERN.matcher(name).matches()) {
            throw new ValidationException(
                    "dependency name must match " + NAME_PATTERN.pattern()
                            + ", got '" + name + "'");
        }
    }

    public static Builder builder(String name, DependencyType type) {
        return new Builder(name, type);
    }

    public static final class Builder {
        private final String name;
        private final DependencyType type;
        private Boolean criticalValue;
        private List<Endpoint> endpoints = List.of();
        private CheckConfig config = CheckConfig.defaults();

        private Builder(String name, DependencyType type) {
            this.name = Objects.requireNonNull(name, "name");
            this.type = Objects.requireNonNull(type, "type");
        }

        /**
         * Sets the criticality of the dependency. Required parameter.
         */
        public Builder critical(boolean critical) {
            this.criticalValue = critical;
            return this;
        }

        public Builder endpoints(List<Endpoint> endpoints) {
            this.endpoints = List.copyOf(endpoints);
            return this;
        }

        public Builder endpoint(Endpoint endpoint) {
            this.endpoints = List.of(endpoint);
            return this;
        }

        public Builder config(CheckConfig config) {
            this.config = Objects.requireNonNull(config, "config");
            return this;
        }

        public Dependency build() {
            validate();
            return new Dependency(this);
        }

        private void validate() {
            validateName(name);
            if (endpoints.isEmpty()) {
                throw new ValidationException("dependency '" + name + "' must have at least one endpoint");
            }
            if (criticalValue == null) {
                throw new ValidationException(
                        "dependency '" + name + "' must have critical flag set explicitly");
            }
            // Validate labels of each endpoint
            for (Endpoint ep : endpoints) {
                Endpoint.validateLabels(ep.labels());
            }
        }
    }
}
