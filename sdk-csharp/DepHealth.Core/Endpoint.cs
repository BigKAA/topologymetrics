using System.Collections.ObjectModel;
using System.Text.RegularExpressions;

namespace DepHealth;

/// <summary>
/// Dependency endpoint (host + port + labels). Immutable.
/// </summary>
public sealed partial class Endpoint : IEquatable<Endpoint>
{
    private static readonly HashSet<string> ReservedLabels = new(StringComparer.Ordinal)
    {
        "name", "group", "dependency", "type", "host", "port", "critical"
    };

    [GeneratedRegex("^[a-zA-Z_][a-zA-Z0-9_]*$")]
    private static partial Regex LabelNamePattern();

    /// <summary>Endpoint hostname or IP address.</summary>
    public string Host { get; }

    /// <summary>Endpoint port number as string.</summary>
    public string Port { get; }

    /// <summary>Custom Prometheus labels for this endpoint.</summary>
    public IReadOnlyDictionary<string, string> Labels { get; }

    /// <summary>Additional metadata key-value pairs.</summary>
    [Obsolete("Use Labels instead")]
    public IReadOnlyDictionary<string, string> Metadata => Labels;

    /// <summary>Creates an endpoint with host and port.</summary>
    /// <param name="host">Hostname or IP address.</param>
    /// <param name="port">Port number as string.</param>
    public Endpoint(string host, string port)
        : this(host, port, new Dictionary<string, string>())
    {
    }

    /// <summary>Creates an endpoint with host, port, and custom labels.</summary>
    /// <param name="host">Hostname or IP address.</param>
    /// <param name="port">Port number as string.</param>
    /// <param name="labels">Custom Prometheus labels.</param>
    public Endpoint(string host, string port, IDictionary<string, string> labels)
    {
        Host = host ?? throw new ArgumentNullException(nameof(host));
        Port = port ?? throw new ArgumentNullException(nameof(port));
        Labels = new ReadOnlyDictionary<string, string>(
            new Dictionary<string, string>(labels ?? new Dictionary<string, string>()));
    }

    /// <summary>Returns the port as an integer.</summary>
    public int PortAsInt() => int.Parse(Port, System.Globalization.CultureInfo.InvariantCulture);

    /// <summary>
    /// Validates that the label name is valid: [a-zA-Z_][a-zA-Z0-9_]* and not reserved.
    /// </summary>
    public static void ValidateLabelName(string name)
    {
        if (string.IsNullOrEmpty(name))
        {
            throw new ValidationException("label name must not be empty");
        }

        if (!LabelNamePattern().IsMatch(name))
        {
            throw new ValidationException(
                $"label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '{name}'");
        }

        if (ReservedLabels.Contains(name))
        {
            throw new ValidationException(
                $"label name '{name}' is reserved");
        }
    }

    /// <summary>
    /// Validates all keys in the labels dictionary.
    /// </summary>
    public static void ValidateLabels(IReadOnlyDictionary<string, string> labels)
    {
        foreach (var key in labels.Keys)
        {
            ValidateLabelName(key);
        }
    }

    /// <inheritdoc />
    public bool Equals(Endpoint? other)
    {
        if (other is null)
        {
            return false;
        }

        return Host == other.Host && Port == other.Port;
    }

    /// <inheritdoc />
    public override bool Equals(object? obj) => Equals(obj as Endpoint);

    /// <inheritdoc />
    public override int GetHashCode() => HashCode.Combine(Host, Port);

    /// <inheritdoc />
    public override string ToString() => $"{Host}:{Port}";
}
