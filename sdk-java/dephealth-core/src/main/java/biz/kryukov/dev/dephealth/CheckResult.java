package biz.kryukov.dev.dephealth;

import java.util.Objects;

/**
 * Classification of a health check outcome: status category and detail.
 */
public record CheckResult(String category, String detail) {

    public CheckResult {
        Objects.requireNonNull(category, "category");
        Objects.requireNonNull(detail, "detail");
    }

    /** Successful check result. */
    public static final CheckResult OK = new CheckResult(StatusCategory.OK, StatusCategory.OK);
}
