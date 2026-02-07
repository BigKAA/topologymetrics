namespace DepHealth;

/// <summary>
/// Тип зависимости.
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
    Kafka
}

/// <summary>
/// Расширения для <see cref="DependencyType"/>.
/// </summary>
public static class DependencyTypeExtensions
{
    /// <summary>
    /// Возвращает строковое представление для Prometheus-метки type.
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
        _ => throw new ArgumentOutOfRangeException(nameof(type), type, null)
    };

    /// <summary>
    /// Находит тип по строковому представлению (case-insensitive).
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
        _ => throw new ArgumentException($"Unknown dependency type: {label}", nameof(label))
    };
}
