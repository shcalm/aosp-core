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

package libcore.junit.util;

import android.aconfig.Aconfig;
import android.platform.test.flag.junit.CheckFlagsRule;
import android.platform.test.flag.junit.DeviceFlagsValueProvider;
import android.platform.test.flag.junit.IFlagsValueProvider;
import android.platform.test.flag.util.Flag;
import android.platform.test.flag.util.FlagReadException;

import java.io.IOException;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;

/**
 * LibcoreFlagsValueProvider creates an {@link CheckFlagsRule} instance to be used without
 * Android Frameworks in the master-art branch.
 */
public final class LibcoreFlagsValueProvider {
    private LibcoreFlagsValueProvider() {}

    public static CheckFlagsRule createCheckFlagsRule() {
        if (isFrameworksAvailable()) {
            return DeviceFlagsValueProvider.createCheckFlagsRule();
        } else {
            return createLibcoreProvider();
        }
    }

    private static boolean isFrameworksAvailable() {
        try {
            // http://b/443939040 Instrumentation is required for DeviceFlagsValueProvider().
            Class.forName("android.app.Instrumentation");
            return true;
        } catch (ClassNotFoundException e) {
            return false;
        }
    }

    private static CheckFlagsRule createLibcoreProvider() {
        return new CheckFlagsRule(getFlagsValueProvider());
    }

    public static LibcoreCheckFlagsRule getFlagsValueProvider() {
        return LibcoreCheckFlagsRule.sInstance;
    }

    /**
     * Unlike {@link DeviceFlagsValueProvider}, LibcoreCheckFlagsRule only supports the values from
     * the aconfig.pb file and doesn't support reading read-write flag values from the
     * frameworks' AconfigPackage APIs.
     *
     * To print the aconfig.pb content from the core-tests.jar, run
     * `unzip -p out/soong/.intermediates/libcore/core-tests/android_common/withres/core-tests.jar \
     *    aconfig.pb | protoc --decode_raw`
     */

    public static class LibcoreCheckFlagsRule implements IFlagsValueProvider {

        private static final LibcoreCheckFlagsRule sInstance = new LibcoreCheckFlagsRule();

        private final Map<String, Aconfig.parsed_flag> mAconfigPbFlags;

        LibcoreCheckFlagsRule() {
            mAconfigPbFlags = loadAconfigPbFromTestResource();
        }

        @Override
        public void setUp() throws FlagReadException {
            if (mAconfigPbFlags == null) {
                throw new IllegalStateException("aconfig.pb isn't loaded");
            }
        }

        @Override
        public boolean getBoolean(String flag) throws FlagReadException {
            Flag parsedFlag = Flag.createFlag(flag);
            return getFlagBooleanViaAconfigPb(parsedFlag);
        }

        private boolean getFlagBooleanViaAconfigPb(Flag flag) {
            if (!mAconfigPbFlags.containsKey(flag.fullFlagName())) {
                return false;
            }
            Aconfig.parsed_flag staticFlag = mAconfigPbFlags.get(flag.fullFlagName());
            // If the flag is READ_ONLY, read the flag value from the aconfig.pb in test resources
            return staticFlag.getState().equals(Aconfig.flag_state.ENABLED);
        }

        private static Map<String, Aconfig.parsed_flag> loadAconfigPbFromTestResource() {
            try (InputStream stream = LibcoreFlagsValueProvider.class.getClassLoader()
                    .getResourceAsStream("aconfig.pb")) {
                if (stream == null) {
                    return null;
                }
                Map<String, Aconfig.parsed_flag> result = new HashMap<>();
                Aconfig.parsed_flags flags = Aconfig.parsed_flags.parseFrom(stream);
                for (Aconfig.parsed_flag flag : flags.getParsedFlagList()) {
                    String fullFlagName =
                            String.format(
                                    Flag.ACONFIG_FULL_FLAG_FORMAT, flag.getPackage(), flag.getName());
                    result.put(fullFlagName, flag);
                }
                return result;
            } catch (IOException exception) {
                throw new FlagReadException(
                        "ALL_FLAGS", "Failed to read static flags from aconfig.pb", exception);
            }
        }
    }

}
