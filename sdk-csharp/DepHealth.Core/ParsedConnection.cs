namespace DepHealth;

/// <summary>
/// Result of URL/connection string parsing: host, port, dependency type.
/// </summary>
public sealed record ParsedConnection(string Host, string Port, DependencyType Type)
{
    public override string ToString() => $"{Type.Label()}://{Host}:{Port}";
}
