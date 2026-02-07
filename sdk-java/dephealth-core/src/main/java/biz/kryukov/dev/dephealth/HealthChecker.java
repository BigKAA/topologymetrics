package biz.kryukov.dev.dephealth;

import java.time.Duration;

/**
 * Интерфейс проверки здоровья зависимости.
 *
 * <p>Реализации должны быть thread-safe.</p>
 */
public interface HealthChecker {

    /**
     * Выполняет проверку здоровья эндпоинта.
     *
     * @param endpoint эндпоинт для проверки
     * @param timeout  максимальное время ожидания
     * @throws Exception если зависимость нездорова или проверка не удалась
     */
    void check(Endpoint endpoint, Duration timeout) throws Exception;

    /** Возвращает тип зависимости. */
    DependencyType type();
}
