package biz.kryukov.dev.dephealth;

/**
 * DNS resolution failure.
 */
public class CheckDnsException extends CheckException {

    public CheckDnsException(String message) {
        super(message, StatusCategory.DNS_ERROR, "dns_error");
    }

    public CheckDnsException(String message, Throwable cause) {
        super(message, cause, StatusCategory.DNS_ERROR, "dns_error");
    }
}
