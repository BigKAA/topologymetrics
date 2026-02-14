package biz.kryukov.dev.dephealth;

/**
 * Base check exception with status classification.
 *
 * <p>Subclasses provide specific category/detail values for common failure types.
 * The scheduler uses {@link #statusCategory()} and {@link #statusDetail()} to set
 * the {@code app_dependency_status} and {@code app_dependency_status_detail} metrics.</p>
 */
public class CheckException extends Exception {

    private final String statusCategory;
    private final String statusDetail;

    public CheckException(String message, String statusCategory, String statusDetail) {
        super(message);
        this.statusCategory = statusCategory;
        this.statusDetail = statusDetail;
    }

    public CheckException(String message, Throwable cause,
                          String statusCategory, String statusDetail) {
        super(message, cause);
        this.statusCategory = statusCategory;
        this.statusDetail = statusDetail;
    }

    /** Returns the status category for this error. */
    public String statusCategory() {
        return statusCategory;
    }

    /** Returns the detail value for this error. */
    public String statusDetail() {
        return statusDetail;
    }
}
