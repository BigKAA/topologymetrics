namespace DepHealth;

/// <summary>
/// Dependency type.
/// </summary>
public enum DependencyType
{
    Http,
    Grpc,
    Tcp,
    Postgres,
    MySql,
    Redis,
    Amqp,
    Kafka,
    Ldap
}

/// <summary>
/// Extension methods for <see cref="DependencyType"/>.
/// </summary>
public static class DependencyTypeExtensions
{
    /// <summary>
    /// Returns the string representation for the Prometheus "type" label.
    /// </summary>
    public static string Label(this DependencyType type) => type switch
    {
        DependencyType.Http => "http",
        DependencyType.Grpc => "grpc",
        DependencyType.Tcp => "tcp",
        DependencyType.Postgres => "postgres",
        DependencyType.MySql => "mysql",
        DependencyType.Redis => "redis",
        DependencyType.Amqp => "amqp",
        DependencyType.Kafka => "kafka",
        DependencyType.Ldap => "ldap",
        _ => throw new ArgumentOutOfRangeException(nameof(type), type, null)
    };

    /// <summary>
    /// Resolves the dependency type from its string representation (case-insensitive).
    /// </summary>
    public static DependencyType FromLabel(string label) => label.ToLowerInvariant() switch
    {
        "http" => DependencyType.Http,
        "grpc" => DependencyType.Grpc,
        "tcp" => DependencyType.Tcp,
        "postgres" => DependencyType.Postgres,
        "mysql" => DependencyType.MySql,
        "redis" => DependencyType.Redis,
        "amqp" => DependencyType.Amqp,
        "kafka" => DependencyType.Kafka,
        "ldap" => DependencyType.Ldap,
        _ => throw new ArgumentException($"Unknown dependency type: {label}", nameof(label))
    };
}
