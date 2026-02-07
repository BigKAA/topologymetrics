package biz.kryukov.dev.dephealth.parser;

import biz.kryukov.dev.dephealth.ConfigurationException;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.junit.jupiter.params.provider.NullAndEmptySource;
import org.junit.jupiter.params.provider.ValueSource;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class ConfigParserTest {

    @Nested
    class ParseUrlTest {

        @ParameterizedTest
        @CsvSource({
                "postgres://localhost:5432/db, localhost, 5432, POSTGRES",
                "postgresql://db.example.com:5432/mydb, db.example.com, 5432, POSTGRES",
                "mysql://mysql-host:3306/db, mysql-host, 3306, MYSQL",
                "redis://redis-host:6379, redis-host, 6379, REDIS",
                "rediss://redis-host:6380, redis-host, 6380, REDIS",
                "amqp://user:pass@rabbit:5672/vhost, rabbit, 5672, AMQP",
                "amqps://rabbit:5671, rabbit, 5671, AMQP",
                "http://api.example.com:8080/health, api.example.com, 8080, HTTP",
                "https://api.example.com:443, api.example.com, 443, HTTP",
                "grpc://grpc-host:9090, grpc-host, 9090, GRPC"
        })
        void parsesStandardUrls(String url, String host, String port, String typeName) {
            List<ParsedConnection> result = ConfigParser.parseUrl(url);
            assertEquals(1, result.size());
            ParsedConnection pc = result.get(0);
            assertEquals(host, pc.host());
            assertEquals(port, pc.port());
            assertEquals(DependencyType.valueOf(typeName), pc.type());
        }

        @ParameterizedTest
        @CsvSource({
                "postgres://localhost/db, localhost, 5432",
                "mysql://db-host/mydb, db-host, 3306",
                "redis://cache, cache, 6379",
                "http://api.example.com, api.example.com, 80",
                "https://api.example.com, api.example.com, 443",
                "grpc://grpc-host, grpc-host, 443",
                "kafka://broker, broker, 9092"
        })
        void defaultPorts(String url, String host, String expectedPort) {
            List<ParsedConnection> result = ConfigParser.parseUrl(url);
            assertEquals(1, result.size());
            assertEquals(host, result.get(0).host());
            assertEquals(expectedPort, result.get(0).port());
        }

        @Test
        void multiHostKafka() {
            List<ParsedConnection> result =
                    ConfigParser.parseUrl("kafka://broker1:9092,broker2:9093,broker3:9094");
            assertEquals(3, result.size());
            assertEquals("broker1", result.get(0).host());
            assertEquals("9092", result.get(0).port());
            assertEquals("broker2", result.get(1).host());
            assertEquals("9093", result.get(1).port());
            assertEquals("broker3", result.get(2).host());
            assertEquals("9094", result.get(2).port());
            result.forEach(pc -> assertEquals(DependencyType.KAFKA, pc.type()));
        }

        @Test
        void multiHostKafkaDefaultPort() {
            List<ParsedConnection> result =
                    ConfigParser.parseUrl("kafka://broker1,broker2:9093");
            assertEquals(2, result.size());
            assertEquals("9092", result.get(0).port());
            assertEquals("9093", result.get(1).port());
        }

        @Test
        void ipv6Address() {
            List<ParsedConnection> result =
                    ConfigParser.parseUrl("postgres://[::1]:5432/db");
            assertEquals(1, result.size());
            assertEquals("::1", result.get(0).host());
            assertEquals("5432", result.get(0).port());
        }

        @Test
        void urlWithUserInfo() {
            List<ParsedConnection> result =
                    ConfigParser.parseUrl("postgres://user:pass@db-host:5432/db");
            assertEquals("db-host", result.get(0).host());
            assertEquals("5432", result.get(0).port());
        }

        @Test
        void urlWithQueryParams() {
            List<ParsedConnection> result =
                    ConfigParser.parseUrl("postgres://localhost:5432/db?sslmode=require");
            assertEquals("localhost", result.get(0).host());
            assertEquals("5432", result.get(0).port());
        }

        @ParameterizedTest
        @NullAndEmptySource
        @ValueSource(strings = {"   "})
        void emptyUrlThrows(String url) {
            assertThrows(ConfigurationException.class, () -> ConfigParser.parseUrl(url));
        }

        @Test
        void noSchemeThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseUrl("localhost:5432"));
        }

        @Test
        void unsupportedSchemeThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseUrl("ftp://localhost:21"));
        }

        @Test
        void invalidPortThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseUrl("http://localhost:99999"));
        }

        @Test
        void zeroPortThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseUrl("http://localhost:0"));
        }
    }

    @Nested
    class ParseJdbcTest {

        @ParameterizedTest
        @CsvSource({
                "jdbc:postgresql://localhost:5432/db, localhost, 5432, POSTGRES",
                "jdbc:mysql://mysql-host:3306/db, mysql-host, 3306, MYSQL"
        })
        void parsesJdbcUrls(String url, String host, String port, String typeName) {
            List<ParsedConnection> result = ConfigParser.parseJdbc(url);
            assertEquals(1, result.size());
            assertEquals(host, result.get(0).host());
            assertEquals(port, result.get(0).port());
            assertEquals(DependencyType.valueOf(typeName), result.get(0).type());
        }

        @Test
        void jdbcDefaultPort() {
            List<ParsedConnection> result =
                    ConfigParser.parseJdbc("jdbc:postgresql://db-host/mydb");
            assertEquals("5432", result.get(0).port());
        }

        @ParameterizedTest
        @NullAndEmptySource
        void emptyThrows(String url) {
            assertThrows(ConfigurationException.class, () -> ConfigParser.parseJdbc(url));
        }

        @Test
        void noJdbcPrefixThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseJdbc("postgresql://localhost:5432/db"));
        }

        @Test
        void unsupportedSubprotocolThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseJdbc("jdbc:oracle:thin:@localhost:1521:xe"));
        }
    }

    @Nested
    class ParseConnectionStringTest {

        @Test
        void standardFormat() {
            Endpoint ep = ConfigParser.parseConnectionString(
                    "Host=db-host;Port=5432;Database=mydb");
            assertEquals("db-host", ep.host());
            assertEquals("5432", ep.port());
        }

        @Test
        void caseInsensitiveKeys() {
            Endpoint ep = ConfigParser.parseConnectionString(
                    "HOST=db-host;PORT=5432");
            assertEquals("db-host", ep.host());
            assertEquals("5432", ep.port());
        }

        @Test
        void serverKey() {
            Endpoint ep = ConfigParser.parseConnectionString(
                    "Server=db-host;Port=3306");
            assertEquals("db-host", ep.host());
            assertEquals("3306", ep.port());
        }

        @Test
        void sqlServerCommaPort() {
            Endpoint ep = ConfigParser.parseConnectionString(
                    "Server=db-host,1433;Database=mydb");
            assertEquals("db-host", ep.host());
            assertEquals("1433", ep.port());
        }

        @Test
        void hostWithColonPort() {
            Endpoint ep = ConfigParser.parseConnectionString(
                    "Host=db-host:5432;Database=mydb");
            assertEquals("db-host", ep.host());
            assertEquals("5432", ep.port());
        }

        @Test
        void noHostThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseConnectionString("Database=mydb;Port=5432"));
        }

        @Test
        void noPortThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseConnectionString("Host=db-host;Database=mydb"));
        }

        @ParameterizedTest
        @NullAndEmptySource
        void emptyThrows(String connStr) {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseConnectionString(connStr));
        }
    }

    @Nested
    class ParseParamsTest {

        @Test
        void standardParams() {
            Endpoint ep = ConfigParser.parseParams("localhost", "5432");
            assertEquals("localhost", ep.host());
            assertEquals("5432", ep.port());
        }

        @Test
        void ipv6WithBrackets() {
            Endpoint ep = ConfigParser.parseParams("[::1]", "5432");
            assertEquals("::1", ep.host());
            assertEquals("5432", ep.port());
        }

        @Test
        void emptyHostThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseParams("", "5432"));
        }

        @Test
        void nullPortThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseParams("localhost", null));
        }

        @Test
        void invalidPortThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseParams("localhost", "abc"));
        }

        @Test
        void portOutOfRangeThrows() {
            assertThrows(ConfigurationException.class, () ->
                    ConfigParser.parseParams("localhost", "70000"));
        }
    }
}
