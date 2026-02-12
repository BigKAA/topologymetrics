namespace DepHealth;

/// <summary>
/// Dependency health check configuration. Immutable, created via Builder.
/// </summary>
public sealed class CheckConfig
{
    public static readonly TimeSpan DefaultInterval = TimeSpan.FromSeconds(15);
    public static readonly TimeSpan DefaultTimeout = TimeSpan.FromSeconds(5);
    public static readonly TimeSpan DefaultInitialDelay = TimeSpan.FromSeconds(5);
    public const int DefaultFailureThreshold = 1;
    public const int DefaultSuccessThreshold = 1;

    public static readonly TimeSpan MinInterval = TimeSpan.FromSeconds(1);
    public static readonly TimeSpan MaxInterval = TimeSpan.FromMinutes(10);
    public static readonly TimeSpan MinTimeout = TimeSpan.FromMilliseconds(100);
    public static readonly TimeSpan MaxTimeout = TimeSpan.FromSeconds(30);
    public static readonly TimeSpan MinInitialDelay = TimeSpan.Zero;
    public static readonly TimeSpan MaxInitialDelay = TimeSpan.FromMinutes(5);
    public const int MinThreshold = 1;
    public const int MaxThreshold = 10;

    public TimeSpan Interval { get; }
    public TimeSpan Timeout { get; }
    public TimeSpan InitialDelay { get; }
    public int FailureThreshold { get; }
    public int SuccessThreshold { get; }

    private CheckConfig(Builder builder)
    {
        Interval = builder.IntervalValue;
        Timeout = builder.TimeoutValue;
        InitialDelay = builder.InitialDelayValue;
        FailureThreshold = builder.FailureThresholdValue;
        SuccessThreshold = builder.SuccessThresholdValue;
    }

    public static Builder CreateBuilder() => new();

    public static CheckConfig Defaults() => CreateBuilder().Build();

    public sealed class Builder
    {
        internal TimeSpan IntervalValue = DefaultInterval;
        internal TimeSpan TimeoutValue = DefaultTimeout;
        internal TimeSpan InitialDelayValue = DefaultInitialDelay;
        internal int FailureThresholdValue = DefaultFailureThreshold;
        internal int SuccessThresholdValue = DefaultSuccessThreshold;

        internal Builder() { }

        public Builder WithInterval(TimeSpan interval)
        {
            IntervalValue = interval;
            return this;
        }

        public Builder WithTimeout(TimeSpan timeout)
        {
            TimeoutValue = timeout;
            return this;
        }

        public Builder WithInitialDelay(TimeSpan initialDelay)
        {
            InitialDelayValue = initialDelay;
            return this;
        }

        public Builder WithFailureThreshold(int failureThreshold)
        {
            FailureThresholdValue = failureThreshold;
            return this;
        }

        public Builder WithSuccessThreshold(int successThreshold)
        {
            SuccessThresholdValue = successThreshold;
            return this;
        }

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
