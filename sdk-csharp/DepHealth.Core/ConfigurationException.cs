namespace DepHealth;

/// <summary>
/// Ошибка конфигурации (парсинг URL, connection string и т.п.).
/// </summary>
public class ConfigurationException : Exception
{
    public ConfigurationException(string message) : base(message) { }
}
