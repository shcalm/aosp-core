/*
 * Copyright (C) 2010 The Android Open Source Project
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

package libcore.highmemorytest.java.util;

import static org.junit.Assert.assertEquals;

import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;

import dalvik.annotation.compat.VersionCodes;
import libcore.test.annotation.NonMts;
import libcore.test.reasons.NonMtsReasons;

import java.time.Instant;
import java.time.LocalDateTime;
import java.time.ZoneId;
import java.util.Currency;
import java.util.Locale;

/**
 * Runs in a separate java package because loading data in all locales creates a lot of garbage /
 * permanent heap growth in ICU4J and causes gcstress test failing in ART test environment.
 * See http://b/161420453.
 */
@RunWith(Parameterized.class)
public class CurrencyTest {
    @Parameterized.Parameters(name = "{0}")
    public static Locale[] getAllLocales() {
        return Locale.getAvailableLocales();
    }

    @Parameterized.Parameter(0)
    public Locale locale;

    @NonMts(reason = NonMtsReasons.ICU_VERSION_DEPENDENCY, disabledUntilSdk = VersionCodes.CINNAMON_BUN)
    @Test
    public void test_currencyCodeIcuConsistency() {
        String countryCode = locale.getCountry();
        // java.util.Currency.getCurrency is time-sensitive. And Croatia doesn't use Euro until
        // 2023/1/1. https://unicode-org.atlassian.net/browse/CLDR-16061
        // We skip the test until Feb 2023.
        if ("HR".equals(countryCode) &&
                LocalDateTime.of(2023, 2, 1, 0, 0).atZone(ZoneId.of("GMT")).toInstant()
                        .isAfter(Instant.now())) {
            return;
        }
        // https://unicode-org.atlassian.net/browse/CLDR-17274
        // Effective 2025-03-31, Curaçao and Sint Maarten are replacing ANG (Netherlands
        // Antillean Guilder) with XCG (Caribbean Guilder).
        if (("CW".equals(countryCode) || "SX".equals(countryCode)) &&
                LocalDateTime.of(2025, 5, 1, 0, 0).atZone(ZoneId.of("GMT")).toInstant()
                        .isAfter(Instant.now())) {
            return;
        }
        // https://unicode-org.atlassian.net/browse/CLDR-19002
        // Effective 2026-01-01, Bulgaria is replacing BGN (Bulgarian Lev) with EUR (Euro).
        if ("BG".equals(countryCode) &&
                LocalDateTime.of(2026, 1, 1, 0, 0).atZone(ZoneId.of("GMT")).toInstant()
                        .isAfter(Instant.now())) {
            return;
        }
        Currency javaCurrency = getCurrency(locale);
        if (javaCurrency == null) {
            return;
        }
        assertEquals("Currency code is not consistent:" + locale,
                android.icu.util.Currency.getInstance(locale).getCurrencyCode(),
                javaCurrency.getCurrencyCode());
    }

    private static Currency getCurrency(Locale l) {
        try {
            return Currency.getInstance(l);
        } catch (IllegalArgumentException e) {
            // The locale could have no country or does not have currency for other reasons.
            return null;
        }
    }
}
