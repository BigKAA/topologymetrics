package com.github.bigkaa.dephealth.spring;

import com.github.bigkaa.dephealth.DepHealth;
import org.springframework.context.SmartLifecycle;

/**
 * SmartLifecycle: автоматический start/stop DepHealth при запуске/остановке приложения.
 */
public class DepHealthLifecycle implements SmartLifecycle {

    private final DepHealth depHealth;
    private volatile boolean running;

    public DepHealthLifecycle(DepHealth depHealth) {
        this.depHealth = depHealth;
    }

    @Override
    public void start() {
        depHealth.start();
        running = true;
    }

    @Override
    public void stop() {
        depHealth.stop();
        running = false;
    }

    @Override
    public boolean isRunning() {
        return running;
    }

    @Override
    public int getPhase() {
        return Integer.MAX_VALUE; // запускаемся после всех бинов
    }
}
