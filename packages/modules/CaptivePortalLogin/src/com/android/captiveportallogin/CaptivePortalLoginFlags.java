/*
 * Copyright (C) 2024 The Android Open Source Project
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

package com.android.captiveportallogin;

/**
 * Collection of experiment flags used in the CaptivePortalLogin module.
 */
public class CaptivePortalLoginFlags {
    /**
     * Experiment flag to create Android Custom Tabs instead of WebView to display the
     * captive portal when connecting to a network that presents a captive portal.
     */
    public static final String CAPTIVE_PORTAL_CUSTOM_TABS = "captive_portal_custom_tabs";

    /**
     * Experiment flag to use any browser in the system to launch custom tabs temporarily if
     * the default browser doesn't support custom tabs and multi-networking, which is useful
     * for testing.
     */
    public static final String USE_ANY_CUSTOM_TAB_PROVIDER = "use_any_custom_tab_provider";

    /**
     * Experiment flag to use a full custom tab instead of a partial custom tab.
     *
     * <p>Enabled by default. This is only used if {@link #CAPTIVE_PORTAL_CUSTOM_TABS} is enabled,
     * or the custom tab is forced on (for example if the network operator opted in).
     */
    public static final String USE_FULL_CUSTOM_TAB = "use_full_custom_tab";

    /**
     * **TESTING AND DEBUGGING ONLY.**
     *
     * This flag bypasses the MultiNetwork API for Custom Tabs. It is a **debugging
     * tool**, not a feature kill switch.
     *
     * **Purpose:** To verify if browser bugs are linked to the multi-network handling logic.
     *
     * **NOTE:** Setting this flag will break expected network functionality and
     * should only be used in controlled tests (ideally when only one network,
     * like Wi-Fi, is active, and mobile data/Private DNS are disconnected).
     */
    public static final String CAPTIVE_PORTAL_CUSTOM_TABS_WITH_MULTINETWORK_API =
            "captive_portal_custom_tabs_with_multinetwork_api";
}
