package biz.kryukov.dev.dephealth.checks;

import biz.kryukov.dev.dephealth.CheckAuthException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.HealthChecker;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;
import java.time.Duration;

/**
 * MySQL health checker — SELECT 1 via JDBC (pool DataSource or standalone).
 */
public final class MysqlHealthChecker implements HealthChecker {

    private static final String DEFAULT_QUERY = "SELECT 1";

    private final String query;
    private final String username;
    private final String password;
    private final String database;
    private final DataSource dataSource;

    private MysqlHealthChecker(Builder builder) {
        this.query = builder.query;
        this.username = builder.username;
        this.password = builder.password;
        this.database = builder.database;
        this.dataSource = builder.dataSource;
    }

    @Override
    public void check(Endpoint endpoint, Duration timeout) throws Exception {
        int timeoutSec = Math.max(1, (int) timeout.getSeconds());

        if (dataSource != null) {
            checkWithDataSource(timeoutSec);
        } else {
            checkStandalone(endpoint, timeoutSec);
        }
    }

    private void checkWithDataSource(int timeoutSec) throws Exception {
        try (Connection conn = dataSource.getConnection();
             Statement stmt = conn.createStatement()) {
            stmt.setQueryTimeout(timeoutSec);
            stmt.execute(query);
        } catch (java.sql.SQLException e) {
            throw classifyMysqlError(e);
        }
    }

    private void checkStandalone(Endpoint endpoint, int timeoutSec) throws Exception {
        String db = database != null ? database : "";
        String url = "jdbc:mysql://" + endpoint.host() + ":" + endpoint.port() + "/" + db;

        DriverManager.setLoginTimeout(timeoutSec);
        try (Connection conn = DriverManager.getConnection(url, username, password);
             Statement stmt = conn.createStatement()) {
            stmt.setQueryTimeout(timeoutSec);
            stmt.execute(query);
        } catch (java.sql.SQLException e) {
            throw classifyMysqlError(e);
        }
    }

    private static Exception classifyMysqlError(java.sql.SQLException e) {
        int errorCode = e.getErrorCode();
        String msg = e.getMessage();
        if (errorCode == 1045 || (msg != null && msg.contains("Access denied"))) {
            return new CheckAuthException("MySQL auth error: " + msg, e);
        }
        return e;
    }

    @Override
    public DependencyType type() {
        return DependencyType.MYSQL;
    }

    /** Creates a new builder with default settings. */
    public static Builder builder() {
        return new Builder();
    }

    /** Builder for {@link MysqlHealthChecker}. */
    public static final class Builder {
        private String query = DEFAULT_QUERY;
        private String username;
        private String password;
        private String database;
        private DataSource dataSource;

        private Builder() {}

        /** Sets the health check query (default: {@code SELECT 1}). */
        public Builder query(String query) {
            this.query = query;
            return this;
        }

        /** Sets the database username. */
        public Builder username(String username) {
            this.username = username;
            return this;
        }

        /** Sets the database password. */
        public Builder password(String password) {
            this.password = password;
            return this;
        }

        /** Sets the database name. */
        public Builder database(String database) {
            this.database = database;
            return this;
        }

        /** Sets a connection pool DataSource (preferred over standalone). */
        public Builder dataSource(DataSource dataSource) {
            this.dataSource = dataSource;
            return this;
        }

        /** Builds the checker. */
        public MysqlHealthChecker build() {
            return new MysqlHealthChecker(this);
        }
    }
}
