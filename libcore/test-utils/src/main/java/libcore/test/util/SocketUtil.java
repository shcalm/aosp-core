/*
 * Copyright (C) 2025 The Android Open Source Project
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

package libcore.test.util;

import java.net.SocketException;

import org.junit.Assume;

public final class SocketUtil {

    private SocketUtil() {}

    public static void checkIfNetworkUnavailable(SocketException ex) {
        // If the test environment does not have connectivity, the connect() will fail with
        // ENETUNREACH straight away. In this case, we should not fail the test.
        String msg = ex.getMessage();
        boolean isNetworkUnreachable = (msg != null) &&
            (msg.contains("ENETUNREACH") || msg.contains("Network is unreachable"));
        Assume.assumeFalse(
                "Connection failed due to unavailable connectivity in the test environment",
                isNetworkUnreachable);
    }
}
