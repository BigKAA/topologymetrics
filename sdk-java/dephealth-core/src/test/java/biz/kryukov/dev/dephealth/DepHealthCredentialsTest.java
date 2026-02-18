package biz.kryukov.dev.dephealth;

import biz.kryukov.dev.dephealth.checks.AmqpHealthChecker;
import biz.kryukov.dev.dephealth.checks.MysqlHealthChecker;
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;
import biz.kryukov.dev.dephealth.scheduler.CheckScheduler;

import io.micrometer.core.instrument.simple.SimpleMeterRegistry;

import org.junit.jupiter.api.Test;

import java.lang.reflect.Field;
import java.util.List;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.fail;

/**
 * Tests for extracting credentials from URL when building DepHealth.
 */
class DepHealthCredentialsTest {

    private final SimpleMeterRegistry registry = new SimpleMeterRegistry();

    @Test
    void postgresUrlCredentials() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .url("postgres://user:pass@host:5432/mydb")
                        .critical(true))
                .build();

        PostgresHealthChecker checker = getChecker(dh, PostgresHealthChecker.class);
        assertEquals("user", getField(checker, "username"));
        assertEquals("pass", getField(checker, "password"));
        assertEquals("mydb", getField(checker, "database"));
    }

    @Test
    void mysqlUrlCredentials() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("mysql-db", DependencyType.MYSQL, d -> d
                        .url("mysql://user:pass@host:3306/mydb")
                        .critical(true))
                .build();

        MysqlHealthChecker checker = getChecker(dh, MysqlHealthChecker.class);
        assertEquals("user", getField(checker, "username"));
        assertEquals("pass", getField(checker, "password"));
        assertEquals("mydb", getField(checker, "database"));
    }

    @Test
    void redisUrlPassword() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("redis-cache", DependencyType.REDIS, d -> d
                        .url("redis://:secret@host:6379/2")
                        .critical(false))
                .build();

        RedisHealthChecker checker = getChecker(dh, RedisHealthChecker.class);
        assertEquals("secret", getField(checker, "password"));
        assertEquals(2, (int) getField(checker, "database"));
    }

    @Test
    void amqpUrlCredentials() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("mq", DependencyType.AMQP, d -> d
                        .url("amqp://user:pass@host:5672/vhost")
                        .critical(false))
                .build();

        AmqpHealthChecker checker = getChecker(dh, AmqpHealthChecker.class);
        assertEquals("user", getField(checker, "username"));
        assertEquals("pass", getField(checker, "password"));
        assertEquals("vhost", getField(checker, "virtualHost"));
    }

    @Test
    void explicitDbParamsOverrideUrl() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .url("postgres://urluser:urlpass@host:5432/urldb")
                        .dbUsername("explicit-user")
                        .dbPassword("explicit-pass")
                        .dbDatabase("explicit-db")
                        .critical(true))
                .build();

        PostgresHealthChecker checker = getChecker(dh, PostgresHealthChecker.class);
        assertEquals("explicit-user", getField(checker, "username"));
        assertEquals("explicit-pass", getField(checker, "password"));
        assertEquals("explicit-db", getField(checker, "database"));
    }

    @Test
    void urlEncodedCredentials() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .url("postgres://user%40dom:p%40ss@host:5432/db")
                        .critical(true))
                .build();

        PostgresHealthChecker checker = getChecker(dh, PostgresHealthChecker.class);
        assertEquals("user@dom", getField(checker, "username"));
        assertEquals("p@ss", getField(checker, "password"));
    }

    @Test
    void amqpDefaultVhost() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("mq", DependencyType.AMQP, d -> d
                        .url("amqp://user:pass@host:5672/")
                        .critical(false))
                .build();

        AmqpHealthChecker checker = getChecker(dh, AmqpHealthChecker.class);
        assertEquals("/", getField(checker, "virtualHost"));
    }

    @Test
    void postgresNoCredentials() throws Exception {
        DepHealth dh = DepHealth.builder("test-app", "test-group", registry)
                .dependency("pg", DependencyType.POSTGRES, d -> d
                        .url("postgres://host:5432/mydb")
                        .critical(true))
                .build();

        PostgresHealthChecker checker = getChecker(dh, PostgresHealthChecker.class);
        assertNull(getField(checker, "username"));
        assertNull(getField(checker, "password"));
        assertEquals("mydb", getField(checker, "database"));
    }

    // --- Helper methods ---

    @SuppressWarnings("unchecked")
    private <T> T getChecker(DepHealth dh, Class<T> checkerType) throws Exception {
        // DepHealth -> scheduler (CheckScheduler)
        Field schedulerField = DepHealth.class.getDeclaredField("scheduler");
        schedulerField.setAccessible(true);
        CheckScheduler scheduler = (CheckScheduler) schedulerField.get(dh);

        // CheckScheduler -> deps (List<ScheduledDep>)
        Field depsField = CheckScheduler.class.getDeclaredField("deps");
        depsField.setAccessible(true);
        List<?> deps = (List<?>) depsField.get(scheduler);

        // Find ScheduledDep with the checker of the required type
        for (Object dep : deps) {
            Field checkerField = dep.getClass().getDeclaredField("checker");
            checkerField.setAccessible(true);
            Object checker = checkerField.get(dep);
            if (checkerType.isInstance(checker)) {
                return checkerType.cast(checker);
            }
        }
        fail("Checker " + checkerType.getSimpleName() + " not found");
        return null; // unreachable
    }

    @SuppressWarnings("unchecked")
    private <T> T getField(Object obj, String fieldName) throws Exception {
        Field field = obj.getClass().getDeclaredField(fieldName);
        field.setAccessible(true);
        return (T) field.get(obj);
    }
}
