using System.Collections.ObjectModel;
using System.Text.RegularExpressions;

namespace DepHealth;

/// <summary>
/// Dependency descriptor: name, type, set of endpoints, check configuration. Immutable.
/// </summary>
public sealed partial class Dependency
{
    private const int MaxNameLength = 63;

    [GeneratedRegex("^[a-z][a-z0-9-]*$")]
    private static partial Regex NamePattern();

    /// <summary>Dependency name (lowercase, alphanumeric with hyphens).</summary>
    public string Name { get; }

    /// <summary>Type of the dependency (e.g. Http, Postgres, Redis).</summary>
    public DependencyType Type { get; }

    /// <summary>Whether this dependency is critical for the application.</summary>
    public bool Critical { get; }

    /// <summary>Endpoints to monitor for this dependency.</summary>
    public IReadOnlyList<Endpoint> Endpoints { get; }

    /// <summary>Health check configuration (intervals, timeouts, thresholds).</summary>
    public CheckConfig Config { get; }

    private Dependency(Builder builder)
    {
        Name = builder.NameValue;
        Type = builder.TypeValue;
        Critical = builder.CriticalValue!.Value;
        Endpoints = new ReadOnlyCollection<Endpoint>(new List<Endpoint>(builder.EndpointsValue));
        Config = builder.ConfigValue;
    }

    /// <summary>Converts a boolean to "yes"/"no" string.</summary>
    public static string BoolToYesNo(bool value) => value ? "yes" : "no";

    /// <summary>Creates a new builder for constructing a <see cref="Dependency"/>.</summary>
    /// <param name="name">Dependency name (lowercase, alphanumeric with hyphens, max 63 chars).</param>
    /// <param name="type">Type of the dependency.</param>
    public static Builder CreateBuilder(string name, DependencyType type) => new(name, type);

    /// <summary>
    /// Fluent builder for constructing a <see cref="Dependency"/> instance.
    /// </summary>
    public sealed class Builder
    {
        internal string NameValue;
        internal DependencyType TypeValue;
        internal bool? CriticalValue;
        internal List<Endpoint> EndpointsValue = [];
        internal CheckConfig ConfigValue = CheckConfig.Defaults();

        internal Builder(string name, DependencyType type)
        {
            NameValue = name ?? throw new ArgumentNullException(nameof(name));
            TypeValue = type;
        }

        /// <summary>Sets whether this dependency is critical.</summary>
        /// <param name="critical"><c>true</c> if the dependency is critical for the application.</param>
        public Builder WithCritical(bool critical)
        {
            CriticalValue = critical;
            return this;
        }

        /// <summary>Sets the endpoints to monitor.</summary>
        /// <param name="endpoints">Collection of endpoints.</param>
        public Builder WithEndpoints(IEnumerable<Endpoint> endpoints)
        {
            EndpointsValue = new List<Endpoint>(endpoints);
            return this;
        }

        /// <summary>Sets a single endpoint to monitor.</summary>
        /// <param name="endpoint">The endpoint.</param>
        public Builder WithEndpoint(Endpoint endpoint)
        {
            EndpointsValue = [endpoint];
            return this;
        }

        /// <summary>Sets the health check configuration.</summary>
        /// <param name="config">Check configuration with intervals, timeouts, and thresholds.</param>
        public Builder WithConfig(CheckConfig config)
        {
            ConfigValue = config ?? throw new ArgumentNullException(nameof(config));
            return this;
        }

        /// <summary>Validates and builds the <see cref="Dependency"/> instance.</summary>
        /// <exception cref="ValidationException">Thrown when validation fails.</exception>
        public Dependency Build()
        {
            Validate();
            return new Dependency(this);
        }

        private void Validate()
        {
            if (string.IsNullOrEmpty(NameValue) || NameValue.Length > MaxNameLength)
            {
                throw new ValidationException(
                    $"dependency name must be 1-{MaxNameLength} characters, got '{NameValue}' ({NameValue?.Length ?? 0} chars)");
            }

            if (!NamePattern().IsMatch(NameValue))
            {
                throw new ValidationException(
                    $"dependency name must match ^[a-z][a-z0-9-]*$, got '{NameValue}'");
            }

            if (CriticalValue is null)
            {
                throw new ValidationException(
                    $"dependency '{NameValue}': critical must be explicitly set");
            }

            if (EndpointsValue.Count == 0)
            {
                throw new ValidationException(
                    $"dependency '{NameValue}' must have at least one endpoint");
            }

            foreach (var ep in EndpointsValue)
            {
                Endpoint.ValidateLabels(ep.Labels);
            }
        }
    }
}
