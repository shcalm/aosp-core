/*
 * Copyright (C) 2018 The Android Open Source Project
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

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertThrows;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import dalvik.annotation.compat.VersionCodes;
import dalvik.system.VMRuntime;

import java.lang.reflect.Method;
import java.util.Arrays;
import javax.net.ssl.SSLParameters;
import org.junit.Assume;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class SSLParametersTest {

  @Test
  public void applicationProtocols() {
    SSLParameters params = new SSLParameters();
    try {
      params.setApplicationProtocols(null);
      fail();
    } catch (IllegalArgumentException expected) {
    }
    try {
      params.setApplicationProtocols(new String[] {""});
      fail();
    } catch (IllegalArgumentException expected) {
    }
    try {
      params.setApplicationProtocols(new String[] {null});
      fail();
    } catch (IllegalArgumentException expected) {
    }
    try {
      params.setApplicationProtocols(new String[] {"h2", ""});
      fail();
    } catch (IllegalArgumentException expected) {
    }
    try {
      params.setApplicationProtocols(new String[] {"h2", null});
      fail();
    } catch (IllegalArgumentException expected) {
    }

    // Test that both setApplicationProtocols and getApplicationProtocols clone arrays properly
    String[] protocols = new String[] {"h2"};
    params.setApplicationProtocols(protocols);
    assertTrue(Arrays.equals(new String[] {"h2"}, params.getApplicationProtocols()));
    protocols[0] = "bad";
    assertTrue(Arrays.equals(new String[] {"h2"}, params.getApplicationProtocols()));
    protocols = params.getApplicationProtocols();
    protocols[0] = "bad";
    assertTrue(Arrays.equals(new String[] {"h2"}, params.getApplicationProtocols()));
  }

  @Test
  public void getSetUseCipherSuitesOrder() {
    SSLParameters params = new SSLParameters();
    // Default should be false
    assertFalse(params.getUseCipherSuitesOrder());
    params.setUseCipherSuitesOrder(true);
    assertTrue(params.getUseCipherSuitesOrder());
    params.setUseCipherSuitesOrder(false);
    assertFalse(params.getUseCipherSuitesOrder());
  }

  @Test
  public void test_getSetNamedGroups() {
    assumeSdkC();
    SSLParameters params = new SSLParameters();

    assertTrue(params.getNamedGroups() == null);

    String[] groups = new String[] {"x25519", "secp256r1"};
    params.setNamedGroups(groups);
    assertTrue(Arrays.equals(new String[] {"x25519", "secp256r1"}, params.getNamedGroups()));

    groups[0] = "modified";
    assertTrue(Arrays.equals(new String[] {"x25519", "secp256r1"}, params.getNamedGroups()));

    String[] returnedGroups = params.getNamedGroups();
    returnedGroups[0] = "modified";
    assertTrue(Arrays.equals(new String[] {"x25519", "secp256r1"}, params.getNamedGroups()));

    params.setNamedGroups(null);
    assertTrue(params.getNamedGroups() == null);

    params.setNamedGroups(new String[0]);
    assertTrue(params.getNamedGroups().length == 0);
  }

  @Test
  public void test_setNamedGroups_invalidInputs() {
    assumeSdkC();
    SSLParameters params = new SSLParameters();

    assertThrows(IllegalArgumentException.class, () -> {
      params.setNamedGroups(new String[] {"x25519", null});
    });

    assertThrows(IllegalArgumentException.class, () -> {
      params.setNamedGroups(new String[] {" "});
    });

    assertThrows(IllegalArgumentException.class, () -> {
      params.setNamedGroups(new String[] {"x25519", ""});
    });

    assertThrows(IllegalArgumentException.class, () -> {
      params.setNamedGroups(new String[] {"x25519", "secp256r1", "x25519"});
    });
  }

  private static void assumeSdkC() {
    Assume.assumeTrue(isVmRuntimeGetSdkVersionAvailable());
    Assume.assumeTrue(VMRuntime.getSdkVersion() >= VersionCodes.CINNAMON_BUN);
  }

  private static boolean isVmRuntimeGetSdkVersionAvailable() {
    try {
      Class<?> clazz = Class.forName("dalvik.system.VMRuntime");
      Method m = clazz.getDeclaredMethod("getSdkVersion");
      return m != null;
    } catch (ClassNotFoundException | NoSuchMethodException e) {
      return false;
    }
  }
}
