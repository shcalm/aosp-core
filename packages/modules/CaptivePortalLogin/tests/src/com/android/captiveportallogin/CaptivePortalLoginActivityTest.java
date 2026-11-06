/*
 * Copyright (C) 2020 The Android Open Source Project
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

import static android.Manifest.permission.MANAGE_TEST_NETWORKS;
import static android.app.Activity.RESULT_OK;
import static android.content.Intent.ACTION_CREATE_DOCUMENT;
import static android.content.pm.PackageManager.FEATURE_WIFI;
import static android.net.ConnectivityManager.ACTION_CAPTIVE_PORTAL_SIGN_IN;
import static android.net.ConnectivityManager.EXTRA_CAPTIVE_PORTAL;
import static android.net.ConnectivityManager.EXTRA_CAPTIVE_PORTAL_URL;
import static android.net.ConnectivityManager.EXTRA_CAPTIVE_PORTAL_USER_AGENT;
import static android.net.ConnectivityManager.EXTRA_NETWORK;
import static android.net.NetworkCapabilities.NET_CAPABILITY_VALIDATED;
import static android.view.accessibility.AccessibilityEvent.TYPE_NOTIFICATION_STATE_CHANGED;

import static androidx.browser.customtabs.CustomTabsCallback.NAVIGATION_FINISHED;
import static androidx.browser.customtabs.CustomTabsCallback.NAVIGATION_STARTED;
import static androidx.browser.customtabs.CustomTabsService.ACTION_CUSTOM_TABS_CONNECTION;
import static androidx.lifecycle.Lifecycle.State.DESTROYED;
import static androidx.test.core.app.ApplicationProvider.getApplicationContext;
import static androidx.test.espresso.Espresso.pressBack;
import static androidx.test.espresso.intent.Intents.intended;
import static androidx.test.espresso.intent.Intents.intending;
import static androidx.test.espresso.intent.matcher.IntentMatchers.hasAction;
import static androidx.test.espresso.intent.matcher.IntentMatchers.hasData;
import static androidx.test.espresso.intent.matcher.IntentMatchers.hasExtra;
import static androidx.test.espresso.intent.matcher.IntentMatchers.hasPackage;
import static androidx.test.espresso.intent.matcher.IntentMatchers.isInternal;
import static androidx.test.espresso.web.assertion.WebViewAssertions.webMatches;
import static androidx.test.espresso.web.sugar.Web.onWebView;
import static androidx.test.espresso.web.webdriver.DriverAtoms.findElement;
import static androidx.test.espresso.web.webdriver.DriverAtoms.getText;
import static androidx.test.espresso.web.webdriver.DriverAtoms.webClick;
import static androidx.test.platform.app.InstrumentationRegistry.getInstrumentation;

import static com.android.captiveportallogin.CaptivePortalLoginActivity.EXTRA_USE_CLASSIC_VIEW;
import static com.android.captiveportallogin.CaptivePortalLoginFlags.CAPTIVE_PORTAL_CUSTOM_TABS;
import static com.android.captiveportallogin.DownloadService.DOWNLOAD_ABORTED_REASON_FILE_TOO_LARGE;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW;
import static com.android.testutils.MiscAsserts.assertEventuallyTrue;
import static com.android.testutils.TestNetworkTrackerKt.initTestNetwork;
import static com.android.testutils.TestPermissionUtil.runAsShell;

import static junit.framework.Assert.assertEquals;
import static junit.framework.Assert.assertNotNull;
import static junit.framework.Assert.assertNull;

import static org.hamcrest.CoreMatchers.containsString;
import static org.hamcrest.CoreMatchers.not;
import static org.hamcrest.core.AllOf.allOf;
import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;
import static org.junit.Assume.assumeTrue;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.any;
import static org.mockito.Mockito.anyInt;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.doNothing;
import static org.mockito.Mockito.doReturn;
import static org.mockito.Mockito.inOrder;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.spy;
import static org.mockito.Mockito.timeout;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

import android.app.Instrumentation.ActivityResult;
import android.app.KeyguardManager;
import android.app.UiAutomation;
import android.app.admin.DevicePolicyManager;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.ServiceConnection;
import android.content.pm.PackageManager;
import android.content.pm.PackageManager.NameNotFoundException;
import android.net.CaptivePortal;
import android.net.CaptivePortalData;
import android.net.ConnectivityManager;
import android.net.InetAddresses;
import android.net.LinkAddress;
import android.net.LinkProperties;
import android.net.Network;
import android.net.NetworkCapabilities;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.ConditionVariable;
import android.os.OutcomeReceiver;
import android.os.Parcel;
import android.os.Parcelable;
import android.os.ServiceSpecificException;
import android.system.OsConstants;
import android.util.ArrayMap;
import android.view.accessibility.AccessibilityEvent;
import android.webkit.WebView;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.browser.customtabs.CustomTabsCallback;
import androidx.browser.customtabs.CustomTabsClient;
import androidx.browser.customtabs.CustomTabsIntent;
import androidx.browser.customtabs.CustomTabsServiceConnection;
import androidx.browser.customtabs.CustomTabsSession;
import androidx.test.core.app.ActivityScenario;
import androidx.test.espresso.intent.Intents;
import androidx.test.espresso.web.webdriver.Locator;
import androidx.test.filters.SmallTest;
import androidx.test.uiautomator.By;
import androidx.test.uiautomator.BySelector;
import androidx.test.uiautomator.UiDevice;
import androidx.test.uiautomator.UiObject;
import androidx.test.uiautomator.UiObject2;
import androidx.test.uiautomator.UiSelector;
import androidx.test.uiautomator.Until;

import com.android.net.module.util.CollectionUtils;
import com.android.testutils.ConnectUtil;
import com.android.testutils.DevSdkIgnoreRule;
import com.android.testutils.DevSdkIgnoreRule.IgnoreAfter;
import com.android.testutils.DevSdkIgnoreRule.IgnoreUpTo;
import com.android.testutils.DevSdkIgnoreRunner;
import com.android.testutils.TestNetworkTracker;
import com.android.testutils.com.android.testutils.SetFeatureFlagsRule;
import com.android.testutils.com.android.testutils.SetFeatureFlagsRule.FeatureFlag;

import org.junit.After;
import org.junit.Before;
import org.junit.BeforeClass;
import org.junit.Rule;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.mockito.ArgumentCaptor;
import org.mockito.InOrder;
import org.mockito.MockitoAnnotations;
import org.mockito.Spy;

import java.io.File;
import java.io.IOException;
import java.lang.reflect.Method;
import java.net.ServerSocket;
import java.net.URI;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.Executor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.concurrent.atomic.AtomicReference;
import java.util.function.BooleanSupplier;
import java.util.function.Predicate;

import fi.iki.elonen.NanoHTTPD;

@RunWith(DevSdkIgnoreRunner.class)
@SmallTest
public class CaptivePortalLoginActivityTest {
    private static final String TEST_URL = "http://android.test.com";
    private static final int TEST_NETID = 1234;
    private static final int TEST_CUSTOM_TABS_PROVIDER_UID = 12345;
    private static final String TEST_NC_SSID = "Test NetworkCapabilities SSID";
    private static final String TEST_URL_QUERY = "testquery";
    private static final long TEST_TIMEOUT_MS = 30_000L;
    private static final LinkAddress TEST_LINKADDR = new LinkAddress(
            InetAddresses.parseNumericAddress("2001:db8::8"), 64);
    private static final String TEST_USERAGENT = "Test/42.0 Unit-test";
    private static final String TEST_FRIENDLY_NAME = "Network friendly name";
    private static final String TEST_PORTAL_HOSTNAME = "localhost";
    private static final String TEST_CUSTOM_TABS_PACKAGE_NAME = "com.android.customtabs";
    private static final String TEST_CUSTOM_TABS_CLASS = "com.android.customtabs.MockCustomTabs";
    private static final String TEST_WIFI_CONFIG_TYPE = "application/x-wifi-config";
    private static final String TEST_PRIVATE_DNS_SERVER = "dns.server";
    private static final String TEST_DOWNLOAD_SERVICE_COMPONENT_CLASS_NAME =
            DownloadService.class.getName();
    private static final int CUSTOM_TAB_MENU_ITEM_DO_NOT_USE_THIS_NETWORK = 0;
    private static final int CUSTOM_TAB_MENU_ITEM_USE_THIS_NETWORK = 1;
    private static final int CUSTOM_TAB_MENU_ITEM_USE_CLASSIC_VIEW = 2;

    private final Context mContext = getInstrumentation().getContext();
    private final PackageManager mPackageManager = mContext.getPackageManager();
    private final ConnectUtil mConnectUtil = new ConnectUtil(mContext);
    private ActivityScenario<InstrumentedCaptivePortalLoginActivity> mActivityScenario;
    private Network mNetwork = new Network(TEST_NETID);
    private TestNetworkTracker mTestNetworkTracker;

    private @Spy DownloadService mDownloadService = new DownloadService();

    private static ConnectivityManager sConnectivityManager;
    private static DevicePolicyManager sMockDevicePolicyManager;
    private static DownloadService.DownloadServiceBinder sDownloadServiceBinder;
    private static CustomTabsClient sMockCustomTabsClient;
    private static CaptivePortalLoginMetrics sMockCaptivePortalLoginMetrics;
    private static final ArrayMap<String, Boolean> sFeatureFlags = new ArrayMap<>();
    private static boolean sIsMultiNetworkingSupportedByProvider;
    private static String sMockCustomTabsPackageName;
    private static boolean sTestBrowserNotSupportCustomTab;
    private static ConditionVariable sDestroyedCv;
    private static long sMockCurrentTimeMs = 0;
    @Rule
    public final SetFeatureFlagsRule mSetFeatureFlagsRule =
            new SetFeatureFlagsRule((name, enabled) -> {
                sFeatureFlags.put(name, enabled);
                return null;
            }, (name) -> sFeatureFlags.getOrDefault(name, false));
    @Rule
    public final DevSdkIgnoreRule mIgnoreRule = new DevSdkIgnoreRule();

    public static class InstrumentedCaptivePortalLoginActivity extends CaptivePortalLoginActivity {
        private final CompletableFuture<Intent> mForegroundServiceStart = new CompletableFuture<>();
        // Workaround for https://github.com/android/android-test/issues/1119
        private final CompletableFuture<Intent> mOpenInBrowserIntent =
                new CompletableFuture<>();
        private Intent mServiceIntent = new Intent();
        private final CompletableFuture<ServiceConnection> mDownloadServiceBound =
                new CompletableFuture<>();
        private final CompletableFuture<CustomTabsServiceConnection> mCustomTabsServiceBound =
                new CompletableFuture<>();
        private final ConditionVariable mDlServiceunbindCv = new ConditionVariable(false);

        @Override
        public Object getSystemService(String name) {
            switch (name) {
                case Context.CONNECTIVITY_SERVICE:
                    return sConnectivityManager;
                case Context.DEVICE_POLICY_SERVICE:
                    return sMockDevicePolicyManager;
                default:
                    return super.getSystemService(name);
            }
        }



        @Override
        public ComponentName startForegroundService(Intent service) {
            assertTrue("Multiple foreground services were started during the test",
                    mForegroundServiceStart.complete(service));
            // Do not actually start the service
            return service.getComponent();
        }

        @Override
        public ComponentName startService(Intent service) {
            mServiceIntent = service;
            // Do not actually start the service
            return service.getComponent();
        }

        @Override
        public void onDestroy() {
            super.onDestroy();
            sDestroyedCv.open();
        }

        @Override
        public boolean bindService(Intent service, ServiceConnection conn, int flags) {
            if (service.getAction() == null
                    && service.getComponent().getClassName().equals(
                            TEST_DOWNLOAD_SERVICE_COMPONENT_CLASS_NAME)) {
                assertTrue("Download foreground service was bound multiple times during the test",
                        mDownloadServiceBound.complete(conn));
                getMainThreadHandler().post(() -> conn.onServiceConnected(
                        getInstrumentation().getComponentName(), sDownloadServiceBinder));
                return true;
            } else if (service.getAction().equals(ACTION_CUSTOM_TABS_CONNECTION)
                    && TEST_CUSTOM_TABS_PACKAGE_NAME.equals(service.getPackage())) {
                assertTrue("CustomTabs foreground service was bound multiple times during the test",
                        mCustomTabsServiceBound.complete((CustomTabsServiceConnection) conn));
                getMainThreadHandler().post(() -> {
                    ((CustomTabsServiceConnection) conn).onCustomTabsServiceConnected(
                            getInstrumentation().getComponentName(), sMockCustomTabsClient);
                });
                return true;
            } else {
                return super.bindService(service, conn, flags);
            }
        }

        @Override
        public void unbindService(ServiceConnection conn) {
            mDlServiceunbindCv.open();
        }

        @Override
        Context getContextForCustomTabsBinding() {
            return this;
        }

        @Override
        public void startActivity(Intent intent) {
            if (Intent.ACTION_VIEW.equals(intent.getAction())
                    && intent.getData() != null
                    && intent.getData().getAuthority().startsWith(TEST_PORTAL_HOSTNAME)) {
                mOpenInBrowserIntent.complete(intent);
                return;
            }
            super.startActivity(intent);
        }

        @Override
        String getFileProviderAuthority() {
            // Matches the test provider in the test app manifest
            return "com.android.captiveportallogin.tests.fileprovider";
        }

        @Override
        CaptivePortalLoginMetrics getCaptivePortalLoginMetrics() {
            return sMockCaptivePortalLoginMetrics;
        }

        @Override
        public long getElapsedRealtime() {
            return sMockCurrentTimeMs;
        }

        private boolean isPackageInstalled(String packageName) {
            try {
                getInstrumentation().getContext().getPackageManager().getPackageInfo(packageName,
                        0);
                return true;
            } catch (NameNotFoundException e) {
                return false;
            }
        }

        @Override
        String getDefaultCustomTabsProviderPackage() {
            if (sTestBrowserNotSupportCustomTab) {
                return null;
            }
            if (sMockCustomTabsPackageName != null) {
                return sMockCustomTabsPackageName;
            }
            final String defaultPackage = super.getDefaultCustomTabsProviderPackage();
            if (isPackageInstalled("com.android.chrome")) {
                return "com.android.chrome";
            }
            return defaultPackage;
        }

        @Override
        int getPackageUid(@NonNull final String customTabsProviderPackageName)
                throws NameNotFoundException {
            if (sMockCustomTabsPackageName != null) {
                return TEST_CUSTOM_TABS_PROVIDER_UID;
            }
            return super.getPackageUid(customTabsProviderPackageName);
        }

        @Override
        boolean isMultiNetworkingSupportedByProvider(final String defaultPackageName) {
            if (sTestBrowserNotSupportCustomTab) {
                return super.isMultiNetworkingSupportedByProvider(null);
            }
            // We should always mock this method, otherwise the CCT will not be launched if the
            // chrome version is too old on the DUT and doesn't support multi-networking.
            // TODO: only mock if sMockCustomTabsPackageName is set when the browser is updated
            //  before running tests.
            return sIsMultiNetworkingSupportedByProvider;
        }

        @Override
        boolean isFeatureEnabled(final String name) {
            if (sFeatureFlags.get(name) == null) return false;
            return sFeatureFlags.get(name);
        }

        @Override
        boolean isFeatureNotChickenedOut(String name) {
            if (sFeatureFlags.get(name) == null) return true;
            return sFeatureFlags.get(name);
        }

        @Override
        boolean isCustomTabsEnabledThroughMainlineFlag() {
            return false;
        }

        @Override
        boolean getDeviceConfigPropertyBoolean(final String name, boolean defaultValue) {
            return defaultValue;
        }
    }

    /** Class to replace CaptivePortal to prevent mock object is updated and replaced by parcel. */
    public static class MockCaptivePortal extends CaptivePortal {
        public OutcomeReceiver<Void, ServiceSpecificException> mDelegateUidReceiver;

        int mDismissTimes;
        int mIgnoreTimes;
        int mUseTimes;
        int mReevaluateTimes;
        int mSetDelegateUidTimes;

        private MockCaptivePortal() {
            this(0, 0, 0, 0, 0);
        }
        private MockCaptivePortal(int dismissTimes, int ignoreTimes, int useTimes,
                int reevaluateTimes, int setDelegateUidTimes) {
            super(null);
            mDismissTimes = dismissTimes;
            mIgnoreTimes = ignoreTimes;
            mUseTimes = useTimes;
            mReevaluateTimes = reevaluateTimes;
            mSetDelegateUidTimes = setDelegateUidTimes;
        }

        @Override
        public void reportCaptivePortalDismissed() {
            mDismissTimes++;
        }

        @Override
        public void ignoreNetwork() {
            mIgnoreTimes++;
        }

        @Override
        public void useNetwork() {
            mUseTimes++;
        }

        @Override
        public void reevaluateNetwork() {
            mReevaluateTimes++;
        }

        @Override
        public void setDelegateUid(int uid, Executor executor, OutcomeReceiver receiver) {
            mDelegateUidReceiver = receiver;
            mSetDelegateUidTimes++;
        }

        @Override
        public void writeToParcel(Parcel out, int flags) {
            out.writeInt(mDismissTimes);
            out.writeInt(mIgnoreTimes);
            out.writeInt(mUseTimes);
            out.writeInt(mReevaluateTimes);
        }

        public static final Parcelable.Creator<MockCaptivePortal> CREATOR =
                new Parcelable.Creator<MockCaptivePortal>() {
                @Override
                public MockCaptivePortal createFromParcel(Parcel in) {
                    return new MockCaptivePortal(in.readInt(), in.readInt(), in.readInt(),
                            in.readInt(), in.readInt());
                }

                @Override
                public MockCaptivePortal[] newArray(int size) {
                    return new MockCaptivePortal[size];
                }
        };
    }

    private static BySelector makeMoreButtonSelector() {
        return By.res("com.android.chrome:id/menu_button");
    }

    private static boolean tryClickElement(UiDevice device, BySelector selector) {
        final UiObject2 object = device.findObject(selector);
        if (object == null || !object.isClickable()) {
            return false;
        }
        object.click();
        device.waitForIdle();
        return true;
    }

    @BeforeClass
    public static void setUpClass() throws Exception {
        if (!isChromeDefaultBrowser()) {
            return;
        }
        // TODO: to use --disable-fre and stop using the engagement signals.
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        final Context context = getInstrumentation().getContext();
        final String defaultBrowser = CustomTabsClient.getPackageName(
                getApplicationContext(), null /* packages */);

        // If the default browser is not set there is nothing to do.
        if (defaultBrowser == null) return;

        final Intent launchIntent = context.getPackageManager().getLaunchIntentForPackage(
                defaultBrowser);
        // If the default browser is not launchable, there is nothing to do.
        if (launchIntent == null) return;

        launchIntent.addFlags(Intent.FLAG_ACTIVITY_CLEAR_TASK | Intent.FLAG_ACTIVITY_NEW_TASK);
        context.startActivity(launchIntent);

        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        final BySelector moreButton = makeMoreButtonSelector();
        final List<BySelector> trySelectors = List.of(
                // "Accept & continue" button on the welcome screen
                By.res("com.android.chrome:id/terms_accept"),
                // "No thanks" button on the sign-in screen
                By.res("com.android.chrome:id/negative_button"),
                // Sign in screen
                By.res("com.android.chrome:id/signin_fre_dismiss_button"),
                // Privacy sandbox screen
                By.res("com.android.chrome:id/ack_button").hasAncestor(
                        By.res("com.android.chrome:id/privacy_sandbox_notice_row_view"))
        );
        // Try at least (number of screens) times to dismiss setup screens, by trying to click
        // each of the dismiss buttons
        for (int setupScreenIndex = 0; setupScreenIndex < trySelectors.size(); setupScreenIndex++) {
            device.waitForIdle();
            if (device.findObject(moreButton) != null) {
                break;
            }
            for (BySelector selector : trySelectors) {
                if (tryClickElement(device, selector)) {
                    break;
                }
            }
        }
        // Close the default browser by pressing home button
        device.pressHome();
    }

    @Before
    public void setUp() throws Exception {
        final Context context = getInstrumentation().getContext();
        sConnectivityManager = spy(context.getSystemService(ConnectivityManager.class));
        sMockDevicePolicyManager = mock(DevicePolicyManager.class);
        sDownloadServiceBinder = mock(DownloadService.DownloadServiceBinder.class);
        sMockCustomTabsClient = mock(CustomTabsClient.class);
        sMockCaptivePortalLoginMetrics = mock(CaptivePortalLoginMetrics.class);
        sMockCustomTabsPackageName = null;
        sTestBrowserNotSupportCustomTab = false;
        sDestroyedCv = new ConditionVariable(false);
        sMockCurrentTimeMs = 0;

        MockitoAnnotations.initMocks(this);
        // Use a real (but test) network for the application. The application will pass this
        // network to ConnectivityManager#bindProcessToNetwork, so it needs to be a real, existing
        // network on the device but otherwise has no functional use at all. The http server set up
        // by this test will run on the loopback interface and will not use this test network.
        final UiAutomation automation = getInstrumentation().getUiAutomation();
        automation.adoptShellPermissionIdentity(MANAGE_TEST_NETWORKS);
        try {
            mTestNetworkTracker = initTestNetwork(
                    getInstrumentation().getContext(), TEST_LINKADDR, TEST_TIMEOUT_MS);
        } finally {
            automation.dropShellPermissionIdentity();
        }
        mNetwork = mTestNetworkTracker.getNetwork();
        doReturn(CustomTabsSession.createMockSessionForTesting(new ComponentName(
                TEST_CUSTOM_TABS_PACKAGE_NAME, TEST_CUSTOM_TABS_CLASS))
        ).when(sMockCustomTabsClient).newSession(any());
        configNonVpnNetwork();
        doReturn(CustomTabsSession.createMockSessionForTesting(new ComponentName(
                TEST_CUSTOM_TABS_PACKAGE_NAME, TEST_CUSTOM_TABS_CLASS))
        ).when(sMockCustomTabsClient).newSession(any());
        Intents.init();
    }



    @After
    public void tearDown() throws Exception {
        if (mActivityScenario != null) {
            // Note this may sometimes block for 45 seconds until
            // https://github.com/android/android-test/issues/676 is fixed
            mActivityScenario.close();
            getInstrumentation().getUiAutomation().dropShellPermissionIdentity();
        }
        getInstrumentation().getUiAutomation().setOnAccessibilityEventListener(null);
        getInstrumentation().getContext().getSystemService(ConnectivityManager.class)
                .bindProcessToNetwork(null);
        if (mTestNetworkTracker != null) {
            runAsShell(MANAGE_TEST_NETWORKS, mTestNetworkTracker::teardown);
        }
        Intents.release();
    }

    private Intent makeIntent(Context context, String url, boolean useClassicView,
            Parcelable captivePortal) {
        return makeIntent(context, url, useClassicView, captivePortal, mNetwork);
    }

    private Intent makeIntent(Context context, String url, boolean useClassicView,
            Parcelable captivePortal, Network network) {
        return new Intent(context, InstrumentedCaptivePortalLoginActivity.class)
                .setAction(ACTION_CAPTIVE_PORTAL_SIGN_IN)
                .putExtra(EXTRA_CAPTIVE_PORTAL_URL, url)
                .putExtra(EXTRA_NETWORK, network)
                .putExtra(EXTRA_CAPTIVE_PORTAL_USER_AGENT, TEST_USERAGENT)
                .putExtra(EXTRA_CAPTIVE_PORTAL, captivePortal)
                .putExtra(EXTRA_USE_CLASSIC_VIEW, useClassicView);
    }

    private void initActivity(String url) {
        initActivity(url, false /* useClassicView */);
    }

    private void initActivity(String url, boolean useClassicView) {
        initActivity(url, useClassicView, mNetwork);
    }

    private void initActivity(String url, boolean useClassicView, Network network) {
        initActivity(url, useClassicView, network, true /* notifyCapabilities */);
    }

    private void initActivity(String url, boolean useClassicView, Network network,
            boolean notifyCapabilities) {
        final Context ctx = getInstrumentation().getContext();
        final Intent intent = makeIntent(ctx, url, useClassicView,
                new MockCaptivePortal(), network);
        mActivityScenario = ActivityScenario.launch(intent);
        mActivityScenario.onActivity(activity -> {
            getInstrumentation().getUiAutomation().adoptShellPermissionIdentity(
                    android.Manifest.permission.POST_NOTIFICATIONS);
            ctx.getSystemService(KeyguardManager.class).requestDismissKeyguard(activity, null);
            // Dismiss dialogs or notification shade, so the test can interact with the activity.
            activity.sendBroadcast(new Intent(Intent.ACTION_CLOSE_SYSTEM_DIALOGS));
        });
        if (notifyCapabilities) {
            // Simulate the capabilities update callback as soon as the captive portal is detected.
            // This should generally happen on activity creation as the callback is registered
            // there.
            notifyValidatedChangedNotDone(network,
                    sConnectivityManager.getNetworkCapabilities(network));
        }
        getInstrumentation().waitForIdleSync();
    }

    @Test
    public void testonCreateWithNullCaptivePortal() throws Exception {
        final Context ctx = getInstrumentation().getContext();
        final Intent intent = makeIntent(ctx, TEST_URL, false /* useClassicView */,
                (Bundle) null /* CaptivePortal */);
        try (ActivityScenario<InstrumentedCaptivePortalLoginActivity> scenario =
                     ActivityScenario.launch(intent)) {
            getInstrumentation().waitForIdleSync();
            // Verify that activity calls finish() immediately in its onCreate
            assertEquals(DESTROYED, scenario.getState());
        }
    }

    /**
     * Get the activity MockCaptivePortal.
     *
     * The activity may use a different MockCaptivePortal instance after being recreated, so the
     * MockCaptivePortal should not be kept across possible activity recreation.
     */
    private MockCaptivePortal getCaptivePortal() {
        final AtomicReference<MockCaptivePortal> portalRef = new AtomicReference<>();
        mActivityScenario.onActivity(a -> portalRef.set((MockCaptivePortal) a.mCaptivePortal));
        return portalRef.get();
    }

    private void configNonVpnNetwork() {
        final Network[] networks = new Network[] {new Network(mNetwork)};
        doReturn(networks).when(sConnectivityManager).getAllNetworks();
        final NetworkCapabilities nonVpnCapabilities = new NetworkCapabilities.Builder()
                .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
                .setSsid(TEST_NC_SSID)
                .build();
        doReturn(nonVpnCapabilities).when(sConnectivityManager).getNetworkCapabilities(
                mNetwork);
        doReturn(new LinkProperties()).when(sConnectivityManager).getLinkProperties(mNetwork);
    }

    private void configVpnNetwork() {
        final Network network1 = new Network(TEST_NETID + 1);
        final Network network2 = new Network(TEST_NETID + 2);
        final Network[] networks = new Network[] {network1, network2};
        doReturn(networks).when(sConnectivityManager).getAllNetworks();
        final NetworkCapabilities underlyingCapabilities = new NetworkCapabilities()
                .addTransportType(NetworkCapabilities.TRANSPORT_WIFI);
        final NetworkCapabilities vpnCapabilities = new NetworkCapabilities(underlyingCapabilities)
                .addTransportType(NetworkCapabilities.TRANSPORT_VPN);
        doReturn(underlyingCapabilities).when(sConnectivityManager).getNetworkCapabilities(
                network1);
        doReturn(vpnCapabilities).when(sConnectivityManager).getNetworkCapabilities(network2);
    }

    @Test
    public void testHasVpnNetwork() throws Exception {
        initActivity(TEST_URL);

        // Test non-vpn case.
        configNonVpnNetwork();
        mActivityScenario.onActivity(activity -> assertFalse(activity.hasVpnNetwork()));

        // Test vpn case.
        configVpnNetwork();
        mActivityScenario.onActivity(activity -> assertTrue(activity.hasVpnNetwork()));
    }

    @Test
    public void testIsAlwaysOnVpnEnabled() throws Exception {
        initActivity(TEST_URL);

        doReturn(false).when(sMockDevicePolicyManager).isAlwaysOnVpnLockdownEnabled(any());
        mActivityScenario.onActivity(activity -> assertFalse(activity.isAlwaysOnVpnEnabled()));

        doReturn(true).when(sMockDevicePolicyManager).isAlwaysOnVpnLockdownEnabled(any());
        mActivityScenario.onActivity(activity -> assertTrue(activity.isAlwaysOnVpnEnabled()));
    }

    private void runVpnMsgOrLinkToBrowser(boolean useVpnMatcher) {
        initActivity(TEST_URL);

        // Test non-vpn case.
        configNonVpnNetwork();
        doReturn(false).when(sMockDevicePolicyManager).isAlwaysOnVpnLockdownEnabled(any());
        final String linkMatcher = ".*<a[^>]+href.*";
        mActivityScenario.onActivity(act ->
                assertTrue(act.getWebViewClient().getVpnMsgOrLinkToBrowser().matches(linkMatcher)));

        // Test has vpn case.
        configVpnNetwork();
        final String vpnMatcher = ".*<div.*vpnwarning.*";
        mActivityScenario.onActivity(act ->
                assertTrue(act.getWebViewClient().getVpnMsgOrLinkToBrowser().matches(vpnMatcher)));

        // Test always-on vpn case.
        configNonVpnNetwork();
        doReturn(true).when(sMockDevicePolicyManager).isAlwaysOnVpnLockdownEnabled(any());
        mActivityScenario.onActivity(act ->
                assertTrue(act.getWebViewClient().getVpnMsgOrLinkToBrowser().matches(
                        (useVpnMatcher ? vpnMatcher : linkMatcher))));
    }

    @Test
    public void testVpnMsgOrLinkToBrowser() throws Exception {
        // After Android R(including), DevicePolicyManager allows the caller who has the
        // PERMISSION_MAINLINE_NETWORK_STACK can call the isAlwaysOnVpnLockdownEnabled() to get the
        // status of VPN always-on. So the CaptivePortalLogin could know the status of VPN always-on
        // and show the related warning message to the user.
        runVpnMsgOrLinkToBrowser(true /* useVpnMatcher */);
    }

    private void notifyCapabilitiesChanged(final NetworkCapabilities nc) {
        notifyCapabilitiesChanged(mNetwork, nc);
    }

    private void notifyCapabilitiesChanged(final Network network, final NetworkCapabilities nc) {
        mActivityScenario.onActivity(a -> a.handleCapabilitiesChanged(network, nc));
        getInstrumentation().waitForIdleSync();
    }

    private void notifyLinkPropertiesChanged(final LinkProperties lp) {
        mActivityScenario.onActivity(a -> a.handleLinkPropertiesChanged(mNetwork, lp));
        getInstrumentation().waitForIdleSync();
    }

    private void notifyValidatedChangedAndDismissed(final NetworkCapabilities nc) {
        // Get the MockCaptivePortal before the activity destroys itself
        final MockCaptivePortal cp = getCaptivePortal();
        notifyCapabilitiesChanged(nc);

        assertEquals(cp.mDismissTimes, 1);
        assertEquals(cp.mIgnoreTimes, 0);
        assertEquals(cp.mUseTimes, 0);
    }

    private void verifyNotDone() {
        final MockCaptivePortal cp = getCaptivePortal();
        assertEquals(cp.mDismissTimes, 0);
        assertEquals(cp.mIgnoreTimes, 0);
        assertEquals(cp.mUseTimes, 0);
    }

    private void notifyValidatedChangedNotDone(final NetworkCapabilities nc) {
        notifyValidatedChangedNotDone(mNetwork, nc);
    }

    private void notifyValidatedChangedNotDone(final Network network,
            final NetworkCapabilities nc) {
        notifyCapabilitiesChanged(network, nc);
        verifyNotDone();
    }

    void checkActivityEventuallyDestroyed() throws Exception {
        assertTrue("The activity was not destroyed after " + TEST_TIMEOUT_MS + "ms",
                sDestroyedCv.block(TEST_TIMEOUT_MS));
    }

    @Test
    public void testNetworkCapabilitiesUpdate() throws Exception {
        initActivity(TEST_URL);

        // NetworkCapabilities updates w/o NET_CAPABILITY_VALIDATED.
        final NetworkCapabilities nc = new NetworkCapabilities();
        notifyValidatedChangedNotDone(nc);

        // NetworkCapabilities updates w/ NET_CAPABILITY_VALIDATED.
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);

        checkActivityEventuallyDestroyed();
    }

    private HttpServer runCustomSchemeTest(String linkUri) throws Exception {
        final HttpServer server = new HttpServer();
        server.setResponseBody(TEST_URL_QUERY,
                "<a id='tst_link' href='" + linkUri + "'>Test link</a>");

        server.start();
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        initActivity(server.makeUrl(TEST_URL_QUERY));
        // Mock all external intents
        intending(not(isInternal())).respondWith(new ActivityResult(RESULT_OK, null));

        onWebView().withElement(findElement(Locator.ID, "tst_link")).perform(webClick());
        getInstrumentation().waitForIdleSync();
        return server;
    }

    @Test
    public void testTelScheme() throws Exception {
        final String telUri = "tel:0123456789";
        final HttpServer server = runCustomSchemeTest(telUri);

        final Intent sentIntent = CollectionUtils.findFirst(Intents.getIntents(),
                i -> i.getAction().equals(Intent.ACTION_DIAL));
        assertNotNull(sentIntent);
        assertEquals(Uri.parse(telUri), sentIntent.getData());

        server.stop();
    }

    @Test
    public void testSmsScheme() throws Exception {
        final String telUri = "sms:0123456789";
        final HttpServer server = runCustomSchemeTest(telUri);

        final Intent sentIntent = CollectionUtils.findFirst(Intents.getIntents(),
                i -> i.getAction().equals(Intent.ACTION_SENDTO));
        assertNotNull(sentIntent);
        assertEquals(Uri.parse(telUri), sentIntent.getData());

        server.stop();
    }

    @Test
    public void testUnsupportedScheme() throws Exception {
        final HttpServer server = runCustomSchemeTest("mailto:test@example.com");

        // Mockito intents cannot be used for an intent sent in onDestroy, due to
        // https://github.com/android/android-test/issues/1119
        final CompletableFuture<Intent> viewIntent = new CompletableFuture<>();
        mActivityScenario.onActivity(a -> a.mOpenInBrowserIntent.thenAccept(viewIntent::complete));

        final MockCaptivePortal cp = getCaptivePortal();
        onWebView().withElement(findElement(Locator.ID, "continue_link"))
                .perform(webClick());

        try {
            viewIntent.get(TEST_TIMEOUT_MS, TimeUnit.MILLISECONDS);
        } catch (TimeoutException e) {
            throw new AssertionError(
                    "Open in browser intent not received after " + TEST_TIMEOUT_MS + "ms", e);
        }

        getInstrumentation().waitForIdleSync();
        assertEquals(DESTROYED, mActivityScenario.getState());

        assertEquals(cp.mDismissTimes, 0);
        assertEquals(cp.mIgnoreTimes, 0);
        assertEquals(cp.mUseTimes, 1);
        server.stop();
    }

    @Test
    public void testWebViewBackButton() throws Exception {
        final HttpServer server = new HttpServer();
        final String page1Query = "index.html";
        final String page2Query = "some_other.html";
        final String page1LinkId = "page1_link";
        final String page1LinkText = "Go to some other page";
        final String page2ContentId = "page2_content";
        final String page2ContentText = "This is the other page";
        server.start();

        // page1 contains a link that takes the whole page so clicking anywhere in the webview
        // works, this configuration is necessary to make the UiDevice based click work.
        server.setResponseBody(page1Query,
                "<p><a style=\"position: absolute; display: block; top: 0; left: 0; width: 100%; "
                        + "height: 100%;\" href=\""
                        + page2Query + "\" id='" + page1LinkId
                        + "'>" + page1LinkText + "</a></p>");
        server.setResponseBody(page2Query,
                "<p id='" + page2ContentId + "'>" + page2ContentText + "</p>");
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        initActivity(server.makeUrl(page1Query));
        // Check page 1 is loaded.
        onWebView().withElement(findElement(Locator.ID, page1LinkId))
                .check(webMatches(getText(), containsString(page1LinkText)));

        // Use the actually click event to perform the navigation.
        // This is a hack to work around the issue that webClick() doesn't properly store
        // navigation history in the webview during testing, so we'll simulate an actual user
        // click instead.
        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        device.findObject(By.clazz(WebView.class.getName())).click();

        // Wait for the second page to load by checking for its content.
        assertTrue("Timeout waiting for second page to load", isEventually(() -> {
            try {
                onWebView().withElement(findElement(Locator.ID, page2ContentId))
                        .check(webMatches(getText(), containsString(page2ContentText)));
                return true;
            } catch (Throwable t) {
                return false;
            }
        }, TEST_TIMEOUT_MS));

        // Now that a history entry for page2 exists, pressing back should navigate back.
        pressBack();

        // Check we are back on page 1
        onWebView().withElement(findElement(Locator.ID, page1LinkId))
                .check(webMatches(getText(), containsString(page1LinkText)));
        server.stop();
    }

    @Test
    public void testDownload() throws Exception {
        // Setup the server with a single link on the portal page, leading to a download
        final HttpServer server = new HttpServer();
        final String linkIdDownload = "download";
        final String downloadQuery = "dl";
        final String filename = "testfile.png";
        final String mimetype = "image/png";
        server.setResponseBody(TEST_URL_QUERY,
                "<a id='" + linkIdDownload + "' href='" + downloadQuery + "'>Download</a>");
        server.setResponse(downloadQuery, "This is a test file", mimetype, Collections.singletonMap(
                "Content-Disposition", "attachment; filename=\"" + filename + "\""));
        server.start();

        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        initActivity(server.makeUrl(TEST_URL_QUERY));

        // Create a mock file to be returned when mocking the file chooser
        final Intent mockFileResponse = new Intent();
        final Uri mockFile = Uri.parse("content://mockdata");
        mockFileResponse.setData(mockFile);

        // Mock file chooser and DownloadService intents
        intending(hasAction(ACTION_CREATE_DOCUMENT)).respondWith(
                new ActivityResult(RESULT_OK, mockFileResponse));
        // No intent fired yet
        final Predicate<Intent> createDocumentMatcher = i ->
                i.getAction().equals(ACTION_CREATE_DOCUMENT);
        assertFalse(CollectionUtils.any(Intents.getIntents(), createDocumentMatcher));

        onWebView().withElement(findElement(Locator.ID, linkIdDownload))
                .perform(webClick());

        // The create file intent should be fired when the download starts
        assertEventuallyTrue("Create file intent not received within timeout", TEST_TIMEOUT_MS,
                () -> CollectionUtils.any(Intents.getIntents(), createDocumentMatcher));

        final Intent fileIntent = CollectionUtils.findFirst(
                Intents.getIntents(), createDocumentMatcher);
        assertEquals(mimetype, fileIntent.getType());
        assertEquals(filename, fileIntent.getStringExtra(Intent.EXTRA_TITLE));

        // The download intent should be fired after the create file result is received
        final CompletableFuture<Intent> dlIntentFuture = new CompletableFuture<>();
        mActivityScenario.onActivity(a ->
                a.mForegroundServiceStart.thenAccept(dlIntentFuture::complete));

        final String expectedUrl = server.makeUrl(downloadQuery);
        verify(sDownloadServiceBinder, times(1)).requestDownload(eq(mNetwork),
                any() /* userAgent */, eq(expectedUrl), eq(filename),
                eq(mockFile), any() /* context */, eq(mimetype));

        server.stop();
    }

    @Test
    public void testVenueFriendlyNameTitle() throws Exception {
        final LinkProperties linkProperties = new LinkProperties();
        CaptivePortalData.Builder captivePortalDataBuilder = new CaptivePortalData.Builder();
        // TODO: Use reflection for setVenueFriendlyName until shims are available
        final Class captivePortalDataBuilderClass = captivePortalDataBuilder.getClass();
        final Method setVenueFriendlyNameMethod;

        setVenueFriendlyNameMethod = captivePortalDataBuilderClass.getDeclaredMethod(
                "setVenueFriendlyName", CharSequence.class);

        captivePortalDataBuilder = (CaptivePortalData.Builder)
                setVenueFriendlyNameMethod.invoke(captivePortalDataBuilder, TEST_FRIENDLY_NAME);

        final CaptivePortalData captivePortalData = captivePortalDataBuilder.build();
        linkProperties.setCaptivePortalData(captivePortalData);

        when(sConnectivityManager.getLinkProperties(mNetwork)).thenReturn(linkProperties);
        initActivity("https://tc.example.com/");

        // Verify that the correct venue friendly name is used
        mActivityScenario.onActivity(activity ->
                assertEquals(getInstrumentation().getContext().getString(R.string.action_bar_title,
                        TEST_FRIENDLY_NAME), activity.getActionBar().getTitle()));
    }

    @Test
    public void testWifiSsid() throws Exception {
        initActivity("https://portal.example.com/");
        mActivityScenario.onActivity(activity ->
                assertEquals(activity.getActionBar().getTitle(),
                        getInstrumentation().getContext().getString(R.string.action_bar_title,
                                TEST_NC_SSID)));
    }



    private static boolean isEventually(BooleanSupplier condition, long timeout)
            throws InterruptedException {
        final long start = System.currentTimeMillis();
        do {
            if (condition.getAsBoolean()) return true;
            Thread.sleep(10);
        } while ((System.currentTimeMillis() - start) < timeout);

        return false;
    }

    private static class HttpServer extends NanoHTTPD {
        private final ServerSocket mSocket;
        // Responses per URL query
        private final HashMap<String, MockResponse> mResponses = new HashMap<>();

        private static final class MockResponse {
            private final String mBody;
            private final String mMimetype;
            private final Map<String, String> mHeaders;

            MockResponse(String body, String mimetype, Map<String, String> headers) {
                this.mBody = body;
                this.mMimetype = mimetype;
                this.mHeaders = Collections.unmodifiableMap(new HashMap<>(headers));
            }
        }

        HttpServer() throws IOException {
            this(new ServerSocket());
        }

        private HttpServer(ServerSocket socket) {
            // 0 as port for picking a port automatically
            super(TEST_PORTAL_HOSTNAME, 0);
            mSocket = socket;
        }

        @Override
        public ServerSocketFactory getServerSocketFactory() {
            return () -> mSocket;
        }

        private String makeUrl(String query) {
            return new Uri.Builder()
                    .scheme("http")
                    .encodedAuthority(TEST_PORTAL_HOSTNAME + ":" + mSocket.getLocalPort())
                    .path(query)
                    .build()
                    .toString();
        }

        private void setResponseBody(String query, String body) {
            setResponse(query, body, NanoHTTPD.MIME_HTML, Collections.emptyMap());
        }

        private void setResponse(String query, String body, String mimetype,
                Map<String, String> headers) {
            mResponses.put(query, new MockResponse(body, mimetype, headers));
        }

        @Override
        public Response serve(IHTTPSession session) {
            final String path = URI.create(session.getUri()).getPath().substring(1);
            final MockResponse mockResponse = mResponses.get(path);
            if (mockResponse == null) {
                // Default response is a 404
                return super.serve(session);
            }

            final Response response = newFixedLengthResponse(Response.Status.OK,
                    mockResponse.mMimetype,
                    "<!doctype html>"
                    + "<html>"
                    + "<head><title>Test portal</title></head>"
                    + "<body>" + mockResponse.mBody + "</body>"
                    + "</html>");
            mockResponse.mHeaders.forEach(response::addHeader);
            return response;
        }
    }

    private HttpServer prepareTestDirectlyOpen(String linkIdDownload, String downloadQuery,
            String filename, String mimetype) throws Exception {
        // Setup the server with a single link on the portal page, leading to a download
        final HttpServer server = new HttpServer();
        server.setResponseBody(TEST_URL_QUERY,
                "<a id='" + linkIdDownload + "' href='" + downloadQuery + "'>Download</a>");
        server.setResponse(downloadQuery, "This is a test file", mimetype, Collections.singletonMap(
                "Content-Disposition", "attachment; filename=\"" + filename + "\""));
        server.start();

        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        initActivity(server.makeUrl(TEST_URL_QUERY));
        return server;
    }

    private UiObject getUiSpinner() {
        final String resourceId = getInstrumentation().getContext().getResources()
                .getResourceName(R.id.download_in_progress);
        final UiSelector selector = new UiSelector().resourceId(resourceId);
        return UiDevice.getInstance(getInstrumentation()).findObject(selector);
    }

    private CompletableFuture<Boolean> initToastListener(String expectedMsg) {
        final CompletableFuture<Boolean> messageFuture = new CompletableFuture<>();
        getInstrumentation().getUiAutomation().setOnAccessibilityEventListener(
                new UiAutomation.OnAccessibilityEventListener() {
                    @Override
                    public void onAccessibilityEvent(AccessibilityEvent event) {
                        // Toast is contained in notification state change. Ignore other types.
                        if (event.getEventType() != TYPE_NOTIFICATION_STATE_CHANGED) {
                            return;
                        }

                        // Skip empty text events.
                        if (event.getText().size() == 0) return;

                        final String msg = (String) event.getText().get(0);
                        // The event class name in older SDK platform will be
                        // "android.widget.Toast$TN" instead of "android.widget.Toast".
                        if (event.getClassName().toString().contains(Toast.class.getName())
                                && expectedMsg.equals(msg)) {
                            messageFuture.complete(true);
                        }
                    }
                });
        return messageFuture;
    }

    @Test
    public void testDirectlyOpen_onCreateDeleteFile() throws Exception {
        final String linkIdDownload = "download";
        final HttpServer server = prepareTestDirectlyOpen(linkIdDownload, "dl",
                "test.wificonfig", TEST_WIFI_CONFIG_TYPE);
        final UiObject spinner = getUiSpinner();
        final File downloadPath = new File(getInstrumentation().getContext().getFilesDir(),
                CaptivePortalLoginActivity.FILE_PROVIDER_DOWNLOAD_PATH);

        assertNull(downloadPath.listFiles());

        onWebView().withElement(findElement(Locator.ID, linkIdDownload)).perform(webClick());
        assertTrue(spinner.waitForExists(TEST_TIMEOUT_MS));

        // The download file should be created.
        assertNotNull(downloadPath.listFiles());

        mActivityScenario.recreate();
        // OnCreate should clean the previous created files.
        assertNull(downloadPath.listFiles());
        server.stop();
    }

    @Test
    public void testDirectlyOpen_onDownloadAborted() throws Exception {
        initActivity(TEST_URL);
        final Uri mockFile = Uri.parse("content://mockdata");
        final String expectMsg = getInstrumentation().getContext().getString(
                R.string.file_too_large_cancel_download);
        final CompletableFuture<Boolean> toastFuture = initToastListener(expectMsg);

        mActivityScenario.onActivity(a -> a.mProgressCallback.onDownloadAborted(
                1, DOWNLOAD_ABORTED_REASON_FILE_TOO_LARGE));

        assertTrue(toastFuture.get(TEST_TIMEOUT_MS, TimeUnit.MILLISECONDS));

    }

    @Test
    public void testDirectlyOpen_taskCancelToast() throws Exception {
        final String linkIdDownload = "download";
        final String expectMsg = getInstrumentation().getContext().getString(
                R.string.cancel_pending_downloads);

        final HttpServer server = prepareTestDirectlyOpen(linkIdDownload, "dl",
                "test.wificonfig", TEST_WIFI_CONFIG_TYPE);
        onWebView().withElement(findElement(Locator.ID, linkIdDownload)).perform(webClick());

        final UiObject spinner = getUiSpinner();
        // Expect to see the spinner
        assertTrue(spinner.waitForExists(TEST_TIMEOUT_MS));
        final CompletableFuture<Boolean> toastFuture = initToastListener(expectMsg);
        mActivityScenario.onActivity(a -> a.cancelPendingTask());
        assertTrue(toastFuture.get(TEST_TIMEOUT_MS, TimeUnit.MILLISECONDS));
        server.stop();
    }

    @Test
    public void testDirectlyOpen_cancelPendingTask() throws Exception {
        final String linkIdDownload = "download";
        final String downloadQuery = "dl";
        final String filename = "test.wificonfig";
        final String mimeType = TEST_WIFI_CONFIG_TYPE;
        final int requestId = 123;
        final HttpServer server = prepareTestDirectlyOpen(linkIdDownload, downloadQuery,
                filename, mimeType);

        final UiObject spinner = getUiSpinner();
        // Verify no spinner first.
        assertFalse(spinner.exists());
        doReturn(requestId).when(sDownloadServiceBinder)
                .requestDownload(any(), any(), any(), any(), any(), any(), eq(mimeType));
        onWebView().withElement(findElement(Locator.ID, linkIdDownload)).perform(webClick());
        // Expect to see the spinner
        assertTrue(spinner.waitForExists(TEST_TIMEOUT_MS));
        final ArgumentCaptor<Uri> outfileCaptor = ArgumentCaptor.forClass(Uri.class);
        verify(sDownloadServiceBinder).requestDownload(any(), any(),
                eq(server.makeUrl(downloadQuery)),
                eq(filename),
                outfileCaptor.capture(), any(), eq(mimeType));

        // Cancel pending task.
        mActivityScenario.onActivity(a -> a.cancelPendingTask());
        verify(sDownloadServiceBinder).cancelTask(anyInt());
        // Callback with target task should hide the spinner.
        mActivityScenario.onActivity(a -> a.mProgressCallback.onDownloadComplete(
                outfileCaptor.getValue(), mimeType, requestId, false));
        assertTrue(spinner.waitUntilGone(TEST_TIMEOUT_MS));

        server.stop();
    }

    @Test
    public void testDirectlyOpen_successfullyDownload() throws Exception {
        final String linkIdDownload = "download";
        final String mimeType = TEST_WIFI_CONFIG_TYPE;
        final String filename = "test.wificonfig";
        final Uri mockFile = Uri.parse("content://mockdata");
        final Uri otherFile = Uri.parse("content://otherdata");
        final int downloadId = 123;
        final int otherDownloadId = 456;
        final HttpServer server = prepareTestDirectlyOpen(linkIdDownload, "dl",
                filename, mimeType);
        doReturn(downloadId).when(sDownloadServiceBinder)
                .requestDownload(any(), any(), any(), any(), any(), any(), eq(mimeType));
        // Mock intents trying to actually install the profile
        Intents.intending(not(isInternal())).respondWith(new ActivityResult(RESULT_OK, null));

        final UiObject spinner = getUiSpinner();
        // Verify no spinner first.
        assertFalse(spinner.exists());

        onWebView().withElement(findElement(Locator.ID, linkIdDownload)).perform(webClick());

        // Expect to see the spinner
        assertTrue(spinner.waitForExists(TEST_TIMEOUT_MS));
        // File does not start a create file intent, i.e. no file picker
        assertFalse(CollectionUtils.any(Intents.getIntents(),
                i -> i.getAction().equals(Intent.ACTION_CREATE_DOCUMENT)));
        // Trigger callback with negative result with other undesired other download file.
        mActivityScenario.onActivity(a ->
                a.mProgressCallback.onDownloadComplete(otherFile, mimeType, otherDownloadId,
                        false));
        // Verify spinner is still visible and no intent to open the target file.
        assertTrue(spinner.exists());
        assertFalse(CollectionUtils.any(Intents.getIntents(),
                i -> i.getAction().equals(Intent.ACTION_VIEW)));

        // Trigger callback with positive result
        mActivityScenario.onActivity(a -> a.mProgressCallback.onDownloadComplete(
                mockFile, mimeType, downloadId, true));
        // Verify intent sent to open the target file
        final Intent sentIntent = CollectionUtils.findFirst(Intents.getIntents(),
                i -> i.getAction().equals(Intent.ACTION_VIEW));
        assertNotNull(sentIntent);
        assertEquals(mimeType, sentIntent.getType());
        assertEquals(mockFile, sentIntent.getData());
        assertEquals(Intent.FLAG_GRANT_READ_URI_PERMISSION
                | Intent.FLAG_GRANT_WRITE_URI_PERMISSION, sentIntent.getFlags());
        // Spinner should become invisible.
        assertTrue(spinner.waitUntilGone(TEST_TIMEOUT_MS));

        server.stop();
    }

    private LinkProperties makeLinkPropertiesWithPrivateDns() {
        final LinkProperties linkProperties = new LinkProperties();
        linkProperties.setUsePrivateDns(true);
        linkProperties.setPrivateDnsServerName(TEST_PRIVATE_DNS_SERVER);
        return linkProperties;
    }

    private MockCaptivePortal prepareMockCustomTabs(
            boolean isDelegateUidSetSuccessfully, final LinkProperties linkProperties) {
        return prepareMockCustomTabs(isDelegateUidSetSuccessfully, linkProperties,
                true /* notifyCapabilities */);
    }

    private MockCaptivePortal prepareMockCustomTabs(
            boolean isDelegateUidSetSuccessfully, final LinkProperties linkProperties,
            boolean notifyCapabilities) {
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        sIsMultiNetworkingSupportedByProvider = true;
        doReturn(linkProperties).when(sConnectivityManager).getLinkProperties(mNetwork);

        mockCustomTabsPackage();
        initActivity(TEST_URL, false /* useClassicView */, mNetwork, notifyCapabilities);

        final MockCaptivePortal cp = getCaptivePortal();
        // Wait for setDelegateUid to be called before accessing mDelegateUidReceiver
        assertEventuallyTrue("setDelegateUid was not called", TEST_TIMEOUT_MS,
                () -> cp.mDelegateUidReceiver != null);
        if (isDelegateUidSetSuccessfully) {
            mActivityScenario.onActivity(a -> cp.mDelegateUidReceiver.onResult(null));
        } else {
            mActivityScenario.onActivity(a -> cp.mDelegateUidReceiver.onError(
                    new ServiceSpecificException(OsConstants.EBUSY)));
        }
        return cp;
    }

    private MockCaptivePortal prepareRealCustomTabs() {
        return prepareRealCustomTabs(TEST_URL, mNetwork);
    }

    private MockCaptivePortal prepareRealCustomTabs(String url, Network network) {
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);
        sIsMultiNetworkingSupportedByProvider = true;
        initActivity(url, false /* useClassicView */, network);
        final MockCaptivePortal cp = getCaptivePortal();
        // Wait for setDelegateUid to be called before accessing mDelegateUidReceiver
        assertEventuallyTrue("setDelegateUid was not called", TEST_TIMEOUT_MS,
                () -> cp.mDelegateUidReceiver != null);
        mActivityScenario.onActivity(a -> cp.mDelegateUidReceiver.onResult(null));
        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        device.waitForIdle();
        // To ensure focus is on the web content and not an initial UI element in the
        // custom tab (like a 'minimize' button), simulate a click in the center of the
        // screen. This mimics a user touching the web page.
        // If the focus is not on the web content, then pressing the back button will just bring
        // the focus back to the web content. This will disrupt the test flow that needs to test
        // the back button behavior.
        final int screenWidth = device.getDisplayWidth();
        final int screenHeight = device.getDisplayHeight();
        device.click(screenWidth / 2, screenHeight / 2);
        device.waitForIdle();
        return cp;
    }

    private void runCaptivePortalUsingCustomTabsTest(boolean isDelegateUidSetSuccessfully,
            final LinkProperties linkProperties) throws Exception {
        final MockCaptivePortal cp =
                prepareMockCustomTabs(isDelegateUidSetSuccessfully, linkProperties);

        final ArgumentCaptor<CustomTabsCallback> captor =
                ArgumentCaptor.forClass(CustomTabsCallback.class);
        verify(sMockCustomTabsClient, timeout(TEST_TIMEOUT_MS)).newSession(captor.capture());
        final CustomTabsCallback callback = captor.getValue();
        assertNotNull(callback);
        intended(allOf(
                hasExtra(Intent.EXTRA_REFERRER, Uri.parse("")),
                hasExtra(CustomTabsIntent.EXTRA_NETWORK, mNetwork),
                hasExtra(CustomTabsIntent.EXTRA_OPEN_IN_BROWSER_STATE,
                        CustomTabsIntent.OPEN_IN_BROWSER_STATE_OFF),
                hasData(Uri.parse(TEST_URL))));

        // Send navigation start event, verify if the network will be reevaluated.
        callback.onNavigationEvent(NAVIGATION_STARTED, null /* extras */);
        assertEquals(1, cp.mReevaluateTimes);
        assertEquals(1, cp.mSetDelegateUidTimes);

        // Send navigation finished event, verify that the network is reevaluated again.
        callback.onNavigationEvent(NAVIGATION_FINISHED, null /* extras */);
        assertEquals(2, cp.mReevaluateTimes);
        assertEquals(1, cp.mSetDelegateUidTimes);

        // Simulate a success login with NET_CAPABILITY_VALIDATED callback.
        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);

        // Send another navigation start event, verify that network won't be reevaluated
        // again after login.
        callback.onNavigationEvent(NAVIGATION_STARTED, null /* extras */);
        assertEquals(2, cp.mReevaluateTimes);
        assertEquals(1, cp.mSetDelegateUidTimes);

        // Send another navigation finished event, verify that network won't be reevaluated
        // again after login.
        callback.onNavigationEvent(NAVIGATION_FINISHED, null /* extras */);
        assertEquals(2, cp.mReevaluateTimes);
        assertEquals(1, cp.mSetDelegateUidTimes);
    }

    // only run this test on B and above when the private DNS is on, otherwise, Webview will
    // be launched rather than CCT.
    @Test
    @IgnoreUpTo(Build.VERSION_CODES.VANILLA_ICE_CREAM)
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_privateDnsOn_bypassVpnOrPrivateDnsSuccess()
            throws Exception {
        final LinkProperties lp = makeLinkPropertiesWithPrivateDns();
        runCaptivePortalUsingCustomTabsTest(true /* isDelegateUidSetSuccessfully */, lp);
    }

    // only run this test on B and above when the private DNS is on, otherwise, Webview will
    // be launched rather than CCT.
    @Test
    @IgnoreUpTo(Build.VERSION_CODES.VANILLA_ICE_CREAM)
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_privateDnsOn_bypassVpnOrPrivateDnsFailure()
            throws Exception {
        final LinkProperties lp = makeLinkPropertiesWithPrivateDns();
        runCaptivePortalUsingCustomTabsTest(false /* isDelegateUidSetSuccessfully */, lp);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_privateDnsOff_bypassVpnOrPrivateDnsSuccess()
            throws Exception {
        final LinkProperties lp = new LinkProperties();
        runCaptivePortalUsingCustomTabsTest(true /* isDelegateUidSetSuccessfully */, lp);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_privateDnsOff_bypassVpnOrPrivateDnsFailure()
            throws Exception {
        final LinkProperties lp = new LinkProperties();
        runCaptivePortalUsingCustomTabsTest(false /* isDelegateUidSetSuccessfully */, lp);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_navigationEventBeforeCapabilitiesChanged() {
        // Verify that onNavigationEvent does not crash when called before
        // handleCapabilitiesChanged (mIsPortal is still null).
        final LinkProperties lp = new LinkProperties();
        // Skip the initial capabilities notification so mIsPortal remains null.
        final MockCaptivePortal cp = prepareMockCustomTabs(
                true /* isDelegateUidSetSuccessfully */, lp, false /* notifyCapabilities */);

        final ArgumentCaptor<CustomTabsCallback> captor =
                ArgumentCaptor.forClass(CustomTabsCallback.class);
        verify(sMockCustomTabsClient, timeout(TEST_TIMEOUT_MS)).newSession(captor.capture());
        final CustomTabsCallback callback = captor.getValue();
        assertNotNull(callback);

        callback.onNavigationEvent(NAVIGATION_STARTED, null /* extras */);
        callback.onNavigationEvent(NAVIGATION_FINISHED, null /* extras */);
        assertEquals(0, cp.mReevaluateTimes);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_closeAppWhenClosingCustomTab() throws Exception {
        assumeTrue(isChromeDefaultBrowser());
        prepareRealCustomTabs();
        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        device.waitForIdle();
        device.pressBack();

        checkActivityEventuallyDestroyed();
    }

    // Only run this test on R+ and B-, because on B and above private DNS bypass is supported,
    // on R and below, OutcomeReceiver class is not available yet which is required for Custom
    // tab implementation.
    @Test
    @IgnoreAfter(Build.VERSION_CODES.VANILLA_ICE_CREAM)
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_privateDnsOnWhileCustomTabActive()
            throws Exception {
        final LinkProperties lp = new LinkProperties();
        runCaptivePortalUsingCustomTabsTest(true /* isDelegateUidSetSuccessfully */, lp);

        // Simulate to enable the private DNS while the custom tab is active.
        lp.setUsePrivateDns(true);
        lp.setPrivateDnsServerName("strict.example.com");
        notifyLinkPropertiesChanged(lp);

        // Verify the done(Result.DISMISSED) will be called.
        final MockCaptivePortal cp = getCaptivePortal();
        assertEquals(cp.mDismissTimes, 1);
        assertEquals(cp.mIgnoreTimes, 0);
        assertEquals(cp.mUseTimes, 0);
    }

    private void verifyWebViewInitialization() {
        verify(sConnectivityManager).bindProcessToNetwork(any());
        verify(sMockCustomTabsClient, never()).newSession(any());
        mActivityScenario.onActivity(activity ->
                assertNotNull(activity.findViewById(R.id.webview)));
    }

    private void mockCustomTabsPackage() {
        sMockCustomTabsPackageName = TEST_CUSTOM_TABS_PACKAGE_NAME;
        intending(hasPackage(TEST_CUSTOM_TABS_PACKAGE_NAME))
                .respondWith(new ActivityResult(RESULT_OK, null));
    }

    private void verifyUsingWebViewRatherThanCustomTabs() {
        mockCustomTabsPackage();
        initActivity(TEST_URL);
        verifyWebViewInitialization();
    }



    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = false)
    public void testCaptivePortalUsingCustomTabs_flagOff() throws Exception {
        sIsMultiNetworkingSupportedByProvider = true;
        verifyUsingWebViewRatherThanCustomTabs();
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_multiNetworkNotSupported() throws Exception {
        sIsMultiNetworkingSupportedByProvider = false;
        final LinkProperties linkProperties = new LinkProperties();
        doReturn(linkProperties).when(sConnectivityManager).getLinkProperties(mNetwork);
        verifyUsingWebViewRatherThanCustomTabs();
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalUsingCustomTabs_nullLinkProperties() throws Exception {
        sIsMultiNetworkingSupportedByProvider = true;
        doReturn(null).when(sConnectivityManager).getLinkProperties(mNetwork);
        verifyUsingWebViewRatherThanCustomTabs();
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testLaunchWithNoCustomTabsProvider() throws Exception {
        sTestBrowserNotSupportCustomTab = true;
        initActivity(TEST_URL);
        verifyWebViewInitialization();
    }

    private void findMoreButtonAndClickTheMenuItem(final int menuItemIndex) throws Exception {
        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        device.waitForIdle();
        final BySelector moreButtonSelector = makeMoreButtonSelector();
        assertEventuallyTrue("The Custom Tab more button was not found.", TEST_TIMEOUT_MS,
                () -> device.findObject(moreButtonSelector) != null);
        final UiObject2 moreButton = device.findObject(moreButtonSelector);

        moreButton.click();

        device.waitForIdle();

        // Choosing the menu item by index is not a great solution because there is no
        // guarantee that Chrome will never change the order of the menus. This is only
        // the index of the menu added by the captive portal login activity and this test
        // assumes they are all at the top, which is the current chrome behavior.
        final UiSelector menuItemSelector =
                new UiSelector().className("android.widget.TextView").instance(menuItemIndex);
        final UiObject menuItem = device.findObject(menuItemSelector);
        assertTrue("The expected menu item was not found.",
                menuItem.waitForExists(TEST_TIMEOUT_MS));
        menuItem.click();

        device.waitForIdle();
    }

    private MockCaptivePortal runCustomTabsMenuItemsTest(final int menuItemIndex) throws Exception {
        // Turn on the screen and dismiss the keyguard, allow UI automation to select the button.
        ActivityScenario.launch(RequestDismissKeyguardActivity.class);

        final MockCaptivePortal cp = prepareRealCustomTabs();
        findMoreButtonAndClickTheMenuItem(menuItemIndex);
        return cp;
    }

    private static boolean isChromeDefaultBrowser() {
        final String defaultBrowser = CustomTabsClient.getPackageName(
                getApplicationContext(), null /* packages */);
        return Objects.equals(defaultBrowser, "com.android.chrome");
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCustomTabsMenuItems_clickDoNotUseThisNetwork() throws Exception {
        // This test requires to launch a custom tab provider Chrome and depends on the
        // specific UI of chrome. For those platforms without chrome browser installed
        // such as aosp platforms, then skip this test.
        assumeTrue(isChromeDefaultBrowser());
        final MockCaptivePortal cp =
                runCustomTabsMenuItemsTest(CUSTOM_TAB_MENU_ITEM_DO_NOT_USE_THIS_NETWORK);
        checkActivityEventuallyDestroyed();
        assertEquals(1, cp.mSetDelegateUidTimes);
        assertEquals(1, cp.mIgnoreTimes);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCustomTabsMenuItems_clickUseThisNetwork() throws Exception {
        // This test requires to launch a custom tab provider Chrome and depends on the
        // specific UI of chrome. For those platforms without chrome browser installed
        // such as aosp platforms, then skip this test.
        assumeTrue(isChromeDefaultBrowser());
        final MockCaptivePortal cp =
                runCustomTabsMenuItemsTest(CUSTOM_TAB_MENU_ITEM_USE_THIS_NETWORK);
        checkActivityEventuallyDestroyed();
        assertEquals(1, cp.mSetDelegateUidTimes);
        assertEquals(1, cp.mUseTimes);
    }

    private void verifyIntentExtrasForForceWebview(final Bundle intentExtras) {
        assertEquals(mNetwork, intentExtras.getParcelable(EXTRA_NETWORK));
        assertEquals(TEST_URL, intentExtras.getString(EXTRA_CAPTIVE_PORTAL_URL));
        assertEquals(TEST_USERAGENT,
                intentExtras.getString(EXTRA_CAPTIVE_PORTAL_USER_AGENT));
        assertEquals(true, intentExtras.getBoolean(EXTRA_USE_CLASSIC_VIEW));
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCustomTabsMenuItems_clickUseOldInterface() throws Exception {
        // This test requires to launch a custom tab provider Chrome and depends on the
        // specific UI of chrome. For those platforms without chrome browser installed
        // such as aosp platforms, then skip this test.
        assumeTrue(isChromeDefaultBrowser());
        doNothing().when(sConnectivityManager).startCaptivePortalApp(any(), any());
        runCustomTabsMenuItemsTest(CUSTOM_TAB_MENU_ITEM_USE_CLASSIC_VIEW);
        final ArgumentCaptor<Bundle> bundleCaptor = ArgumentCaptor.forClass(Bundle.class);
        verify(sConnectivityManager, timeout(TEST_TIMEOUT_MS)).startCaptivePortalApp(eq(mNetwork),
                bundleCaptor.capture());
        verifyIntentExtrasForForceWebview(bundleCaptor.getValue());
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testStartActivityWithUseClassicView() throws Exception {
        // Simulate launching a new CaptivePortalLoginActivity with intent including
        // EXTRA_USE_CLASSIC_VIEW, which is set when menu item "Use classic view" is
        // clicked, and verify the Webview should be initialized instead of CCT.
        initActivity(TEST_URL, true /* useClassicView */);
        verifyWebViewInitialization();
    }

    private void verifyCaptivePortalLoginMetrics(final boolean expectWebview, final int result,
            final int reason) {
        // UID isn't set in Webview case.
        if (!expectWebview) {
            verify(sMockCaptivePortalLoginMetrics).setUid(eq(TEST_CUSTOM_TABS_PROVIDER_UID));
        } else {
            verify(sMockCaptivePortalLoginMetrics, never()).setUid(anyInt());
        }
        verify(sMockCaptivePortalLoginMetrics).setPortalResult(eq(result));
        if (reason != CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN) {
            verify(sMockCaptivePortalLoginMetrics).setReason(eq(reason));
        }
    }

    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = false)
    public void testCaptivePortalMetrics_useWebview_dismissed() throws Exception {
        initActivity(TEST_URL);

        // NetworkCapabilities updates w/ NET_CAPABILITY_VALIDATED.
        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        verifyCaptivePortalLoginMetrics(true /* expectWebview */,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalMetrics_useCustomTabs_dismissed() throws Exception {
        final LinkProperties lp = new LinkProperties();
        runCaptivePortalUsingCustomTabsTest(true /* isDelegateUidSetSuccessfully */, lp);

        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        verifyCaptivePortalLoginMetrics(false /* expectWebview */,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_UNKNOWN);

        // Simulate to bring the captive portal activity back to foreground by launching a
        // new activity with the same EXTRA_CAPTIVE_PORTAL.
        final Context ctx = getInstrumentation().getContext();
        final MockCaptivePortal cp = getCaptivePortal();
        final Intent intent = makeIntent(ctx, TEST_URL, false /* useClassicView */, cp);
        mActivityScenario = ActivityScenario.launch(intent);
        getInstrumentation().waitForIdleSync();

        // After bringing the activity to the foreground, trigger another NET_CAPABILITY_VALIDATED
        // callback and verify that the dismiss logic does not run a second time.
        notifyCapabilitiesChanged(nc);
        assertEquals(cp.mDismissTimes, 1); // the portal has been dismissed once before.
        assertEquals(cp.mIgnoreTimes, 0);
        assertEquals(cp.mUseTimes, 0);
    }



    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalMetrics_fallbackToWebview_notSupportMultiNetwork()
            throws Exception {
        sIsMultiNetworkingSupportedByProvider = false;
        verifyUsingWebViewRatherThanCustomTabs();

        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        verifyCaptivePortalLoginMetrics(true /* expectWebview */,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalMetrics_fallbackToWebview_useClassicView()
            throws Exception {
        initActivity(TEST_URL, true /* useClassicView */);
        verifyWebViewInitialization();

        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        verifyCaptivePortalLoginMetrics(true /* expectWebview */,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW);
    }

    // only run this test on V and below when the private DNS is on, otherwise, metrics won't be
    // logged.
    @Test
    @IgnoreAfter(Build.VERSION_CODES.VANILLA_ICE_CREAM)
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCaptivePortalMetrics_fallbackToWebview_privateDnsOnVAndBelow()
            throws Exception {
        sIsMultiNetworkingSupportedByProvider = true;
        final LinkProperties linkProperties = makeLinkPropertiesWithPrivateDns();
        doReturn(linkProperties).when(sConnectivityManager).getLinkProperties(mNetwork);
        verifyUsingWebViewRatherThanCustomTabs();

        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        verifyCaptivePortalLoginMetrics(true /* expectWebview */,
                CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS,
                CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW);
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testSecondaryToolbarView() throws Exception {
        assumeTrue(isChromeDefaultBrowser());
        prepareRealCustomTabs();
        final UiDevice device = UiDevice.getInstance(getInstrumentation());
        final BySelector selector = By.clazz("android.widget.TextView")
                .text(getInstrumentation().getContext().getString(R.string.action_bar_title,
                        TEST_NC_SSID));
        assertEventuallyTrue("The remote view was not found.", TEST_TIMEOUT_MS,
                () -> device.findObject(selector) != null);
    }

    @Test
    public void testLoggingAppStartAndDurationMetrics() throws Exception {
        final InOrder inOrder = inOrder(sMockCaptivePortalLoginMetrics);
        // Start first
        sMockCurrentTimeMs = 100;
        initActivity(TEST_URL, true /* useClassicView */);
        verifyWebViewInitialization();
        inOrder.verify(sMockCaptivePortalLoginMetrics).logAppStart();

        // Recreate
        sMockCurrentTimeMs = 200;
        mActivityScenario.recreate();
        inOrder.verify(sMockCaptivePortalLoginMetrics, never()).logAppStart();

        // Dismiss
        sMockCurrentTimeMs = 300;
        final NetworkCapabilities nc = new NetworkCapabilities();
        nc.setCapability(NET_CAPABILITY_VALIDATED, true);
        notifyValidatedChangedAndDismissed(nc);
        checkActivityEventuallyDestroyed();
        inOrder.verify(sMockCaptivePortalLoginMetrics).setDurationMs(200);
        inOrder.verify(sMockCaptivePortalLoginMetrics).statsWrite();
    }

    @Test
    @FeatureFlag(name = CAPTIVE_PORTAL_CUSTOM_TABS, enabled = true)
    public void testCustomTabsHandlesCorsRequests() throws Exception {
        assumeTrue(isChromeDefaultBrowser());
        assumeTrue(mPackageManager.hasSystemFeature(FEATURE_WIFI));
        final String testUrl = "https://captiveportaltest.androidtesting.org";
        final Network wifiNetwork = mConnectUtil.ensureWifiValidated();

        // The Activity will finish if the network is validated.
        // Mock the capabilities to be NOT validated so the activity doesn't finish.
        final NetworkCapabilities nonValidatedCaps = new NetworkCapabilities(
                sConnectivityManager.getNetworkCapabilities(wifiNetwork));
        nonValidatedCaps.removeCapability(NET_CAPABILITY_VALIDATED);
        doReturn(nonValidatedCaps).when(sConnectivityManager).getNetworkCapabilities(wifiNetwork);

        doAnswer(invocation -> {
            ConnectivityManager.NetworkCallback callback = invocation.getArgument(1);
            callback.onCapabilitiesChanged(wifiNetwork, nonValidatedCaps);
            return null;
        }).when(sConnectivityManager).registerNetworkCallback(any(), any(), any());
        doNothing().when(sConnectivityManager).unregisterNetworkCallback(any(
                ConnectivityManager.NetworkCallback.class));

        prepareRealCustomTabs(testUrl, wifiNetwork);

        final UiDevice device = UiDevice.getInstance(getInstrumentation());

        final BySelector titleSelector = By.text("Test Captive Portal Login");
        assertNotNull("Page title should be visible",
                device.wait(Until.findObject(titleSelector), TEST_TIMEOUT_MS));

        final BySelector corsSelector = By.text("CORS SUCCESS");
        assertNotNull("CORS should succeed",
                device.wait(Until.findObject(corsSelector), TEST_TIMEOUT_MS));

        final BySelector buttonSelector = By.text("Connect");
        assertNotNull("Connect button should exist",
                device.wait(Until.findObject(buttonSelector), TEST_TIMEOUT_MS));

        device.pressBack();
        checkActivityEventuallyDestroyed();
    }
}
