/*
 * Copyright (C) 2026 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package libcore.javax.net.ssl;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.fail;

import java.util.Collections;
import java.util.List;
import javax.net.ssl.ExtendedSSLSession;
import javax.net.ssl.SNIServerName;

import org.junit.Assume;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class ExtendedSSLSessionTest {

    @Before
    public void setUp() {
        // TODO(b/481260088): Remove this skip when the API is available.
        try {
            ExtendedSSLSession.class.getMethod("getStatusResponses");
        } catch (NoSuchMethodException e) {
            Assume.assumeNoException("API method getStatusResponses is not available", e);
        }
    }

    @Test
    public void testGetStatusResponses() {
        ExtendedSSLSession session = new TestExtendedSSLSession();
        try {
            session.getStatusResponses();
            fail("Expected UnsupportedOperationException");
        } catch (UnsupportedOperationException expected) {
        }
    }

    @Test
    public void testGetStatusResponses_Override() {
        List<byte[]> expected = Collections.emptyList();
        ExtendedSSLSession session = new TestExtendedSSLSession() {
            @Override
            public List<byte[]> getStatusResponses() {
                return expected;
            }
        };
        assertEquals(expected, session.getStatusResponses());
    }

    private static class TestExtendedSSLSession extends ExtendedSSLSession {
        @Override
        public String[] getLocalSupportedSignatureAlgorithms() {
            return new String[0];
        }

        @Override
        public String[] getPeerSupportedSignatureAlgorithms() {
            return new String[0];
        }

        @Override
        public List<SNIServerName> getRequestedServerNames() {
            return null;
        }

        @Override
        public byte[] getId() {
            return new byte[0];
        }

        @Override
        public javax.net.ssl.SSLSessionContext getSessionContext() {
            return null;
        }

        @Override
        public long getCreationTime() {
            return 0;
        }

        @Override
        public long getLastAccessedTime() {
            return 0;
        }

        @Override
        public void invalidate() {
        }

        @Override
        public boolean isValid() {
            return false;
        }

        @Override
        public void putValue(String name, Object value) {
        }

        @Override
        public Object getValue(String name) {
            return null;
        }

        @Override
        public void removeValue(String name) {
        }

        @Override
        public String[] getValueNames() {
            return new String[0];
        }

        @Override
        public java.security.cert.Certificate[] getPeerCertificates()
                throws javax.net.ssl.SSLPeerUnverifiedException {
            return null;
        }

        @Override
        public java.security.cert.Certificate[] getLocalCertificates() {
            return null;
        }

        @Override
        public javax.security.cert.X509Certificate[] getPeerCertificateChain()
                throws javax.net.ssl.SSLPeerUnverifiedException {
            return null;
        }

        @Override
        public java.security.Principal getPeerPrincipal()
                throws javax.net.ssl.SSLPeerUnverifiedException {
            return null;
        }

        @Override
        public java.security.Principal getLocalPrincipal() {
            return null;
        }

        @Override
        public String getCipherSuite() {
            return null;
        }

        @Override
        public String getProtocol() {
            return null;
        }

        @Override
        public String getPeerHost() {
            return null;
        }

        @Override
        public int getPeerPort() {
            return 0;
        }

        @Override
        public int getPacketBufferSize() {
            return 0;
        }

        @Override
        public int getApplicationBufferSize() {
            return 0;
        }
    }
}
