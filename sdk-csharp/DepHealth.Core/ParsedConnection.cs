namespace DepHealth;

/// <summary>
/// Result of URL/connection string parsing: host, port, dependency type.
/// </summary>
/// <param name="Host">The parsed host name or IP address.</param>
/// <param name="Port">The parsed port number as a string.</param>
/// <param name="Type">The inferred dependency type (e.g. Postgres, Redis, HTTP).</param>
public sealed record ParsedConnection(string Host, string Port, DependencyType Type)
{
    /// <inheritdoc />
    public override string ToString() => $"{Type.Label()}://{Host}:{Port}";
}
