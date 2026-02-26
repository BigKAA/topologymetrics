namespace DepHealth;

/// <summary>
/// Classification of a health check outcome: status category and detail.
/// </summary>
/// <param name="Category">The status category (e.g. "ok", "timeout", "error"). See <see cref="StatusCategory"/>.</param>
/// <param name="Detail">A more specific detail string describing the check outcome.</param>
public readonly record struct CheckResult(string Category, string Detail)
{
    /// <summary>Successful check result.</summary>
    public static readonly CheckResult Ok = new(StatusCategory.Ok, StatusCategory.Ok);
}
