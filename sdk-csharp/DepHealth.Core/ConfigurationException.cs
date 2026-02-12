namespace DepHealth;

/// <summary>
/// Configuration error (URL parsing, connection string, etc.).
/// </summary>
public class ConfigurationException : Exception
{
    public ConfigurationException(string message) : base(message) { }
}
