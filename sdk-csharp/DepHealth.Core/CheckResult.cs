namespace DepHealth;

/// <summary>
/// Classification of a health check outcome: status category and detail.
/// </summary>
public readonly record struct CheckResult(string Category, string Detail)
{
    /// <summary>Successful check result.</summary>
    public static readonly CheckResult Ok = new(StatusCategory.Ok, StatusCategory.Ok);
}
