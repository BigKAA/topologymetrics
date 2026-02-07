package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.SQLException;
import java.sql.Statement;
import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
class PostgresHealthCheckerTest {

    @Mock
    private DataSource dataSource;
    @Mock
    private Connection connection;
    @Mock
    private Statement statement;

    @Test
    void type() {
        assertEquals(DependencyType.POSTGRES, PostgresHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheckWithDataSource() throws Exception {
        when(dataSource.getConnection()).thenReturn(connection);
        when(connection.createStatement()).thenReturn(statement);
        when(statement.execute("SELECT 1")).thenReturn(true);

        PostgresHealthChecker checker = PostgresHealthChecker.builder()
                .dataSource(dataSource)
                .build();

        Endpoint ep = new Endpoint("localhost", "5432");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));

        verify(statement).setQueryTimeout(5);
        verify(statement).execute("SELECT 1");
    }

    @Test
    void customQuery() throws Exception {
        when(dataSource.getConnection()).thenReturn(connection);
        when(connection.createStatement()).thenReturn(statement);
        when(statement.execute("SELECT version()")).thenReturn(true);

        PostgresHealthChecker checker = PostgresHealthChecker.builder()
                .dataSource(dataSource)
                .query("SELECT version()")
                .build();

        Endpoint ep = new Endpoint("localhost", "5432");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
        verify(statement).execute("SELECT version()");
    }

    @Test
    void dataSourceFailureThrows() throws Exception {
        when(dataSource.getConnection()).thenThrow(new SQLException("Connection refused"));

        PostgresHealthChecker checker = PostgresHealthChecker.builder()
                .dataSource(dataSource)
                .build();

        Endpoint ep = new Endpoint("localhost", "5432");
        assertThrows(SQLException.class, () -> checker.check(ep, Duration.ofSeconds(5)));
    }
}
