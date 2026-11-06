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
package libcore.libcore.util;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import android.platform.test.annotations.RequiresFlagsEnabled;
import android.platform.test.flag.junit.CheckFlagsRule;
import android.platform.test.flag.junit.IFlagsValueProvider;

import com.android.libcore.Flags;

import libcore.junit.util.LibcoreFlagsValueProvider;

import org.junit.Rule;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;

@RunWith(JUnit4.class)
public class AConfigFlagTest {

    @Rule
    public final CheckFlagsRule mCheckFlagsRule =
            LibcoreFlagsValueProvider.createCheckFlagsRule();

    @Test
    @RequiresFlagsEnabled(com.android.libcore.Flags.FLAG_ALWAYS_FALSE_TEST_FLAG)
    public void testFlagThrowException() {
        throw new AssertionError("It should never cause the test to fail.");
    }

    @Test
    public void testKnownFlagValues() {
        IFlagsValueProvider provider = LibcoreFlagsValueProvider.getFlagsValueProvider();
        assertFalse(provider.getBoolean(Flags.FLAG_ALWAYS_FALSE_TEST_FLAG));
        assertTrue(provider.getBoolean(Flags.FLAG_OPENJDK_21_V2_APIS));
        assertTrue(provider.getBoolean(com.android.art.rw.flags.Flags.FLAG_TEST_RW_FLAG));
    }

}
