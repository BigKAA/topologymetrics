package com.github.bigkaa.dephealth.checks;

import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;
import com.github.bigkaa.dephealth.HealthChecker;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.Statement;
import java.time.Duration;

/**
 * Postgres health checker — SELECT 1 через JDBC (pool DataSource или standalone).
 */
public final class PostgresHealthChecker implements HealthChecker {

    private static final String DEFAULT_QUERY = "SELECT 1";

    private final String query;
    private final String username;
    private final String password;
    private final String database;
    private final DataSource dataSource;

    private PostgresHealthChecker(Builder builder) {
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
        }
    }

    private void checkStandalone(Endpoint endpoint, int timeoutSec) throws Exception {
        String db = database != null ? database : "";
        String url = "jdbc:postgresql://" + endpoint.host() + ":" + endpoint.port() + "/" + db;

        DriverManager.setLoginTimeout(timeoutSec);
        try (Connection conn = DriverManager.getConnection(url, username, password);
             Statement stmt = conn.createStatement()) {
            stmt.setQueryTimeout(timeoutSec);
            stmt.execute(query);
        }
    }

    @Override
    public DependencyType type() {
        return DependencyType.POSTGRES;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static final class Builder {
        private String query = DEFAULT_QUERY;
        private String username;
        private String password;
        private String database;
        private DataSource dataSource;

        private Builder() {}

        public Builder query(String query) {
            this.query = query;
            return this;
        }

        public Builder username(String username) {
            this.username = username;
            return this;
        }

        public Builder password(String password) {
            this.password = password;
            return this;
        }

        public Builder database(String database) {
            this.database = database;
            return this;
        }

        public Builder dataSource(DataSource dataSource) {
            this.dataSource = dataSource;
            return this;
        }

        public PostgresHealthChecker build() {
            return new PostgresHealthChecker(this);
        }
    }
}
