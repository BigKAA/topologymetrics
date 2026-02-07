package biz.kryukov.dev.dephealth.checks;

import javax.net.ssl.SSLContext;
import javax.net.ssl.TrustManager;
import javax.net.ssl.X509TrustManager;
import java.security.KeyManagementException;
import java.security.NoSuchAlgorithmException;
import java.security.cert.X509Certificate;

/**
 * SSLContext, который принимает все сертификаты (для tlsSkipVerify).
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

    private static final class TrustAllManager implements X509TrustManager {
        @Override
        public void checkClientTrusted(X509Certificate[] chain, String authType) {
            // принимаем всё
        }

        @Override
        public void checkServerTrusted(X509Certificate[] chain, String authType) {
            // принимаем всё
        }

        @Override
        public X509Certificate[] getAcceptedIssuers() {
            return new X509Certificate[0];
        }
    }
}
