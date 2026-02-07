namespace DepHealth;

/// <summary>
/// Интерфейс проверки здоровья зависимости.
/// Реализации должны быть thread-safe.
/// </summary>
public interface IHealthChecker
{
    /// <summary>
    /// Выполняет проверку здоровья эндпоинта.
    /// </summary>
    /// <param name="endpoint">Эндпоинт для проверки.</param>
    /// <param name="ct">Токен отмены (используется как таймаут).</param>
    /// <exception cref="Exception">Если зависимость нездорова или проверка не удалась.</exception>
    Task CheckAsync(Endpoint endpoint, CancellationToken ct);

    /// <summary>
    /// Тип зависимости.
    /// </summary>
    DependencyType Type { get; }
}
