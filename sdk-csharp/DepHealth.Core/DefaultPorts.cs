namespace DepHealth;

/// <summary>
/// Default ports for various dependency types and URL schemes.
/// </summary>
public static class DefaultPorts
{
    private static readonly Dictionary<string, string> SchemeToPort = new(StringComparer.OrdinalIgnoreCase)
    {
        ["postgres"] = "5432",
        ["postgresql"] = "5432",
        ["mysql"] = "3306",
        ["redis"] = "6379",
        ["rediss"] = "6379",
        ["amqp"] = "5672",
        ["amqps"] = "5671",
        ["http"] = "80",
        ["https"] = "443",
        ["grpc"] = "443",
        ["kafka"] = "9092",
        ["ldap"] = "389",
        ["ldaps"] = "636"
    };

    private static readonly Dictionary<DependencyType, string> TypeToPort = new()
    {
        [DependencyType.Postgres] = "5432",
        [DependencyType.MySql] = "3306",
        [DependencyType.Redis] = "6379",
        [DependencyType.Amqp] = "5672",
        [DependencyType.Http] = "80",
        [DependencyType.Grpc] = "443",
        [DependencyType.Kafka] = "9092",
        [DependencyType.Ldap] = "389"
    };

    /// <summary>
    /// Returns the default port for a URL scheme.
    /// </summary>
    public static string? ForScheme(string scheme) =>
        SchemeToPort.TryGetValue(scheme, out var port) ? port : null;

    /// <summary>
    /// Returns the default port for a dependency type.
    /// </summary>
    public static string? ForType(DependencyType type) =>
        TypeToPort.TryGetValue(type, out var port) ? port : null;
}
