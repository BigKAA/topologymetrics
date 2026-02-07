package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;

import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;

import java.time.Duration;

/**
 * Redis health checker — PING/PONG через Jedis (pool или standalone).
 */
public final class RedisHealthChecker implements HealthChecker {

    private final String password;
    private final int database;
    private final JedisPool jedisPool;

    private RedisHealthChecker(Builder builder) {
        this.password = builder.password;
        this.database = builder.database;
        this.jedisPool = builder.jedisPool;
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        int timeoutMs = (int) timeout.toMillis();

        if (jedisPool != null) {
            checkWithPool();
        } else {
            checkStandalone(endpoint, timeoutMs);
        }
    }

    private void checkWithPool() throws Exception {
        try (Jedis jedis = jedisPool.getResource()) {
            String result = jedis.ping();
            if (!"PONG".equals(result)) {
                throw new Exception("Redis PING returned: " + result);
            }
        }
    }

    private void checkStandalone(Endpoint endpoint, int timeoutMs) throws Exception {
        try (Jedis jedis = new Jedis(endpoint.host(), endpoint.portAsInt(), timeoutMs)) {
            if (password != null && !password.isEmpty()) {
                jedis.auth(password);
            }
            if (database > 0) {
                jedis.select(database);
            }
            String result = jedis.ping();
            if (!"PONG".equals(result)) {
                throw new Exception("Redis PING returned: " + result);
            }
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.REDIS;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String password;
        private int database;
        private JedisPool jedisPool;

        private Builder() {}

        public Builder password(String password) {
            this.password = password;
            return this;
        }

        public Builder database(int database) {
            this.database = database;
            return this;
        }

        public Builder jedisPool(JedisPool jedisPool) {
            this.jedisPool = jedisPool;
            return this;
        }

        public RedisHealthChecker build() {
            return new RedisHealthChecker(this);
        }
    }
}
