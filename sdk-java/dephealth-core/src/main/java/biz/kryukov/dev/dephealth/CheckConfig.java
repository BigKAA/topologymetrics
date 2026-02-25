package biz.kryukov.dev.dephealth;

import java.time.Duration;

/**
 * Dependency health check configuration. Immutable, created via Builder.
 */
public final class CheckConfig {

    /** Default check interval: 15 seconds. */
    public static final Duration DEFAULT_INTERVAL = Duration.ofSeconds(15);
    /** Default check timeout: 5 seconds. */
    public static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(5);
    /** Default initial delay before first check: 5 seconds. */
    public static final Duration DEFAULT_INITIAL_DELAY = Duration.ofSeconds(5);
    /** Default failure threshold: 1 consecutive failure. */
    public static final int DEFAULT_FAILURE_THRESHOLD = 1;
    /** Default success threshold: 1 consecutive success. */
    public static final int DEFAULT_SUCCESS_THRESHOLD = 1;

    /** Minimum allowed check interval. */
    public static final Duration MIN_INTERVAL = Duration.ofSeconds(1);
    /** Maximum allowed check interval. */
    public static final Duration MAX_INTERVAL = Duration.ofMinutes(10);
    /** Minimum allowed check timeout. */
    public static final Duration MIN_TIMEOUT = Duration.ofMillis(100);
    /** Maximum allowed check timeout. */
    public static final Duration MAX_TIMEOUT = Duration.ofSeconds(30);
    /** Minimum allowed initial delay. */
    public static final Duration MIN_INITIAL_DELAY = Duration.ZERO;
    /** Maximum allowed initial delay. */
    public static final Duration MAX_INITIAL_DELAY = Duration.ofMinutes(5);
    /** Minimum allowed threshold value. */
    public static final int MIN_THRESHOLD = 1;
    /** Maximum allowed threshold value. */
    public static final int MAX_THRESHOLD = 10;

    private final Duration interval;
    private final Duration timeout;
    private final Duration initialDelay;
    private final int failureThreshold;
    private final int successThreshold;

    private CheckConfig(Builder builder) {
        this.interval = builder.interval;
        this.timeout = builder.timeout;
        this.initialDelay = builder.initialDelay;
        this.failureThreshold = builder.failureThreshold;
        this.successThreshold = builder.successThreshold;
    }

    /** Returns the check interval. */
    public Duration interval() {
        return interval;
    }

    /** Returns the check timeout. */
    public Duration timeout() {
        return timeout;
    }

    /** Returns the initial delay before the first check. */
    public Duration initialDelay() {
        return initialDelay;
    }

    /** Returns the number of consecutive failures to become unhealthy. */
    public int failureThreshold() {
        return failureThreshold;
    }

    /** Returns the number of consecutive successes to become healthy. */
    public int successThreshold() {
        return successThreshold;
    }

    /** Creates a new builder with default values. */
    public static Builder builder() {
        return new Builder();
    }

    /** Returns a configuration with all default values. */
    public static CheckConfig defaults() {
        return builder().build();
    }

    /** Builder for {@link CheckConfig}. */
    public static final class Builder {
        private Duration interval = DEFAULT_INTERVAL;
        private Duration timeout = DEFAULT_TIMEOUT;
        private Duration initialDelay = DEFAULT_INITIAL_DELAY;
        private int failureThreshold = DEFAULT_FAILURE_THRESHOLD;
        private int successThreshold = DEFAULT_SUCCESS_THRESHOLD;

        private Builder() {}

        /** Sets the check interval. */
        public Builder interval(Duration interval) {
            this.interval = interval;
            return this;
        }

        /** Sets the check timeout (must be less than interval). */
        public Builder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        /** Sets the initial delay before the first check. */
        public Builder initialDelay(Duration initialDelay) {
            this.initialDelay = initialDelay;
            return this;
        }

        /** Sets the consecutive failure threshold. */
        public Builder failureThreshold(int failureThreshold) {
            this.failureThreshold = failureThreshold;
            return this;
        }

        /** Sets the consecutive success threshold. */
        public Builder successThreshold(int successThreshold) {
            this.successThreshold = successThreshold;
            return this;
        }

        /** Builds and validates the configuration. */
        public CheckConfig build() {
            validate();
            return new CheckConfig(this);
        }

        private void validate() {
            if (interval.compareTo(MIN_INTERVAL) < 0 || interval.compareTo(MAX_INTERVAL) > 0) {
                throw new ValidationException(
                        "interval must be between " + MIN_INTERVAL + " and " + MAX_INTERVAL
                                + ", got " + interval);
            }
            if (timeout.compareTo(MIN_TIMEOUT) < 0 || timeout.compareTo(MAX_TIMEOUT) > 0) {
                throw new ValidationException(
                        "timeout must be between " + MIN_TIMEOUT + " and " + MAX_TIMEOUT
                                + ", got " + timeout);
            }
            if (timeout.compareTo(interval) >= 0) {
                throw new ValidationException(
                        "timeout (" + timeout + ") must be less than interval (" + interval + ")");
            }
            if (initialDelay.compareTo(MIN_INITIAL_DELAY) < 0
                    || initialDelay.compareTo(MAX_INITIAL_DELAY) > 0) {
                throw new ValidationException(
                        "initialDelay must be between " + MIN_INITIAL_DELAY + " and "
                                + MAX_INITIAL_DELAY + ", got " + initialDelay);
            }
            if (failureThreshold < MIN_THRESHOLD || failureThreshold > MAX_THRESHOLD) {
                throw new ValidationException(
                        "failureThreshold must be between " + MIN_THRESHOLD + " and "
                                + MAX_THRESHOLD + ", got " + failureThreshold);
            }
            if (successThreshold < MIN_THRESHOLD || successThreshold > MAX_THRESHOLD) {
                throw new ValidationException(
                        "successThreshold must be between " + MIN_THRESHOLD + " and "
                                + MAX_THRESHOLD + ", got " + successThreshold);
            }
        }
    }
}
