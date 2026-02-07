package biz.kryukov.dev.dephealth;

import java.time.Duration;

/**
 * Конфигурация проверки здоровья зависимости. Immutable, создаётся через Builder.
 */
public final class CheckConfig {

    public static final Duration DEFAULT_INTERVAL = Duration.ofSeconds(15);
    public static final Duration DEFAULT_TIMEOUT = Duration.ofSeconds(5);
    public static final Duration DEFAULT_INITIAL_DELAY = Duration.ofSeconds(5);
    public static final int DEFAULT_FAILURE_THRESHOLD = 1;
    public static final int DEFAULT_SUCCESS_THRESHOLD = 1;

    public static final Duration MIN_INTERVAL = Duration.ofSeconds(1);
    public static final Duration MAX_INTERVAL = Duration.ofMinutes(10);
    public static final Duration MIN_TIMEOUT = Duration.ofMillis(100);
    public static final Duration MAX_TIMEOUT = Duration.ofSeconds(30);
    public static final Duration MIN_INITIAL_DELAY = Duration.ZERO;
    public static final Duration MAX_INITIAL_DELAY = Duration.ofMinutes(5);
    public static final int MIN_THRESHOLD = 1;
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

    public Duration interval() {
        return interval;
    }

    public Duration timeout() {
        return timeout;
    }

    public Duration initialDelay() {
        return initialDelay;
    }

    public int failureThreshold() {
        return failureThreshold;
    }

    public int successThreshold() {
        return successThreshold;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static CheckConfig defaults() {
        return builder().build();
    }

    public static final class Builder {
        private Duration interval = DEFAULT_INTERVAL;
        private Duration timeout = DEFAULT_TIMEOUT;
        private Duration initialDelay = DEFAULT_INITIAL_DELAY;
        private int failureThreshold = DEFAULT_FAILURE_THRESHOLD;
        private int successThreshold = DEFAULT_SUCCESS_THRESHOLD;

        private Builder() {}

        public Builder interval(Duration interval) {
            this.interval = interval;
            return this;
        }

        public Builder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        public Builder initialDelay(Duration initialDelay) {
            this.initialDelay = initialDelay;
            return this;
        }

        public Builder failureThreshold(int failureThreshold) {
            this.failureThreshold = failureThreshold;
            return this;
        }

        public Builder successThreshold(int successThreshold) {
            this.successThreshold = successThreshold;
            return this;
        }

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
