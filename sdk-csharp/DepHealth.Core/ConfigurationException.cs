namespace DepHealth;

/// <summary>
/// Configuration error (URL parsing, connection string, etc.).
/// </summary>
public class ConfigurationException : Exception
{
    /// <summary>Creates a new ConfigurationException with the specified message.</summary>
    /// <param name="message">The error message.</param>
    public ConfigurationException(string message) : base(message) { }
}
