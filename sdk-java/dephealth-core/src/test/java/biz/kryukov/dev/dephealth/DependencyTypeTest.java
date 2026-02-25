package biz.kryukov.dev.dephealth;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;

import static org.junit.jupiter.api.Assertions.*;

class DependencyTypeTest {

    @ParameterizedTest
    @CsvSource({
            "HTTP, http",
            "GRPC, grpc",
            "TCP, tcp",
            "POSTGRES, postgres",
            "MYSQL, mysql",
            "REDIS, redis",
            "AMQP, amqp",
            "KAFKA, kafka",
            "LDAP, ldap"
    })
    void labelReturnsCorrectString(String enumName, String expectedLabel) {
        DependencyType type = DependencyType.valueOf(enumName);
        assertEquals(expectedLabel, type.label());
    }

    @ParameterizedTest
    @CsvSource({
            "http, HTTP",
            "HTTP, HTTP",
            "postgres, POSTGRES",
            "KAFKA, KAFKA",
            "Grpc, GRPC"
    })
    void fromLabelCaseInsensitive(String label, String expectedEnum) {
        assertEquals(DependencyType.valueOf(expectedEnum), DependencyType.fromLabel(label));
    }

    @Test
    void fromLabelUnknownThrows() {
        assertThrows(IllegalArgumentException.class, () -> DependencyType.fromLabel("unknown"));
    }

    @Test
    void allTypesCount() {
        assertEquals(9, DependencyType.values().length);
    }
}
