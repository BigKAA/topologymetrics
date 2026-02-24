package biz.kryukov.dev.dephealth;

/**
 * Thrown when an endpoint is not found during dynamic update operations.
 */
public class EndpointNotFoundException extends DepHealthException {

    public EndpointNotFoundException(String depName, String host, String port) {
        super("Endpoint not found: " + depName + ":" + host + ":" + port);
    }
}
