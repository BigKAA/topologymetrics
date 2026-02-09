using System.Collections.ObjectModel;
using System.Text.RegularExpressions;

namespace DepHealth;

/// <summary>
/// Описание зависимости: имя, тип, набор эндпоинтов, конфигурация проверки. Immutable.
/// </summary>
public sealed partial class Dependency
{
    private const int MaxNameLength = 63;

    [GeneratedRegex("^[a-z][a-z0-9-]*$")]
    private static partial Regex NamePattern();

    public string Name { get; }
    public DependencyType Type { get; }
    public bool Critical { get; }
    public IReadOnlyList<Endpoint> Endpoints { get; }
    public CheckConfig Config { get; }

    private Dependency(Builder builder)
    {
        Name = builder.NameValue;
        Type = builder.TypeValue;
        Critical = builder.CriticalValue!.Value;
        Endpoints = new ReadOnlyCollection<Endpoint>(new List<Endpoint>(builder.EndpointsValue));
        Config = builder.ConfigValue;
    }

    public static string BoolToYesNo(bool value) => value ? "yes" : "no";

    public static Builder CreateBuilder(string name, DependencyType type) => new(name, type);

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

        public Builder WithCritical(bool critical)
        {
            CriticalValue = critical;
            return this;
        }

        public Builder WithEndpoints(IEnumerable<Endpoint> endpoints)
        {
            EndpointsValue = new List<Endpoint>(endpoints);
            return this;
        }

        public Builder WithEndpoint(Endpoint endpoint)
        {
            EndpointsValue = [endpoint];
            return this;
        }

        public Builder WithConfig(CheckConfig config)
        {
            ConfigValue = config ?? throw new ArgumentNullException(nameof(config));
            return this;
        }

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
