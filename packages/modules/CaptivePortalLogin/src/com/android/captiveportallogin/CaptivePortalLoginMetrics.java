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

package com.android.captiveportallogin;

import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__APP_EVENT__CAPTIVE_PORTAL_LOGIN_APP_EVENT_DONE;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__APP_EVENT__CAPTIVE_PORTAL_LOGIN_APP_EVENT_STARTED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNKNOWN;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNWANTED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_WANTED_AS_IS;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_CCT;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_RUNNING_ANDROID_R;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW;

import android.os.Process;

import androidx.annotation.IntDef;

import com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog;

import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;

/**
 * Class to record the captive portal related metrics into statsd.
 *
 * This class is not thread-safe, and should always be accessed from the same thread.
 *
 * @hide
 */
public class CaptivePortalLoginMetrics {
    private int mUid;
    private int mCaptivePortalResult;
    private int mCaptivePortalUsingWebviewReason;
    private long mDurationMs;

    public CaptivePortalLoginMetrics() {
        mUid = Process.INVALID_UID;
        mCaptivePortalResult =
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNKNOWN;
        mCaptivePortalUsingWebviewReason =
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN;
    }

    // Define the acceptable values for CaptivePortalResult.
    @IntDef({
        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNKNOWN,
        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNWANTED,
        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_WANTED_AS_IS
        // TODO: add new enum here
    })
    @Retention(RetentionPolicy.SOURCE)
    public @interface PortalResult {}

    // Define the acceptable values for CaptivePortalUsingWebViewReason.
    @IntDef({
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_RUNNING_ANDROID_R,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_CCT,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW,
        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW
        // TODO: add new enum here
    })
    @Retention(RetentionPolicy.SOURCE)
    public @interface UseWebViewReson {}

    /**
     * Reset all metrics members.
     */
    public void reset() {
        mUid = Process.INVALID_UID;
        mCaptivePortalResult =
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNKNOWN;
        mCaptivePortalUsingWebviewReason =
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN;
    }

    /**
     * Set the UID of the app showing the portal.
     */
    public void setUid(final int uid) {
        mUid = uid;
    }

    /**
     * Set the specific captive portal result.
     */
    public void setPortalResult(@PortalResult final int result) {
        mCaptivePortalResult = result;
    }

    /**
     * Set the specific reason why captive portal is using WebView.
     */
    public void setReason(@UseWebViewReson final int reason) {
        mCaptivePortalUsingWebviewReason = reason;
    }

    /**
     * Set the duration in milliseconds taken to complete the captive portal login.
     */
    public void setDurationMs(final long durationMs) {
        mDurationMs = durationMs;
    }

    /**
     * Write the CaptivePortalLoginReported proto into statsd.
     */
    public void statsWrite() {
        CaptivePortalLoginStatsLog.write(CAPTIVE_PORTAL_LOGIN_REPORTED,
                mUid,
                mCaptivePortalResult,
                mCaptivePortalUsingWebviewReason,
                CAPTIVE_PORTAL_LOGIN_REPORTED__APP_EVENT__CAPTIVE_PORTAL_LOGIN_APP_EVENT_DONE,
                mDurationMs);
    }

    /**
     * Log the app start event into statsd.
     */
    public void logAppStart() {
        CaptivePortalLoginStatsLog.write(CAPTIVE_PORTAL_LOGIN_REPORTED,
                android.os.Process.INVALID_UID,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNKNOWN,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN,
                CAPTIVE_PORTAL_LOGIN_REPORTED__APP_EVENT__CAPTIVE_PORTAL_LOGIN_APP_EVENT_STARTED,
                0 /* durationMs */);
    }
}
