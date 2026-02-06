package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import redis.clients.jedis.Jedis;
import redis.clients.jedis.JedisPool;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class RedisHealthCheckerTest {

    @Mock
    private JedisPool jedisPool;
    @Mock
    private Jedis jedis;

    @Test
    void type() {
        assertEquals(DependencyType.REDIS, RedisHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheckWithPool() throws Exception {
        when(jedisPool.getResource()).thenReturn(jedis);
        when(jedis.ping()).thenReturn("PONG");

        RedisHealthChecker checker = RedisHealthChecker.builder()
                .jedisPool(jedisPool)
                .build();

        Endpoint ep = new Endpoint("localhost", "6379");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));

        verify(jedis).ping();
        verify(jedis).close();
    }

    @Test
    void unexpectedPingResponseThrows() throws Exception {
        when(jedisPool.getResource()).thenReturn(jedis);
        when(jedis.ping()).thenReturn("ERROR");

        RedisHealthChecker checker = RedisHealthChecker.builder()
                .jedisPool(jedisPool)
                .build();

        Endpoint ep = new Endpoint("localhost", "6379");
        Exception ex = assertThrows(Exception.class,
                () -> checker.check(ep, Duration.ofSeconds(5)));
        assertTrue(ex.getMessage().contains("ERROR"));
    }
}
