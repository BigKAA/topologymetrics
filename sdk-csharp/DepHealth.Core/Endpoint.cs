using System.Collections.ObjectModel;
using System.Text.RegularExpressions;

namespace DepHealth;

/// <summary>
/// Эндпоинт зависимости (хост + порт + метки). Immutable.
/// </summary>
public sealed partial class Endpoint : IEquatable<Endpoint>
{
    private static readonly HashSet<string> ReservedLabels = new(StringComparer.Ordinal)
    {
        "name", "dependency", "type", "host", "port", "critical"
    };

    [GeneratedRegex("^[a-zA-Z_][a-zA-Z0-9_]*$")]
    private static partial Regex LabelNamePattern();

    public string Host { get; }
    public string Port { get; }
    public IReadOnlyDictionary<string, string> Labels { get; }

    [Obsolete("Use Labels instead")]
    public IReadOnlyDictionary<string, string> Metadata => Labels;

    public Endpoint(string host, string port)
        : this(host, port, new Dictionary<string, string>())
    {
    }

    public Endpoint(string host, string port, IDictionary<string, string> labels)
    {
        Host = host ?? throw new ArgumentNullException(nameof(host));
        Port = port ?? throw new ArgumentNullException(nameof(port));
        Labels = new ReadOnlyDictionary<string, string>(
            new Dictionary<string, string>(labels ?? new Dictionary<string, string>()));
    }

    public int PortAsInt() => int.Parse(Port);

    /// <summary>
    /// Проверяет, что имя метки допустимо: [a-zA-Z_][a-zA-Z0-9_]* и не зарезервировано.
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
    /// Валидирует все ключи в словаре меток.
    /// </summary>
    public static void ValidateLabels(IReadOnlyDictionary<string, string> labels)
    {
        foreach (var key in labels.Keys)
        {
            ValidateLabelName(key);
        }
    }

    public bool Equals(Endpoint? other)
    {
        if (other is null)
        {
            return false;
        }

        return Host == other.Host && Port == other.Port;
    }

    public override bool Equals(object? obj) => Equals(obj as Endpoint);

    public override int GetHashCode() => HashCode.Combine(Host, Port);

    public override string ToString() => $"{Host}:{Port}";
}
