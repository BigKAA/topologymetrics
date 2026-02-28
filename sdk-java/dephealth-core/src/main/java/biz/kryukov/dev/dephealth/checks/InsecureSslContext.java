package biz.kryukov.dev.dephealth.checks;

import edu.umd.cs.findbugs.annotations.SuppressFBWarnings;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import java.security.KeyManagementException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.X509Certificate;

/**
 * SSLContext that accepts all certificates (for tlsSkipVerify).
 */
final class InsecureSslContext {

    private InsecureSslContext() {}

    static SSLContext create() {
        try {
            SSLContext ctx = SSLContext.getInstance("TLS");
            ctx.init(null, new TrustManager[]{new TrustAllManager()}, null);
            return ctx;
        } catch (NoSuchAlgorithmException | KeyManagementException e) {
            throw new RuntimeException("Failed to create insecure SSL context", e);
        }
    }

    @SuppressFBWarnings(value = "WEAK_TRUST_MANAGER",
            justification = "Intentional for tlsSkipVerify option")
    private static final class TrustAllManager implements X509TrustManager {
        @Override
        public void checkClientTrusted(X509Certificate[] chain, String authType) {
            // accept all
        }

        @Override
        public void checkServerTrusted(X509Certificate[] chain, String authType) {
            // accept all
        }

        @Override
        public X509Certificate[] getAcceptedIssuers() {
            return new X509Certificate[0];
        }
    }
}
