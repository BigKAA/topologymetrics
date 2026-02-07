using System.Collections.ObjectModel;

namespace DepHealth;

/// <summary>
/// Эндпоинт зависимости (хост + порт + метаданные). Immutable.
/// </summary>
public sealed class Endpoint : IEquatable<Endpoint>
{
    public string Host { get; }
    public string Port { get; }
    public IReadOnlyDictionary<string, string> Metadata { get; }

    public Endpoint(string host, string port)
        : this(host, port, new Dictionary<string, string>())
    {
    }

    public Endpoint(string host, string port, IDictionary<string, string> metadata)
    {
        Host = host ?? throw new ArgumentNullException(nameof(host));
        Port = port ?? throw new ArgumentNullException(nameof(port));
        Metadata = new ReadOnlyDictionary<string, string>(
            new Dictionary<string, string>(metadata ?? new Dictionary<string, string>()));
    }

    public int PortAsInt() => int.Parse(Port);

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
