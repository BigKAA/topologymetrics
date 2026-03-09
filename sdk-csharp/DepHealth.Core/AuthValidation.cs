namespace DepHealth;

/// <summary>
/// Shared validation for authentication configuration conflicts.
/// </summary>
internal static class AuthValidation
{
    /// <summary>
    /// Validates that at most one authentication method is configured.
    /// </summary>
    /// <param name="headersOrMetadata">HTTP headers or gRPC metadata dictionary.</param>
    /// <param name="authKeyName">The key name for the auth header (e.g. "Authorization" or "authorization").</param>
    /// <param name="bearerToken">Optional Bearer token.</param>
    /// <param name="basicAuthUsername">Optional Basic auth username.</param>
    /// <param name="basicAuthPassword">Optional Basic auth password.</param>
    /// <param name="context">Context string for the error message (e.g. "header" or "metadata").</param>
    internal static void ValidateNoConflicts(
        IDictionary<string, string>? headersOrMetadata,
        string authKeyName,
        string? bearerToken,
        string? basicAuthUsername,
        string? basicAuthPassword,
        string context = "header")
    {
        var methods = 0;

        if (!string.IsNullOrEmpty(bearerToken))
        {
            methods++;
        }

        if (!string.IsNullOrEmpty(basicAuthUsername) || !string.IsNullOrEmpty(basicAuthPassword))
        {
            methods++;
        }

        if (headersOrMetadata is not null
            && headersOrMetadata.Keys.Any(key => key.Equals(authKeyName, StringComparison.OrdinalIgnoreCase)))
        {
            methods++;
        }

        if (methods > 1)
        {
            throw new ValidationException(
                $"conflicting auth methods: specify only one of bearerToken, basicAuth, or {authKeyName} {context}");
        }
    }
}
