namespace DepHealth;

/// <summary>
/// Configuration error (URL parsing, connection string, etc.).
/// </summary>
public class ConfigurationException : Exception
{
    /// <summary>
    /// Initializes a new instance of the <see cref="ConfigurationException"/> class.
    /// </summary>
    public ConfigurationException() { }

    /// <summary>Creates a new ConfigurationException with the specified message.</summary>
    /// <param name="message">The error message.</param>
    public ConfigurationException(string message) : base(message) { }

    /// <summary>
    /// Initializes a new instance of the <see cref="ConfigurationException"/> class with a message and inner exception.
    /// </summary>
    /// <param name="message">The error message.</param>
    /// <param name="innerException">The exception that caused the configuration error.</param>
    public ConfigurationException(string message, Exception innerException) : base(message, innerException) { }
}
