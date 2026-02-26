namespace DepHealth;

/// <summary>
/// Dependency health check configuration. Immutable, created via Builder.
/// </summary>
public sealed class CheckConfig
{
    /// <summary>Default value for <see cref="Interval"/>.</summary>
    public static readonly TimeSpan DefaultInterval = TimeSpan.FromSeconds(15);

    /// <summary>Default value for <see cref="Timeout"/>.</summary>
    public static readonly TimeSpan DefaultTimeout = TimeSpan.FromSeconds(5);

    /// <summary>Default value for <see cref="InitialDelay"/>.</summary>
    public static readonly TimeSpan DefaultInitialDelay = TimeSpan.FromSeconds(5);

    /// <summary>Default value for <see cref="FailureThreshold"/>.</summary>
    public const int DefaultFailureThreshold = 1;

    /// <summary>Default value for <see cref="SuccessThreshold"/>.</summary>
    public const int DefaultSuccessThreshold = 1;

    /// <summary>Minimum allowed value for <see cref="Interval"/>.</summary>
    public static readonly TimeSpan MinInterval = TimeSpan.FromSeconds(1);

    /// <summary>Maximum allowed value for <see cref="Interval"/>.</summary>
    public static readonly TimeSpan MaxInterval = TimeSpan.FromMinutes(10);

    /// <summary>Minimum allowed value for <see cref="Timeout"/>.</summary>
    public static readonly TimeSpan MinTimeout = TimeSpan.FromMilliseconds(100);

    /// <summary>Maximum allowed value for <see cref="Timeout"/>.</summary>
    public static readonly TimeSpan MaxTimeout = TimeSpan.FromSeconds(30);

    /// <summary>Minimum allowed value for <see cref="InitialDelay"/>.</summary>
    public static readonly TimeSpan MinInitialDelay = TimeSpan.Zero;

    /// <summary>Maximum allowed value for <see cref="InitialDelay"/>.</summary>
    public static readonly TimeSpan MaxInitialDelay = TimeSpan.FromMinutes(5);

    /// <summary>Minimum allowed threshold value.</summary>
    public const int MinThreshold = 1;

    /// <summary>Maximum allowed threshold value.</summary>
    public const int MaxThreshold = 10;

    /// <summary>Interval between health checks (default 15s).</summary>
    public TimeSpan Interval { get; }

    /// <summary>Timeout for a single health check (default 5s).</summary>
    public TimeSpan Timeout { get; }

    /// <summary>Delay before the first health check (default 5s).</summary>
    public TimeSpan InitialDelay { get; }

    /// <summary>Number of consecutive failures before marking unhealthy (default 1).</summary>
    public int FailureThreshold { get; }

    /// <summary>Number of consecutive successes before marking healthy (default 1).</summary>
    public int SuccessThreshold { get; }

    private CheckConfig(Builder builder)
    {
        Interval = builder.IntervalValue;
        Timeout = builder.TimeoutValue;
        InitialDelay = builder.InitialDelayValue;
        FailureThreshold = builder.FailureThresholdValue;
        SuccessThreshold = builder.SuccessThresholdValue;
    }

    /// <summary>Creates a new builder for constructing a <see cref="CheckConfig"/>.</summary>
    public static Builder CreateBuilder() => new();

    /// <summary>Creates a <see cref="CheckConfig"/> with default values.</summary>
    public static CheckConfig Defaults() => CreateBuilder().Build();

    /// <summary>
    /// Fluent builder for constructing a <see cref="CheckConfig"/> instance.
    /// </summary>
    public sealed class Builder
    {
        internal TimeSpan IntervalValue = DefaultInterval;
        internal TimeSpan TimeoutValue = DefaultTimeout;
        internal TimeSpan InitialDelayValue = DefaultInitialDelay;
        internal int FailureThresholdValue = DefaultFailureThreshold;
        internal int SuccessThresholdValue = DefaultSuccessThreshold;

        internal Builder() { }

        /// <summary>Sets the interval between health checks.</summary>
        /// <param name="interval">Interval (1s–10min).</param>
        public Builder WithInterval(TimeSpan interval)
        {
            IntervalValue = interval;
            return this;
        }

        /// <summary>Sets the timeout for a single health check.</summary>
        /// <param name="timeout">Timeout (100ms–30s, must be less than interval).</param>
        public Builder WithTimeout(TimeSpan timeout)
        {
            TimeoutValue = timeout;
            return this;
        }

        /// <summary>Sets the delay before the first health check.</summary>
        /// <param name="initialDelay">Initial delay (0–5min).</param>
        public Builder WithInitialDelay(TimeSpan initialDelay)
        {
            InitialDelayValue = initialDelay;
            return this;
        }

        /// <summary>Sets the number of consecutive failures before marking unhealthy.</summary>
        /// <param name="failureThreshold">Failure threshold (1–10).</param>
        public Builder WithFailureThreshold(int failureThreshold)
        {
            FailureThresholdValue = failureThreshold;
            return this;
        }

        /// <summary>Sets the number of consecutive successes before marking healthy.</summary>
        /// <param name="successThreshold">Success threshold (1–10).</param>
        public Builder WithSuccessThreshold(int successThreshold)
        {
            SuccessThresholdValue = successThreshold;
            return this;
        }

        /// <summary>Validates and builds the <see cref="CheckConfig"/> instance.</summary>
        /// <exception cref="ValidationException">Thrown when validation fails.</exception>
        public CheckConfig Build()
        {
            Validate();
            return new CheckConfig(this);
        }

        private void Validate()
        {
            if (IntervalValue < MinInterval || IntervalValue > MaxInterval)
            {
                throw new ValidationException(
                    $"interval must be between {MinInterval} and {MaxInterval}, got {IntervalValue}");
            }

            if (TimeoutValue < MinTimeout || TimeoutValue > MaxTimeout)
            {
                throw new ValidationException(
                    $"timeout must be between {MinTimeout} and {MaxTimeout}, got {TimeoutValue}");
            }

            if (TimeoutValue >= IntervalValue)
            {
                throw new ValidationException(
                    $"timeout ({TimeoutValue}) must be less than interval ({IntervalValue})");
            }

            if (InitialDelayValue < MinInitialDelay || InitialDelayValue > MaxInitialDelay)
            {
                throw new ValidationException(
                    $"initialDelay must be between {MinInitialDelay} and {MaxInitialDelay}, got {InitialDelayValue}");
            }

            if (FailureThresholdValue < MinThreshold || FailureThresholdValue > MaxThreshold)
            {
                throw new ValidationException(
                    $"failureThreshold must be between {MinThreshold} and {MaxThreshold}, got {FailureThresholdValue}");
            }

            if (SuccessThresholdValue < MinThreshold || SuccessThresholdValue > MaxThreshold)
            {
                throw new ValidationException(
                    $"successThreshold must be between {MinThreshold} and {MaxThreshold}, got {SuccessThresholdValue}");
            }
        }
    }
}
