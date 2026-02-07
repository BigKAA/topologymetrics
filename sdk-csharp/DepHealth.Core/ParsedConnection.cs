namespace DepHealth;

/// <summary>
/// Результат парсинга URL/connection string: хост, порт, тип зависимости.
/// </summary>
public sealed record ParsedConnection(string Host, string Port, DependencyType Type)
{
    public override string ToString() => $"{Type.Label()}://{Host}:{Port}";
}
