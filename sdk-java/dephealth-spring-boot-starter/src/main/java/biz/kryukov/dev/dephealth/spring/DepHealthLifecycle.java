package biz.kryukov.dev.dephealth.spring;

import biz.kryukov.dev.dephealth.DepHealth;
import org.springframework.context.SmartLifecycle;

/**
 * SmartLifecycle: automatic start/stop of DepHealth on application startup/shutdown.
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
        return Integer.MAX_VALUE; // start after all beans are initialized
    }
}
