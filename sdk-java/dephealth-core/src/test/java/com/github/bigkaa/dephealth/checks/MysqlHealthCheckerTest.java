package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
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
class MysqlHealthCheckerTest {

    @Mock
    private DataSource dataSource;
    @Mock
    private Connection connection;
    @Mock
    private Statement statement;

    @Test
    void type() {
        assertEquals(DependencyType.MYSQL, MysqlHealthChecker.builder().build().type());
    }

    @Test
    void successfulCheckWithDataSource() throws Exception {
        when(dataSource.getConnection()).thenReturn(connection);
        when(connection.createStatement()).thenReturn(statement);
        when(statement.execute("SELECT 1")).thenReturn(true);

        MysqlHealthChecker checker = MysqlHealthChecker.builder()
                .dataSource(dataSource)
                .build();

        Endpoint ep = new Endpoint("localhost", "3306");
        assertDoesNotThrow(() -> checker.check(ep, Duration.ofSeconds(5)));

        verify(statement).setQueryTimeout(5);
        verify(statement).execute("SELECT 1");
    }

    @Test
    void dataSourceFailureThrows() throws Exception {
        when(dataSource.getConnection()).thenThrow(new SQLException("Connection refused"));

        MysqlHealthChecker checker = MysqlHealthChecker.builder()
                .dataSource(dataSource)
                .build();

        Endpoint ep = new Endpoint("localhost", "3306");
        assertThrows(SQLException.class, () -> checker.check(ep, Duration.ofSeconds(5)));
    }
}
