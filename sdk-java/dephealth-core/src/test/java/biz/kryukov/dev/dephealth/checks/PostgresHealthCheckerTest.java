package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.SQLException;
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
    private PreparedStatement preparedStatement;

    @Test
    void type() {
        assertEquals(DependencyType.POSTGRES, PostgresHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheckWithDataSource() throws Exception {
        when(dataSource.getConnection()).thenReturn(connection);
        when(connection.prepareStatement("SELECT 1")).thenReturn(preparedStatement);
        when(preparedStatement.execute()).thenReturn(true);

        PostgresHealthChecker checker = PostgresHealthChecker.builder()
                .dataSource(dataSource)
                .build();

        Endpoint ep = new Endpoint("localhost", "5432");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));

        verify(preparedStatement).setQueryTimeout(5);
        verify(preparedStatement).execute();
    }

    @Test
    void customQuery() throws Exception {
        when(dataSource.getConnection()).thenReturn(connection);
        when(connection.prepareStatement("SELECT version()")).thenReturn(preparedStatement);
        when(preparedStatement.execute()).thenReturn(true);

        PostgresHealthChecker checker = PostgresHealthChecker.builder()
                .dataSource(dataSource)
                .query("SELECT version()")
                .build();

        Endpoint ep = new Endpoint("localhost", "5432");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));
        verify(preparedStatement).execute();
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
