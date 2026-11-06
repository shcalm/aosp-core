/*
 * Copyright (C) 2014 The Android Open Source Project
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

import static android.net.ConnectivityManager.EXTRA_CAPTIVE_PORTAL_PROBE_SPEC;
import static android.net.NetworkCapabilities.NET_CAPABILITY_VALIDATED;

import static com.android.captiveportallogin.CaptivePortalLoginFlags.CAPTIVE_PORTAL_CUSTOM_TABS;
import static com.android.captiveportallogin.CaptivePortalLoginFlags.CAPTIVE_PORTAL_CUSTOM_TABS_WITH_MULTINETWORK_API;
import static com.android.captiveportallogin.CaptivePortalLoginFlags.USE_ANY_CUSTOM_TAB_PROVIDER;
import static com.android.captiveportallogin.CaptivePortalLoginFlags.USE_FULL_CUSTOM_TAB;
import static com.android.captiveportallogin.DownloadService.isDirectlyOpenType;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNWANTED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_WANTED_AS_IS;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_CCT;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW;
import static com.android.os.corenetworking.captiveportallogin.CaptivePortalLoginStatsLog.CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW;

import android.app.Activity;
import android.app.ActivityManager;
import android.app.AlertDialog;
import android.app.Application;
import android.app.PendingIntent;
import android.app.TaskInfo;
import android.app.admin.DevicePolicyManager;
import android.content.ActivityNotFoundException;
import android.content.BroadcastReceiver;
import android.content.ComponentName;
import android.content.Context;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.ServiceConnection;
import android.content.pm.PackageManager;
import android.content.pm.PackageManager.NameNotFoundException;
import android.content.pm.ResolveInfo;
import android.content.res.Configuration;
import android.content.res.Resources;
import android.graphics.Bitmap;
import android.graphics.Color;
import android.graphics.Insets;
import android.graphics.Rect;
import android.net.CaptivePortal;
import android.net.CaptivePortalData;
import android.net.ConnectivityManager;
import android.net.ConnectivityManager.NetworkCallback;
import android.net.LinkProperties;
import android.net.Network;
import android.net.NetworkCapabilities;
import android.net.NetworkRequest;
import android.net.Proxy;
import android.net.Uri;
import android.net.captiveportal.CaptivePortalProbeSpec;
import android.net.http.SslCertificate;
import android.net.http.SslError;
import android.os.Binder;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.os.OutcomeReceiver;
import android.os.ServiceSpecificException;
import android.os.SystemClock;
import android.os.SystemProperties;
import android.provider.DeviceConfig;
import android.provider.DocumentsContract;
import android.provider.MediaStore;
import android.system.OsConstants;
import android.text.TextUtils;
import android.util.ArrayMap;
import android.util.ArraySet;
import android.util.Log;
import android.util.Pair;
import android.util.SparseArray;
import android.util.TypedValue;
import android.view.LayoutInflater;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.view.ViewGroup;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.webkit.CookieManager;
import android.webkit.DownloadListener;
import android.webkit.SslErrorHandler;
import android.webkit.URLUtil;
import android.webkit.WebChromeClient;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;
import android.widget.FrameLayout;
import android.widget.LinearLayout;
import android.widget.ProgressBar;
import android.widget.RemoteViews;
import android.widget.TextView;
import android.widget.Toast;
import android.window.OnBackInvokedDispatcher;

import androidx.annotation.ChecksSdkIntAtLeast;
import androidx.annotation.GuardedBy;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.annotation.RequiresApi;
import androidx.annotation.StringRes;
import androidx.annotation.VisibleForTesting;
import androidx.browser.customtabs.CustomTabsCallback;
import androidx.browser.customtabs.CustomTabsClient;
import androidx.browser.customtabs.CustomTabsIntent;
import androidx.browser.customtabs.CustomTabsServiceConnection;
import androidx.browser.customtabs.CustomTabsSession;
import androidx.core.content.FileProvider;
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout;

import com.android.captiveportallogin.flags.Flags;
import com.android.internal.logging.nano.MetricsProto.MetricsEvent;
import com.android.modules.utils.build.SdkLevel;
import com.android.net.module.util.CollectionUtils;
import com.android.net.module.util.DeviceConfigUtils;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLConnection;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.Objects;
import java.util.Random;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.Executor;
import java.util.concurrent.atomic.AtomicBoolean;

public class CaptivePortalLoginActivity extends Activity {
    private static final String TAG = CaptivePortalLoginActivity.class.getSimpleName();
    private static final boolean DBG = true;
    private static final boolean VDBG = false;

    // Copy this value from CaptivePortalData to avoid needing an API bump, which would slow
    // down this work by many months and have to be maintained forever. See the constant with
    // the same name in CaptivePortalData.
    private static final int CAPTIVE_PORTAL_DATA_SOURCE_CAPPORT_WITH_CUSTOM_TABS_OPTIN = 2;

    private static final int SOCKET_TIMEOUT_MS = 10000;
    public static final String HTTP_LOCATION_HEADER_NAME = "Location";
    private static final String DEFAULT_CAPTIVE_PORTAL_HTTP_URL =
            "http://connectivitycheck.gstatic.com/generate_204";
    // This should match the FileProvider authority specified in the app manifest.
    private static final String FILE_PROVIDER_AUTHORITY =
            "com.android.captiveportallogin.fileprovider";
    // This should match the path name in the FileProvider paths XML.
    @VisibleForTesting
    static final String FILE_PROVIDER_DOWNLOAD_PATH = "downloads";
    private static final int NO_DIRECTLY_OPEN_TASK_ID = -1;
    private enum Result {
        DISMISSED(MetricsEvent.ACTION_CAPTIVE_PORTAL_LOGIN_RESULT_DISMISSED),
        UNWANTED(MetricsEvent.ACTION_CAPTIVE_PORTAL_LOGIN_RESULT_UNWANTED),
        WANTED_AS_IS(MetricsEvent.ACTION_CAPTIVE_PORTAL_LOGIN_RESULT_WANTED_AS_IS);

        final int metricsEvent;
        Result(int metricsEvent) { this.metricsEvent = metricsEvent; }
    };

    // Define the pending intent relevant constants for custom tab menu item.
    private static final String ACTION_CUSTOM_TABS_MENU_ITEM_DO_NOT_USE_THIS_NETWORK_CLICKED =
            "com.android.captiveportallogin.CUSTOM_TABS_MENU_ITEM_DO_NOT_USE_THIS_NETWORK_CLICKED";
    private static final String ACTION_CUSTOM_TABS_MENU_ITEM_USE_THIS_NETWORK_CLICKED =
            "com.android.captiveportallogin.CUSTOM_TABS_MENU_ITEM_USE_THIS_NETWORK_CLICKED";
    private static final String ACTION_CUSTOM_TABS_MENU_ITEM_USE_CLASSIC_VIEW_CLICKED =
            "com.android.captiveportallogin.CUSTOM_TABS_MENU_ITEM_USE_CLASSIC_VIEW_CLICKED";
    @VisibleForTesting
    public static final String EXTRA_USE_CLASSIC_VIEW =
            "com.android.captiveportallogin.EXTRA_USE_CLASSIC_VIEW";
    private static final String EXTRA_CUSTOM_TABS_INSTANCE_TOKEN =
            "com.android.captiveportallogin.CUSTOM_TABS_INSTANCE_TOKEN";
    private static final String CUSTOM_TAB_EXTRA_KEEPALIVE_SERVICE =
            "android.support.customtabs.extra.KEEP_ALIVE";
    private static final int DO_NOT_USE_THIS_NETWORK_PENDING_INTENT_REQUEST_CODE = 1001;
    private static final int USE_THIS_NETWORK_PENDING_INTENT_REQUEST_CODE = 1002;
    private static final int USE_CLASSIC_VIEW_PENDING_INTENT_REQUEST_CODE = 1003;
    private static final int ACTIVITY_REQUEST_CUSTOM_TABS = 100_000;

    private URL mUrl;
    private String mUrlString;
    private CaptivePortalProbeSpec mProbeSpec;
    private String mProbeSpecString;
    private String mUserAgent;
    private Network mNetwork;
    private CharSequence mVenueFriendlyName = null;
    @VisibleForTesting
    protected CaptivePortal mCaptivePortal;
    private NetworkCallback mNetworkCallback;
    private ConnectivityManager mCm;
    private DevicePolicyManager mDpm;
    private boolean mLaunchBrowser = false;
    private MyWebViewClient mWebViewClient;
    private SwipeRefreshLayout mSwipeRefreshLayout;
    // This member is just used in the UI thread model(e.g. onCreate and onDestroy), so non-final
    // should be fine.
    private boolean mCaptivePortalCustomTabsEnabled;
    private boolean mFullCustomTabEnabled;
    private boolean mUseCustomTabWithMultiNetworkApi = true;
    @ChecksSdkIntAtLeast(api = Build.VERSION_CODES.S)
    private boolean mUsingCustomTabs;
    // Ensures that done() happens once exactly, handling concurrent callers with atomic operations.
    private final AtomicBoolean isDone = new AtomicBoolean(false);
    // Must only be touched on the UI thread. This must be initialized to false for thread
    // visibility reasons (if initialized to true, the UI thread may still see false).
    private boolean mIsResumed = false;
    // Indicates the user's login state for the portal. This is set to true if the user is logging
    // in, and false if the user is already logged in, or null before the first NetworkCapabilities
    // change has been received.
    private Boolean mIsPortal;
    private CustomTabsMenuItemReceiver mCustomTabsMenuItemReceiver;
    private final Handler mHandler = new Handler(Looper.getMainLooper());

    // Persistence across configuration changes, e.g. when the device is rotated, the
    // window is resized in multi-window mode, or a hardware keyboard is attached.
    // When this happens the app needs to know not to create a new custom tab, or it will
    // have multiple tabs open on top of each other.
    // This must only be touched on the main thread of the app.
    private static final class PersistentState {
        CaptivePortalCustomTabsServiceConnection mServiceConnection = null;
        CaptivePortalCustomTabsCallback mCallback = null;
        IBinder mInstanceToken = null;
        long mStartTimeMs;

        void copyFrom(@NonNull PersistentState other) {
            mServiceConnection = other.mServiceConnection;
            mCallback = other.mCallback;
            mInstanceToken = other.mInstanceToken;
            mStartTimeMs = other.mStartTimeMs;
        }

        void noteTimeCreated(long startTimeMs) {
            mStartTimeMs = startTimeMs;
        }
    }
    // Must only be touched on the UI thread
    private final PersistentState mPersistentState = new PersistentState();

    private final CaptivePortalLoginMetrics mCaptivePortalLoginMetrics =
            getCaptivePortalLoginMetrics();

    private static final class CaptivePortalCustomTabsCallback extends CustomTabsCallback {
        @NonNull private CaptivePortalLoginActivity mParent;

        CaptivePortalCustomTabsCallback(@NonNull final CaptivePortalLoginActivity parent) {
            mParent = parent;
        }

        public void reparent(@NonNull final CaptivePortalLoginActivity newParent) {
            mParent = newParent;
        }

        @Override
        public void onNavigationEvent(int navigationEvent, @Nullable Bundle extras) {
            // If mIsPortal is null, the network may actually be a captive portal,
            // but we haven't received the first onNetworkCapabilitiesChanged yet.
            // We skip reevaluation in this state to avoid premature triggers before
            // the network state is fully initialized.
            final boolean shouldReevaluate = mParent.mIsPortal != null && mParent.mIsPortal;
            if (!mParent.isDone.get()) {
                // If done() has been called, no need to log further navigation events.
                final String event = navigationEventToString(navigationEvent);
                Log.d(TAG, "onNavigationEvent: " + event + " isPortal: " + shouldReevaluate);
            }
            if (shouldReevaluate
                    && (navigationEvent == NAVIGATION_STARTED
                            || navigationEvent == NAVIGATION_FINISHED)) {
                Log.d(TAG, "CCT navigation trigger network reevaluation");
                mParent.mCaptivePortal.reevaluateNetwork();
            }
            if (navigationEvent == TAB_HIDDEN && !mParent.mFullCustomTabEnabled) {
                // Run on UI thread to make sure mIsResumed is correctly visible.
                mParent.runOnUiThread(() -> {
                    // The tab is hidden when the browser's activity is hidden: screen off,
                    // home button, or press the close button on the tab. In the last case,
                    // close the app. The activity behind the tab is only resumed in that case.
                    if (mParent.mIsResumed) {
                        mParent.finishAndRemoveTask();
                    }
                });
            }
        }

        private static String navigationEventToString(int navigationEvent) {
            switch (navigationEvent) {
                case NAVIGATION_ABORTED: return "NAVIGATION_ABORTED";
                case NAVIGATION_FAILED: return "NAVIGATION_FAILED";
                case NAVIGATION_FINISHED: return "NAVIGATION_FINISHED";
                case NAVIGATION_STARTED: return "NAVIGATION_STARTED";
                case TAB_SHOWN: return "TAB_SHOWN";
                case TAB_HIDDEN: return "TAB_HIDDEN";
                default: return String.valueOf(navigationEvent);
            }
        }

        @Override
        public void onMinimized(@NonNull Bundle extras) {
            super.onMinimized(extras);
            if (!mParent.mFullCustomTabEnabled) {
                return;
            }
            // Hide the app on minimize, as it would otherwise show the activity behind the tab
            mParent.runOnUiThread(() -> {
                mParent.moveTaskToBack(true);
                setExcludeFromRecents(true);
            });
        }

        @Override
        public void onUnminimized(@NonNull Bundle extras) {
            super.onUnminimized(extras);
            if (!mParent.mFullCustomTabEnabled) {
                return;
            }
            mParent.runOnUiThread(() -> setExcludeFromRecents(false));
        }

        private void setExcludeFromRecents(boolean exclude) {
            final ActivityManager.AppTask myTask = CollectionUtils.findFirst(
                    mParent.getSystemService(ActivityManager.class).getAppTasks(),
                    t -> {
                        final TaskInfo info = t.getTaskInfo();
                        return info != null && info.taskId == mParent.getTaskId();
                    });
            if (myTask != null) {
                myTask.setExcludeFromRecents(exclude);
            }
        }
    }

    private void relaunchActivityWithWebview() {
        finishAndRemoveTask();
        final Bundle extras = new Bundle();
        extras.putParcelable(ConnectivityManager.EXTRA_NETWORK, mNetwork);
        extras.putString(ConnectivityManager.EXTRA_CAPTIVE_PORTAL_URL, mUrlString);
        extras.putString(ConnectivityManager.EXTRA_CAPTIVE_PORTAL_PROBE_SPEC,
                mProbeSpecString);
        extras.putString(ConnectivityManager.EXTRA_CAPTIVE_PORTAL_USER_AGENT, mUserAgent);
        extras.putBoolean(EXTRA_USE_CLASSIC_VIEW, true);
        startActivityFromCustomTabsMenuItem(mNetwork, extras);
    }

    /**
     * The broadcast receiver to receive the pending intent when a custom tabs menu item is clicked.
     */
    public final class CustomTabsMenuItemReceiver extends BroadcastReceiver {
        @Override
        public void onReceive(Context context, Intent intent) {
            final IBinder receivedToken =
                    intent.getExtras().getBinder(EXTRA_CUSTOM_TABS_INSTANCE_TOKEN);
            if (mPersistentState.mInstanceToken == null
                    || !Objects.equals(receivedToken, mPersistentState.mInstanceToken)) {
                Log.w(TAG, "Ignoring menu tap for a different instance (expected "
                        + mPersistentState.mInstanceToken + ", got " + receivedToken + ")");
                return;
            }
            final String action = intent.getAction();
            // TODO: if done was called already, but the portal closed again, these actions should
            // not be ignored. Currently done will not do anything if called multiple times.
            if (action.equals(ACTION_CUSTOM_TABS_MENU_ITEM_DO_NOT_USE_THIS_NETWORK_CLICKED)) {
                done(Result.UNWANTED);
            } else if (action.equals(ACTION_CUSTOM_TABS_MENU_ITEM_USE_THIS_NETWORK_CLICKED)) {
                done(Result.WANTED_AS_IS);
            } else if (action.equals(ACTION_CUSTOM_TABS_MENU_ITEM_USE_CLASSIC_VIEW_CLICKED)) {
                relaunchActivityWithWebview();
            } else {
                Log.e(TAG, "unknown menu action " + action);
            }
        }
    }

    @VisibleForTesting
    public long getElapsedRealtime() {
        return SystemClock.elapsedRealtime();
    }

    private static final class CaptivePortalCustomTabsServiceConnection extends
            CustomTabsServiceConnection {
        @NonNull private CaptivePortalLoginActivity mParent;

        CaptivePortalCustomTabsServiceConnection(
                @NonNull final CaptivePortalLoginActivity parent) {
            mParent = parent;
        }

        public void reparent(@NonNull final CaptivePortalLoginActivity newParent) {
            mParent = newParent;
        }

        @Override
        public void onCustomTabsServiceConnected(@NonNull ComponentName name,
                @NonNull CustomTabsClient client) {
            Log.d(TAG, "CustomTabs service connected");
            final CustomTabsSession session = client.newSession(mParent.mPersistentState.mCallback);

            int availableSpace = -1;
            if (!mParent.mFullCustomTabEnabled) {
                // Note: this cannot support screen size changes (rotation) as there is no API to
                // set the height after the initial start.
                final View remainingSpaceView = mParent.findViewById(
                        R.id.custom_tab_header_remaining_space);
                availableSpace = remainingSpaceView.getHeight();
                if (availableSpace < 100) {
                    // If for some reason the height of the view can't be obtained, do not crash.
                    // This used to happen when this code would run before the first layout pass.
                    // This bug should be fixed now, but layout is notoriously difficult to get and
                    // if for any reason there is still an issue it is better to use this estimate
                    // than to crash.
                    Log.wtf(TAG, "Remaining space can't be obtained. Layout not done ?");
                    final Rect windowSize =
                            mParent.getWindowManager().getCurrentWindowMetrics().getBounds();
                    final int top = (int) TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_DIP,
                            96 /* dp */, mParent.getResources().getDisplayMetrics());
                    availableSpace = (windowSize.bottom - windowSize.top) - top;
                }
            }

            final int size = (int) TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_DIP,
                    24 /* dp */, mParent.getResources().getDisplayMetrics());
            final Bitmap emptyIcon = Bitmap.createBitmap(size /* width */, size /* height */,
                    Bitmap.Config.ARGB_8888);
            emptyIcon.setPixel(0, 0, 0);

            // The application package name that will resolve to the CustomTabs intent
            // has been set in {@Link CustomTabsIntent.Builder} constructor, unnecessary
            // to call {@Link Intent#setPackage} to explicitly specify the package name
            // again.
            final CustomTabsIntent.Builder builder = new CustomTabsIntent.Builder(session)
                    .setShareState(CustomTabsIntent.SHARE_STATE_OFF)
                    // Do not show a title to avoid pages pretend they are part of the Android
                    // system.
                    .setShowTitle(false /* showTitle */)
                    // Don't show animations, because there is no content to animate from or to in
                    // this activity. As such, set the res IDs to zero, which code for no animation.
                    .setStartAnimations(mParent, 0, 0)
                    .setExitAnimations(mParent, 0, 0)
                    // Temporary workaround : use an empty icon for the close button. It doesn't
                    // prevent interaction, but it least it doesn't LOOK like you can press it.
                    .setCloseButtonIcon(emptyIcon)
                    // External handlers will not work since they won't know on what network to
                    // operate.
                    .setSendToExternalDefaultHandlerEnabled(false)
                    // No rounding on the corners so as to have the UI of the tab blend more
                    // closely with the header contents.
                    .setToolbarCornerRadiusDp(0)
                    // Use the identity of the captive portal login app
                    .setShareIdentityEnabled(true)
                    // Don't hide the URL bar when scrolling down, to make sure the user is always
                    // aware they are on the page from a captive portal.
                    .setUrlBarHidingEnabled(false)
                    // Remove the close button from tab.
                    // TODO: remove above temporary workaround: setCloseButtonIcon with an empty
                    // close button icon once all custom tabs provider support this API.
                    .setCloseButtonEnabled(false);

            if (mParent.mUseCustomTabWithMultiNetworkApi) {
                builder.setNetwork(mParent.mNetwork);
            }

            if (availableSpace > 0) {
                builder.setInitialActivityHeightPx(availableSpace,
                        CustomTabsIntent.ACTIVITY_HEIGHT_FIXED);
            }

            // Add customized custom tabs menu items if any.
            final List<Pair<String, PendingIntent>> menuItems = mParent.getCustomTabsMenuItems();
            for (Pair<String, PendingIntent> item : menuItems) {
                builder.addMenuItem(item.first, item.second);
            }
            final CustomTabsIntent customTabsIntent = builder.build();
            if (mParent.mFullCustomTabEnabled) {
                final String signInText = mParent.getHeaderTitle();
                final RemoteViews remoteViews = new RemoteViews(mParent.getPackageName(),
                        R.layout.custom_tab_bottom_toolbar);
                remoteViews.setTextViewText(R.id.bottom_toolbar_text, signInText);
                builder.setSecondaryToolbarViews(remoteViews, null, null);
                customTabsIntent.intent.putExtra(CUSTOM_TAB_EXTRA_KEEPALIVE_SERVICE,
                        new Intent(mParent.getApplicationContext(),
                                CustomTabsKeepAliveService.class));
            }
            // Remove Referrer Header from HTTP probe packet by setting an empty Uri
            // instance in EXTRA_REFERRER, make sure users using custom tabs have the
            // same experience as the custom tabs browser.
            final String emptyReferrer = "";
            customTabsIntent.intent.putExtra(Intent.EXTRA_REFERRER, Uri.parse(emptyReferrer));
            // Disable "Open in Browser" option in the toolbar menu.
            customTabsIntent.intent.putExtra(CustomTabsIntent.EXTRA_OPEN_IN_BROWSER_STATE,
                    CustomTabsIntent.OPEN_IN_BROWSER_STATE_OFF);
            // Replicate customTabsIntent.launchUrl, which calls intent.setData(url) and
            // ContextCompat.startActivity(context, intent, startAnimationBundle), but using
            // startActivityForResult to get notified when the activity is closed.
            customTabsIntent.intent.setData(Uri.parse(mParent.mUrl.toString()));
            mParent.startActivityForResult(customTabsIntent.intent, ACTIVITY_REQUEST_CUSTOM_TABS,
                    customTabsIntent.startAnimationBundle);
        }

        @Override
        public void onServiceDisconnected(ComponentName componentName) {
            Log.d(TAG, "CustomTabs service disconnected");
        }
    }

    // When starting downloads a file is created via startActivityForResult(ACTION_CREATE_DOCUMENT).
    // This array keeps the download request until the activity result is received. It is keyed by
    // requestCode sent in startActivityForResult.
    @GuardedBy("mDownloadRequests")
    private final SparseArray<DownloadRequest> mDownloadRequests = new SparseArray<>();
    @GuardedBy("mDownloadRequests")
    private int mNextDownloadRequestId = 1;

    // mDownloadService and mDirectlyOpenId must be always updated from the main thread.
    @VisibleForTesting
    int mDirectlyOpenId = NO_DIRECTLY_OPEN_TASK_ID;
    @Nullable
    private DownloadService.DownloadServiceBinder mDownloadService = null;
    private final ServiceConnection mDownloadServiceConn = new ServiceConnection() {
        @Override
        public void onServiceDisconnected(ComponentName name) {
            Log.d(TAG, "Download service disconnected");
            mDownloadService = null;
            // Service binding is lost. The spinner for the directly open tasks is no longer
            // needed.
            setProgressSpinnerVisibility(View.GONE);
        }

        @Override
        public void onServiceConnected(ComponentName name, IBinder binder) {
            Log.d(TAG, "Download service connected");
            mDownloadService = (DownloadService.DownloadServiceBinder) binder;
            mDownloadService.setProgressCallback(mProgressCallback);
            maybeStartPendingDownloads();
        }
    };

    @Override
    protected void onPause() {
        mIsResumed = false;
        super.onPause();
    }

    @Override
    protected void onResume() {
        mIsResumed = true;
        super.onResume();
    }

    @VisibleForTesting
    final DownloadService.ProgressCallback mProgressCallback =
            new DownloadService.ProgressCallback() {
        @Override
        public void onDownloadComplete(Uri inputFile, String mimeType, int downloadId,
                boolean success) {
            if (isDirectlyOpenType(mimeType) && success) {
                try {
                    startActivity(makeDirectlyOpenIntent(inputFile, mimeType));
                } catch (ActivityNotFoundException e) {
                    // Delete the directly open file if no activity could handle it. This is
                    // verified before downloading, so it should only happen when the handling app
                    // was uninstalled while downloading, which is vanishingly rare. Try to delete
                    // it in case of the target activity being removed somehow.
                    Log.wtf(TAG, "No activity could handle " + mimeType + " file.", e);
                    runOnUiThread(() -> tryDeleteFile(inputFile));
                }
            }

            verifyDownloadIdAndMaybeHideSpinner(downloadId);
        }

        @Override
        public void onDownloadAborted(int downloadId, int reason) {
            if (reason == DownloadService.DOWNLOAD_ABORTED_REASON_FILE_TOO_LARGE) {
                runOnUiThread(() -> Toast.makeText(CaptivePortalLoginActivity.this,
                        R.string.file_too_large_cancel_download, Toast.LENGTH_LONG).show());
            }

            verifyDownloadIdAndMaybeHideSpinner(downloadId);
        }

        private void verifyDownloadIdAndMaybeHideSpinner(int id) {
            // Hide the spinner when the task completed signal for the target task is received.
            //
            // mDirectlyOpenId will not be updated until the existing directly open task is
            // completed or the connection to the DownloadService is lost. If the id is updated to
            // NO_DIRECTLY_OPEN_TASK_ID because of the loss of connection to DownloadService, the
            // spinner should be already hidden. Receiving relevant callback is ignorable.
            runOnUiThread(() -> {
                if (mDirectlyOpenId == id) setProgressSpinnerVisibility(View.GONE);
            });
        }
    };

    @VisibleForTesting
    boolean isFeatureEnabled(final String name) {
        return DeviceConfigUtils.isCaptivePortalLoginFeatureEnabled(getApplicationContext(), name);
    }

    @VisibleForTesting
    boolean isFeatureNotChickenedOut(final String name) {
        return DeviceConfigUtils.isCaptivePortalLoginFeatureNotChickenedOut(getApplicationContext(),
                name);
    }

    @VisibleForTesting
    boolean isCustomTabsEnabledThroughMainlineFlag() {
        return Flags.captivePortalCustomTabsPublic();
    }

    @VisibleForTesting
    boolean getDeviceConfigPropertyBoolean(final String name, boolean defaultValue) {
        return DeviceConfigUtils.getDeviceConfigPropertyBoolean(
                DeviceConfig.NAMESPACE_CAPTIVEPORTALLOGIN, name, defaultValue);
    }

    private void maybeStartPendingDownloads() {
        ensureRunningOnMainThread();

        if (mDownloadService == null) return;
        synchronized (mDownloadRequests) {
            for (int i = 0; i < mDownloadRequests.size(); i++) {
                final DownloadRequest req = mDownloadRequests.valueAt(i);
                if (req.mOutFile == null) continue;

                final int dlId = mDownloadService.requestDownload(mNetwork, mUserAgent, req.mUrl,
                        req.mFilename, req.mOutFile, getApplicationContext(), req.mMimeType);
                if (isDirectlyOpenType(req.mMimeType)) {
                    mDirectlyOpenId = dlId;
                    setProgressSpinnerVisibility(View.VISIBLE);
                }

                mDownloadRequests.removeAt(i);
                i--;
            }
        }
    }

    private Intent makeDirectlyOpenIntent(Uri inputFile, String mimeType) {
        return new Intent(Intent.ACTION_VIEW)
                .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION
                        | Intent.FLAG_GRANT_WRITE_URI_PERMISSION)
                .setDataAndType(inputFile, mimeType);
    }

    private void tryDeleteFile(@NonNull Uri file) {
        ensureRunningOnMainThread();
        try {
            DocumentsContract.deleteDocument(getContentResolver(), file);
        } catch (FileNotFoundException e) {
            // Nothing to delete
            Log.wtf(TAG, file + " not found for deleting");
        }
    }

    private static final class DownloadRequest {
        @NonNull final String mUrl;
        @NonNull final String mFilename;
        @NonNull final String mMimeType;
        // mOutFile is null for requests where the device is currently asking the user to pick a
        // place to put the file. When the user has picked the file name, the request will be
        // replaced by a new one with the correct file name in onActivityResult.
        @Nullable final Uri mOutFile;
        DownloadRequest(@NonNull String url, @NonNull String filename, @NonNull String mimeType,
                @Nullable Uri outFile) {
            mUrl = url;
            mFilename = filename;
            mMimeType = mimeType;
            mOutFile = outFile;
        }
    }

    // Ideally there should be a setting to let the user decide whether they want to
    // use custom tabs from a non-default browser for captive portals. Most users are
    // expected not to want custom tabs from a non-default browser : there
    // is a good chance they don't trust the company making a non-default browser that
    // is installed by default on their phone, or even if they trust it they may just
    // dislike it. Users tend to be passionate about their browser preference.
    // Still there is a use case for this, like playing DRM-protected content. Absent
    // trust and like issues, a non-default browser is still probably a more competent
    // implementation than the webview, and while it probably doesn't have the user's
    // credentials or personal info, it is likely better at handling SSL errors, non-
    // default schemes, login status and the like.
    // Until there is such a setting, the captive portal login app should default to
    // only use the default browser, and use the webview if the default browser does
    // not support custom tabs with multi-networking.
    // However, temporarily to help with tests, using any browser with the available
    // capabilities is useful. As such, only do this if the hidden device config
    // USE_ANY_CUSTOM_TAB_PROVIDER is true.
    @Nullable
    String getAnyCustomTabsProviderPackage() {
        // Get all apps that can handle VIEW intents and Custom Tab service connections.
        final Intent activityIntent = new Intent(Intent.ACTION_VIEW, Uri.parse("http://"));
        final List<String> packages = new ArrayList<>();
        for (final ResolveInfo resolveInfo : getPackageManager()
                .queryIntentActivities(activityIntent, PackageManager.MATCH_ALL)) {
            if (null == resolveInfo || null == resolveInfo.activityInfo) continue;
            if (isMultiNetworkingSupportedByProvider(resolveInfo.activityInfo.packageName)) {
                packages.add(resolveInfo.activityInfo.packageName);
            }
        }
        if (packages.isEmpty()) return null;
        final List<String> priorities = Arrays.asList(".dev", ".canary", ".beta");
        for (String priority : priorities) {
            for (String packageName : packages) {
                if (packageName.endsWith(priority)) {
                    return packageName;
                }
            }
        }
        return packages.get(0);
    }

    @VisibleForTesting
    CaptivePortalLoginMetrics getCaptivePortalLoginMetrics() {
        return new CaptivePortalLoginMetrics();
    }

    @VisibleForTesting
    @Nullable
    String getDefaultCustomTabsProviderPackage() {
        return CustomTabsClient.getPackageName(getApplicationContext(), null /* packages */);
    }

    @VisibleForTesting
    int getPackageUid(@NonNull final String customTabsProviderPackageName)
            throws NameNotFoundException {
        return getPackageManager().getPackageUid(customTabsProviderPackageName, 0);
    }

    @VisibleForTesting
    boolean isMultiNetworkingSupportedByProvider(final String defaultPackageName) {
        if (defaultPackageName == null) {
            return false;
        }
        return CustomTabsClient.isSetNetworkSupported(getApplicationContext(), defaultPackageName);
    }

    @VisibleForTesting
    Context getContextForCustomTabsBinding() {
        return getApplicationContext();
    }

    @VisibleForTesting
    void startActivityFromCustomTabsMenuItem(@NonNull final Network network,
            @NonNull final Bundle intentExtras) {
        mCm.startCaptivePortalApp(network, intentExtras);
    }

    private void setApplyWindowInsetsListener(final int resourceId, final boolean insetBottom) {
        final View view = findViewById(resourceId);
        view.setOnApplyWindowInsetsListener((v, windowInsets) -> {
            final Insets insets = windowInsets.getInsets(WindowInsets.Type.systemBars());
            final int bottom = insetBottom ? insets.bottom : 0;
            v.setPadding(0 /* left */, insets.top /* top */, 0 /* right */, bottom);
            return windowInsets.inset(0, insets.top, 0, bottom);
        });
    }

    private void initializeCustomTabHeader() {
        setContentView(R.layout.activity_custom_tab_header);
        // No action bar as this activity implements its own UI instead, so it can display more
        // useful information, e.g. about VPN or private DNS handling.
        getActionBar().hide();
        final TextView headerTitle = findViewById(R.id.custom_tab_header_title);
        headerTitle.setText(getHeaderTitle());
        // To figure out the initial activity height of the custom tab, the captive
        // portal login app will leave space at the bottom and measure that space. Since
        // the custom tab does not inset its view to fit the bottom system windows, the
        // captive portal app also shouldn't so that the computation will match. Hence
        // insetBottom = false.
        setApplyWindowInsetsListener(R.id.custom_tab_header_top, false /* insetBottom */);
    }

    private void initializeWebView() {
        // Also initializes proxy system properties.
        mCm.bindProcessToNetwork(mNetwork);

        // Proxy system properties must be initialized before setContentView is called
        // because setContentView initializes the WebView logic which in turn reads the
        // system properties.
        setContentView(R.layout.activity_captive_portal_login);

        getActionBar().setDisplayShowHomeEnabled(false);
        getActionBar().setElevation(0); // remove shadow
        getActionBar().setTitle(getHeaderTitle());
        getActionBar().setSubtitle("");

        // In webview mode, the captive portal login app does not want to draw below the
        // system bars, either at the top or the bottom, so apply insets both on the top and
        // the bottom of the window. Note that on V+ the app is forced into edge-to-edge mode,
        // so for simplicity it also sets itself edge-to-edge on lower SDKs.
        setApplyWindowInsetsListener(R.id.container, true /* insetBottom */);

        final WebView webview = getWebview();
        webview.clearCache(true);
        CookieManager.getInstance().setAcceptThirdPartyCookies(webview, true);
        WebSettings webSettings = webview.getSettings();
        webSettings.setJavaScriptEnabled(true);
        webSettings.setMixedContentMode(WebSettings.MIXED_CONTENT_COMPATIBILITY_MODE);
        webSettings.setUseWideViewPort(true);
        webSettings.setLoadWithOverviewMode(true);
        webSettings.setSupportZoom(true);
        webSettings.setBuiltInZoomControls(true);
        webSettings.setDisplayZoomControls(false);
        webSettings.setDomStorageEnabled(true);
        mWebViewClient = new MyWebViewClient();
        webview.setWebViewClient(mWebViewClient);
        webview.setWebChromeClient(new MyWebChromeClient());
        webview.setDownloadListener(new PortalDownloadListener());
        // Start initial page load so WebView finishes loading proxy settings.
        // Actual load of mUrl is initiated by MyWebViewClient.
        webview.loadData("", "text/html", null);

        mSwipeRefreshLayout = findViewById(R.id.swipe_refresh);
        mSwipeRefreshLayout.setOnRefreshListener(() -> {
            webview.reload();
            mSwipeRefreshLayout.setRefreshing(true);
        });
    }

    private void bindCustomTabsService(@NonNull final String customTabsProviderPackageName,
            @NonNull final CaptivePortalCustomTabsServiceConnection connection) {
        CustomTabsClient.bindCustomTabsService(getContextForCustomTabsBinding(),
                customTabsProviderPackageName, connection);
    }

    @RequiresApi(Build.VERSION_CODES.S)
    private boolean bypassVpnAndPrivateDnsForCustomTabsProvider(
            @NonNull final String customTabsProviderPackageName,
            @NonNull final OutcomeReceiver<Void, ServiceSpecificException> receiver) {
        final Class captivePortalClass = mCaptivePortal.getClass();
        try {
            final Method setDelegateUidMethod =
                    captivePortalClass.getMethod("setDelegateUid", int.class, Executor.class,
                            OutcomeReceiver.class);
            setDelegateUidMethod.invoke(mCaptivePortal,
                    getPackageUid(customTabsProviderPackageName),
                    getMainExecutor(),
                    receiver);
            return true;
        } catch (ReflectiveOperationException | IllegalArgumentException e) {
            Log.e(TAG, "Reflection exception while setting delegate uid", e);
            return false;
        } catch (NameNotFoundException e) {
            Log.e(TAG, "Could not find the UID for " + customTabsProviderPackageName, e);
            return false;
        }
    }

    @Nullable
    private String getCustomTabsProviderPackageIfEnabled() {
        if (!mCaptivePortalCustomTabsEnabled) {
            mCaptivePortalLoginMetrics.setReason(
                    CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_FEATURE_NOT_ENABLED);
            return null;
        }

        final LinkProperties lp = mCm.getLinkProperties(mNetwork);
        if (lp == null) return null;
        if (lp.getPrivateDnsServerName() != null && !SdkLevel.isAtLeastB()) {
            Log.i(TAG, "Do not use custom tabs if private DNS (strict mode) is enabled under B");
            mCaptivePortalLoginMetrics.setReason(
                    CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_PRIVATE_DNS_ENABLED_V_AND_BELOW);
            return null;
        }

        final String defaultPackage = getDefaultCustomTabsProviderPackage();
        if (null != defaultPackage && isMultiNetworkingSupportedByProvider(defaultPackage)) {
            return defaultPackage;
        }
        if (null == defaultPackage) {
            mCaptivePortalLoginMetrics.setReason(
                    CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_CCT);
        } else if (!isMultiNetworkingSupportedByProvider(defaultPackage)) {
            mCaptivePortalLoginMetrics.setReason(
                    CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_NOT_SUPPORT_MULTI_NETWORK);
        }

        Log.i(TAG, "Default browser doesn't support custom tabs");

        // Intentionally no UX way to set this. It is useful for verifying the test-only feature
        // with the early development version of browser.
        final boolean useAnyCustomTabProvider =
                getDeviceConfigPropertyBoolean(USE_ANY_CUSTOM_TAB_PROVIDER,
                        false /* defaultValue */);
        if (!useAnyCustomTabProvider) return null;
        return getAnyCustomTabsProviderPackage();
    }

    /**
     * Create a pendingIntent with the specific custom tab menu item action and instance token,
     * the instance token is used to filter the out-of-date intents.
     * see {@link ACTION_CUSTOM_TABS_MENU_ITEM} and {@link EXTRA_CUSTOM_TABS_INSTANCE_TOKEN}.
     */
    private PendingIntent createPendingIntentForMenuItem(@NonNull final String action,
            final int requestCode) {
        final Intent intent = new Intent(action);
        final Bundle bundle = new Bundle();
        bundle.putBinder(EXTRA_CUSTOM_TABS_INSTANCE_TOKEN, mPersistentState.mInstanceToken);
        intent.putExtras(bundle);
        return PendingIntent.getBroadcast(getApplicationContext(), requestCode, intent,
                PendingIntent.FLAG_IMMUTABLE | PendingIntent.FLAG_UPDATE_CURRENT);
    }

    /**
     * Return the Custom Tabs menu items for captive portal operations.
     *
     *    - menu item 0: "Do not use this network"
     *    - menu item 1: "Use this network as is"
     *    - menu item 2: "Use classic view"
     */
    private List<Pair<String, PendingIntent>> getCustomTabsMenuItems() {
        final Resources resources = getResources();
        final ArrayList<Pair<String, PendingIntent>> menuItems = new ArrayList<>();
        final PendingIntent pendingIntent0 = createPendingIntentForMenuItem(
                ACTION_CUSTOM_TABS_MENU_ITEM_DO_NOT_USE_THIS_NETWORK_CLICKED,
                DO_NOT_USE_THIS_NETWORK_PENDING_INTENT_REQUEST_CODE
        );
        final PendingIntent pendingIntent1 = createPendingIntentForMenuItem(
                ACTION_CUSTOM_TABS_MENU_ITEM_USE_THIS_NETWORK_CLICKED,
                USE_THIS_NETWORK_PENDING_INTENT_REQUEST_CODE
        );
        final PendingIntent pendingIntent2 = createPendingIntentForMenuItem(
                ACTION_CUSTOM_TABS_MENU_ITEM_USE_CLASSIC_VIEW_CLICKED,
                USE_CLASSIC_VIEW_PENDING_INTENT_REQUEST_CODE
        );
        menuItems.add(new Pair<>(
                resources.getString(R.string.action_do_not_use_network),
                pendingIntent0));
        menuItems.add(new Pair<>(
                resources.getString(R.string.action_use_network),
                pendingIntent1));
        menuItems.add(new Pair<>(
                resources.getString(R.string.action_use_classic_view),
                pendingIntent2));
        return menuItems;
    }

    @Override
    public Object onRetainNonConfigurationInstance() {
        return mPersistentState;
    }

    private void enableEdgeToEdge() {
        // V and above is already always edge to edge (and it can't be turned off).
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.VANILLA_ICE_CREAM) return;

        // Taken and adapted from androidx.activity.ComponentActivity#enableEdgeToEdge
        final Window window = getWindow();
        window.setDecorFitsSystemWindows(false);

        final View decorView = window.getDecorView();
        final int viewNightMode = decorView.getResources().getConfiguration().uiMode
                & Configuration.UI_MODE_NIGHT_MASK;
        final boolean systemBarsAreDark = Configuration.UI_MODE_NIGHT_YES == viewNightMode;
        window.setStatusBarColor(Color.TRANSPARENT);
        window.setNavigationBarColor(Color.TRANSPARENT);
        window.setStatusBarContrastEnforced(false);
        window.setNavigationBarContrastEnforced(true);

        final int systemUiVisibility = decorView.getSystemUiVisibility();
        if (systemBarsAreDark) {
            decorView.setSystemUiVisibility(
                    systemUiVisibility & ~View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR);
        } else {
            decorView.setSystemUiVisibility(
                    systemUiVisibility | View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR);
        }
        final WindowInsetsController insetsController = window.getInsetsController();
        if (null == insetsController) return;
        insetsController.setSystemBarsAppearance(
                systemBarsAreDark ? 0 : WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS,
                WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS);
    }

    private void registerCustomTabsMenuItemBroadcastReceiver(
            @NonNull final CustomTabsMenuItemReceiver receiver) {
        final IntentFilter filter = new IntentFilter();
        filter.addAction(ACTION_CUSTOM_TABS_MENU_ITEM_DO_NOT_USE_THIS_NETWORK_CLICKED);
        filter.addAction(ACTION_CUSTOM_TABS_MENU_ITEM_USE_THIS_NETWORK_CLICKED);
        filter.addAction(ACTION_CUSTOM_TABS_MENU_ITEM_USE_CLASSIC_VIEW_CLICKED);
        registerReceiver(receiver,
                filter,
                null, /* broadcastPermission */
                mHandler,
                Context.RECEIVER_EXPORTED);
    }

    @Override
    protected void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        mCaptivePortalLoginMetrics.reset();
        // Initialize the feature flag after CaptivePortalLoginActivity is created, otherwise, the
        // context is still null and throw NPE when fetching the package manager from context.
        mCaptivePortal = getIntent().getParcelableExtra(ConnectivityManager.EXTRA_CAPTIVE_PORTAL);
        final PersistentState lastState = (PersistentState) getLastNonConfigurationInstance();
        if (null != lastState) {
            mPersistentState.copyFrom(lastState);
        } else {
            mPersistentState.noteTimeCreated(getElapsedRealtime());
            mCaptivePortalLoginMetrics.logAppStart();
        }
        // Null CaptivePortal is unexpected. The following flow will need to access mCaptivePortal
        // to communicate with system. Thus, finish the activity.
        if (mCaptivePortal == null) {
            Log.e(TAG, "Unexpected null CaptivePortal");
            finish();
            return;
        }
        mCm = getSystemService(ConnectivityManager.class);
        mDpm = getSystemService(DevicePolicyManager.class);
        mNetwork = getIntent().getParcelableExtra(ConnectivityManager.EXTRA_NETWORK);
        mNetwork = mNetwork.getPrivateDnsBypassingCopy();
        final CaptivePortalData capportData = getCapportData();
        final boolean optedInToCustomTabs;
        if (null == capportData) {
            mVenueFriendlyName = null;
            optedInToCustomTabs = false;
        } else {
            mVenueFriendlyName = getVenueFriendlyName(capportData);
            optedInToCustomTabs = isOptedInToCustomTabs(capportData);
        }

        // Custom tabs can be enabled in three ways:
        //  1) By pushing the legacy CAPTIVE_PORTAL_CUSTOM_TABS flag to the device
        //     (for droidfood population).
        //  2) When the captive portal operator opts in via the capport API (for early adoption).
        //  3) Through the mainline beta flag (for public rollout).
        final boolean customTabsFeatureEnabled = isFeatureEnabled(CAPTIVE_PORTAL_CUSTOM_TABS)
                || optedInToCustomTabs
                || isCustomTabsEnabledThroughMainlineFlag();

        // The feature can be force-disabled via a kill switch, useful in case of severe issues.
        final boolean isCustomTabsFeatureKillSwitchOff = isFeatureNotChickenedOut(
                CAPTIVE_PORTAL_CUSTOM_TABS);
        mCaptivePortalCustomTabsEnabled =
                isCustomTabsFeatureKillSwitchOff && customTabsFeatureEnabled;
        mFullCustomTabEnabled = mCaptivePortalCustomTabsEnabled && isFeatureNotChickenedOut(
                USE_FULL_CUSTOM_TAB);
        mUserAgent =
                getIntent().getStringExtra(ConnectivityManager.EXTRA_CAPTIVE_PORTAL_USER_AGENT);
        mUrlString = getIntent().getStringExtra(ConnectivityManager.EXTRA_CAPTIVE_PORTAL_URL);
        mUrl = deduceUrlFromIntentString(mUrlString);
        if (mUrl == null) {
            // getUrl() failed to parse the url provided in the intent: bail out in a way that
            // at least provides network access.
            done(Result.WANTED_AS_IS);
            return;
        }
        if (DBG) {
            Log.d(TAG, String.format("onCreate for %s", mUrl));
        }

        mProbeSpecString = getIntent().getStringExtra(EXTRA_CAPTIVE_PORTAL_PROBE_SPEC);
        try {
            mProbeSpec = CaptivePortalProbeSpec.parseSpecOrNull(mProbeSpecString);
        } catch (Exception e) {
            // Make extra sure that invalid configurations do not cause crashes
            mProbeSpec = null;
        }

        mNetworkCallback = new NetworkCallback() {
            @Override
            public void onLost(Network lostNetwork) {
                // If the network disappears while the app is up, exit.
                if (mNetwork.equals(lostNetwork) && !done(Result.UNWANTED)) {
                    // If done was already called, for example because the user logged in, just
                    // finish the activity.
                    finishAndRemoveTask();
                }
            }

            @Override
            public void onCapabilitiesChanged(Network network, NetworkCapabilities nc) {
                handleCapabilitiesChanged(network, nc);
            }

            @Override
            public void onLinkPropertiesChanged(Network network, LinkProperties lp) {
                handleLinkPropertiesChanged(network, lp);
            }
        };
        mCm.registerNetworkCallback(new NetworkRequest.Builder().build(), mNetworkCallback,
                mHandler);

        // If the network has disappeared, exit.
        final NetworkCapabilities networkCapabilities = mCm.getNetworkCapabilities(mNetwork);
        if (networkCapabilities == null) {
            finishAndRemoveTask();
            return;
        }

        maybeDeleteDirectlyOpenFile();

        final boolean forceWebview = getIntent().getBooleanExtra(EXTRA_USE_CLASSIC_VIEW, false);
        final String customTabsProviderPackageName = getCustomTabsProviderPackageIfEnabled();
        mUsingCustomTabs = !forceWebview && customTabsProviderPackageName != null;
        if (!mUsingCustomTabs) {
            if (forceWebview) {
                mCaptivePortalLoginMetrics.setReason(
                        CAPTIVE_PORTAL_LOGIN_REPORTED__REASON__REASON_USE_CLASSIC_VIEW);
            }
            if (SdkLevel.isAtLeastB()) {
                getOnBackInvokedDispatcher().registerOnBackInvokedCallback(
                        OnBackInvokedDispatcher.PRIORITY_DEFAULT,
                        () -> {
                            if (!webViewHandleBackPress()) {
                                finishAfterTransition();
                            }
                        }
                );
            }
            initializeWebView();
        } else {
            mUseCustomTabWithMultiNetworkApi = isFeatureNotChickenedOut(
                    CAPTIVE_PORTAL_CUSTOM_TABS_WITH_MULTINETWORK_API) || !Build.isDebuggable();
            if (mFullCustomTabEnabled) {
                setTheme(androidx.appcompat.R.style.Theme_AppCompat_DayNight_Dialog_Alert);
            }
            enableEdgeToEdge();

            if (mCustomTabsMenuItemReceiver == null) {
                mCustomTabsMenuItemReceiver = new CustomTabsMenuItemReceiver();
                registerCustomTabsMenuItemBroadcastReceiver(mCustomTabsMenuItemReceiver);
            }

            if (mFullCustomTabEnabled) {
                // full custom tab has the transparent style, only partial custom tab has the header
            } else {
                initializeCustomTabHeader();
            }
            if (mPersistentState.mInstanceToken == null) {
                mPersistentState.mInstanceToken = new Binder();
            }
            if (mPersistentState.mCallback != null) {
                mPersistentState.mCallback.reparent(this);
            } else {
                mPersistentState.mCallback = new CaptivePortalCustomTabsCallback(this);
            }

            final CompletableFuture<CaptivePortalCustomTabsServiceConnection> serviceConnected =
                    new CompletableFuture<>();
            final CompletableFuture<Boolean> layoutDone = new CompletableFuture<>();
            CompletableFuture.allOf(serviceConnected, layoutDone)
                    .thenRun(() -> {
                        final CaptivePortalCustomTabsServiceConnection connection;
                        try {
                            connection = serviceConnected.get();
                        } catch (InterruptedException | ExecutionException e) {
                            Log.e(TAG, "Service connection future completes with error", e);
                            return;
                        }
                        bindCustomTabsService(customTabsProviderPackageName, connection);
                        mPersistentState.mServiceConnection = connection;
                    });

            if (mFullCustomTabEnabled) {
                layoutDone.complete(true);
            } else {
                final View remainingSpaceView = findViewById(
                        R.id.custom_tab_header_remaining_space);
                remainingSpaceView.getViewTreeObserver().addOnGlobalLayoutListener(
                        () -> layoutDone.complete(true));
            }
            if (mPersistentState.mServiceConnection != null) {
                mPersistentState.mServiceConnection.reparent(this);
                serviceConnected.complete(mPersistentState.mServiceConnection);
            } else {
                final CaptivePortalCustomTabsServiceConnection connection =
                        new CaptivePortalCustomTabsServiceConnection(this);
                // TODO: Fall back to WebView if the custom tabs provider is not allowed to
                // bypass VPN or private DNS, e.g. an error or exception happens when calling
                // the {@link CaptivePortal#setDelegateUid} API.
                final boolean success = bypassVpnAndPrivateDnsForCustomTabsProvider(
                        customTabsProviderPackageName,
                        new OutcomeReceiver<>() {
                            // TODO: log the callback result metrics.
                            @Override
                            public void onResult(Void r) {
                                Log.d(TAG, "Set delegate uid for "
                                        + customTabsProviderPackageName
                                        + " to bypass VPN successfully");
                                serviceConnected.complete(connection);
                            }

                            @Override
                            public void onError(ServiceSpecificException e) {
                                Log.e(TAG, "Fail to set delegate uid for "
                                        + customTabsProviderPackageName + " to bypass VPN"
                                        + ", error: " + OsConstants.errnoName(e.errorCode), e);
                                serviceConnected.complete(connection);
                            }
                        });
                if (!success) { // caught an exception
                    serviceConnected.complete(connection);
                }
                try {
                    mCaptivePortalLoginMetrics.setUid(getPackageUid(customTabsProviderPackageName));
                } catch (NameNotFoundException e) {
                    // do nothing
                }
            }
        }
    }

    private void maybeDeleteDirectlyOpenFile() {
        // Try to remove the directly open files if exists.
        final File downloadPath = new File(getFilesDir(), FILE_PROVIDER_DOWNLOAD_PATH);
        try {
            deleteRecursively(downloadPath);
        } catch (IOException e) {
            Log.d(TAG, "Exception while deleting temp download files", e);
        }
    }

    private static boolean deleteRecursively(final File path) throws IOException {
        if (path.isDirectory()) {
            final File[] files = path.listFiles();
            if (files != null) {
                for (final File child : files) {
                    deleteRecursively(child);
                }
            }
        }
        return Files.deleteIfExists(Paths.get(path.toURI()));
    }

    @VisibleForTesting
    MyWebViewClient getWebViewClient() {
        return mWebViewClient;
    }

    @VisibleForTesting
    void handleCapabilitiesChanged(@NonNull final Network network,
            @NonNull final NetworkCapabilities nc) {
        if (!network.equals(mNetwork)) return;

        // The sequence of network capabilities for a captive portal network:
        // 1. `NET_CAPABILITY_CAPTIVE_PORTAL` is present initially, as soon as the portal
        //    is detected.
        // 2. `NET_CAPABILITY_VALIDATED` appears later, after the user has signed in and network
        //    validation completes successfully.
        final boolean wasPortal = mIsPortal != null && mIsPortal;
        final boolean isPortalNow = !nc.hasCapability(NET_CAPABILITY_VALIDATED);
        final boolean justLoggedIn = wasPortal && !isPortalNow;

        mIsPortal = isPortalNow;
        if ((mUsingCustomTabs && justLoggedIn) || (!mUsingCustomTabs && !isPortalNow)) {
            // Dismiss the portal (if using the Webview) or move the portal to backstack
            // (if using the custom tabs) when login is no longer needed since network has
            // validated.
            // TODO: this should also work if called twice. For now done does nothing after the
            // first call.
            done(Result.DISMISSED);
        }
    }

    @VisibleForTesting
    void handleLinkPropertiesChanged(@NonNull final Network network,
            @NonNull final LinkProperties lp) {
        if (!mNetwork.equals(network)) return;

        // The custom tab isn't initialized. The app is using WebView.
        if (!mUsingCustomTabs) return;

        // The private DNS bypass isn't supported on Android V and below for Custom tabs,
        // the activity should be dismissed if a user enables private DNS in settings while
        // a custom tab is active. The WebView will then be launched when the activity is
        // recreated.
        if (SdkLevel.isAtLeastB()) return;

        // Private DNS was not enabled when the app launched, otherwise, mUsingCustomTabs
        // would be false.
        if (lp.getPrivateDnsServerName() == null) return;

        // TODO: log this event metrics.
        done(Result.DISMISSED);
    }

    // Find WebView's proxy BroadcastReceiver and prompt it to read proxy system properties.
    private void setWebViewProxy() {
        // TODO: migrate to androidx WebView proxy setting API as soon as it is finalized
        try {
            final Field loadedApkField = Application.class.getDeclaredField("mLoadedApk");
            final Class<?> loadedApkClass = loadedApkField.getType();
            final Object loadedApk = loadedApkField.get(getApplication());
            Field receiversField = loadedApkClass.getDeclaredField("mReceivers");
            receiversField.setAccessible(true);
            ArrayMap receivers = (ArrayMap) receiversField.get(loadedApk);
            for (Object receiverMap : receivers.values()) {
                for (Object rec : ((ArrayMap) receiverMap).keySet()) {
                    Class clazz = rec.getClass();
                    if (clazz.getName().contains("ProxyChangeListener")) {
                        Method onReceiveMethod = clazz.getDeclaredMethod("onReceive", Context.class,
                                Intent.class);
                        Intent intent = new Intent(Proxy.PROXY_CHANGE_ACTION);
                        onReceiveMethod.invoke(rec, getApplicationContext(), intent);
                        Log.v(TAG, "Prompting WebView proxy reload.");
                    }
                }
            }
        } catch (Exception e) {
            Log.e(TAG, "Exception while setting WebView proxy: " + e);
        }
    }

    /**
     * Dismiss the portal app with the specified result.
     *
     * @return true if the portal was dismissed, false if ignored because this was called already.
     */
    private boolean done(Result result) {
        if (isDone.getAndSet(true)) {
            // isDone was already true: done() already called
            return false;
        }
        if (DBG) {
            Log.d(TAG, String.format("Result %s for %s", result.name(), mUrl));
        }
        switch (result) {
            case DISMISSED:
                mCaptivePortal.reportCaptivePortalDismissed();
                mCaptivePortalLoginMetrics.setPortalResult(
                        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_SUCCESS);
                break;
            case UNWANTED:
                mCaptivePortal.ignoreNetwork();
                mCaptivePortalLoginMetrics.setPortalResult(
                        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_UNWANTED);
                break;
            case WANTED_AS_IS:
                mCaptivePortal.useNetwork();
                mCaptivePortalLoginMetrics.setPortalResult(
                        CAPTIVE_PORTAL_LOGIN_REPORTED__PORTAL_RESULT__CAPTIVE_PORTAL_RESULT_WANTED_AS_IS);
                break;
        }
        mCaptivePortalLoginMetrics.setDurationMs(
                getElapsedRealtime() - mPersistentState.mStartTimeMs);
        mCaptivePortalLoginMetrics.statsWrite();
        if (result == Result.DISMISSED && mUsingCustomTabs) {
            // The user may resume their session in the custom tab portal from the background and
            // trigger new network validation checks (e.g., by refreshing the page or navigating
            // elsewhere). Therefore, continue ignoring upcoming network validation events to ensure
            // the portal remains open.
            moveTaskToBack(true);
            return true;
        }
        finishAndRemoveTask();
        return true;
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.captive_portal_login, menu);
        return true;
    }

    private boolean webViewHandleBackPress() {
        final WebView myWebView = findViewById(R.id.webview);
        // The web view is null if the app is using custom tabs
        if (null != myWebView && myWebView.canGoBack() && mWebViewClient.allowBack()) {
            myWebView.goBack();
            return true;
        } else {
            return false;
        }
    }

    @Override
    public void onBackPressed() {
        // This method is not being called from target SDK 36+. In SDK 36+
        // registerOnBackInvokedCallback() is used instead.
        if (!webViewHandleBackPress()) {
            super.onBackPressed();
        }
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {
        final Result result;
        final String action;
        final int id = item.getItemId();
        // This can't be a switch case because resource will be declared as static only but not
        // static final as of ADT 14 in a library project. See
        // http://tools.android.com/tips/non-constant-fields.
        if (id == R.id.action_use_network) {
            result = Result.WANTED_AS_IS;
            action = "USE_NETWORK";
        } else if (id == R.id.action_do_not_use_network) {
            result = Result.UNWANTED;
            action = "DO_NOT_USE_NETWORK";
        } else {
            return super.onOptionsItemSelected(item);
        }
        if (DBG) {
            Log.d(TAG, String.format("onOptionsItemSelect %s for %s", action, mUrl));
        }
        done(result);
        return true;
    }

    @Override
    public void onStop() {
        super.onStop();
        cancelPendingTask();
    }

    // This must be always called from main thread.
    private void setProgressSpinnerVisibility(int visibility) {
        ensureRunningOnMainThread();

        // getProgressLayout should never return null here, because this method is only ever called
        // when running in webview mode.
        getProgressLayout().setVisibility(visibility);
        if (visibility != View.VISIBLE) {
            mDirectlyOpenId = NO_DIRECTLY_OPEN_TASK_ID;
        }
    }

    @VisibleForTesting
    void cancelPendingTask() {
        ensureRunningOnMainThread();
        if (mDirectlyOpenId != NO_DIRECTLY_OPEN_TASK_ID) {
            Toast.makeText(this, R.string.cancel_pending_downloads, Toast.LENGTH_SHORT).show();
            // Remove the pending task for downloading the directly open file.
            mDownloadService.cancelTask(mDirectlyOpenId);
        }
    }

    private void ensureRunningOnMainThread() {
        if (Looper.getMainLooper().getThread() != Thread.currentThread()) {
            throw new IllegalStateException(
                    "Not running on main thread: " + Thread.currentThread().getName());
        }
    }

    @Override
    public void onDestroy() {
        super.onDestroy();

        if (mDownloadService != null) {
            unbindService(mDownloadServiceConn);
        }

        // When changing configurations, the activity will be restarted immediately by the
        // system. It will retain persistent state with onRetainNonConfigurationInstance,
        // and therefore the connection must not be severed just yet.
        if (!isChangingConfigurations()) {
            if (null != mPersistentState.mServiceConnection) {
                getContextForCustomTabsBinding().unbindService(mPersistentState.mServiceConnection);
                mPersistentState.mServiceConnection = null;
            }
            mPersistentState.mInstanceToken = null;
        }

        final WebView webview = (WebView) findViewById(R.id.webview);
        if (webview != null) {
            webview.stopLoading();
            webview.setWebViewClient(null);
            webview.setWebChromeClient(null);
            // According to the doc of WebView#destroy(), webview should be removed from the view
            // system before calling the WebView#destroy().
            ((ViewGroup) webview.getParent()).removeView(webview);
            webview.destroy();
        }
        if (mNetworkCallback != null) {
            // mNetworkCallback is not null if mUrl is not null.
            mCm.unregisterNetworkCallback(mNetworkCallback);
        }
        if (mCustomTabsMenuItemReceiver != null) {
            unregisterReceiver(mCustomTabsMenuItemReceiver);
        }
        if (mLaunchBrowser) {
            // Give time for this network to become default. After 500ms just proceed.
            for (int i = 0; i < 5; i++) {
                // TODO: This misses when mNetwork underlies a VPN.
                if (mNetwork.equals(mCm.getActiveNetwork())) break;
                try {
                    Thread.sleep(100);
                } catch (InterruptedException e) {
                }
            }
            final String url = mUrl.toString();
            if (DBG) {
                Log.d(TAG, "starting activity with intent ACTION_VIEW for " + url);
            }
            startActivity(new Intent(Intent.ACTION_VIEW, Uri.parse(url)));
        }
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode == ACTIVITY_REQUEST_CUSTOM_TABS) {
            // Close the app if the user closed the browser activity (for example by using the back
            // gesture on the first page).
            finishAndRemoveTask();
            return;
        }
        if (resultCode != RESULT_OK || data == null) return;

        // Start download after receiving a created file to download to
        final DownloadRequest pendingRequest;
        synchronized (mDownloadRequests) {
            pendingRequest = mDownloadRequests.get(requestCode);
            if (pendingRequest == null) {
                Log.e(TAG, "No pending download for request " + requestCode);
                return;
            }
        }

        final Uri fileUri = data.getData();
        if (fileUri == null) {
            Log.e(TAG, "No file received from download file creation result");
            return;
        }

        synchronized (mDownloadRequests) {
            // Replace the pending request with file uri in mDownloadRequests.
            mDownloadRequests.put(requestCode, new DownloadRequest(pendingRequest.mUrl,
                    pendingRequest.mFilename, pendingRequest.mMimeType, fileUri));
        }
        maybeStartPendingDownloads();
    }

    private URL deduceUrlFromIntentString(@Nullable final String url) {
        if (url == null) { // TODO: Have a metric to know how often empty url happened.
            // ConnectivityManager#getCaptivePortalServerUrl is deprecated starting with Android R.
            return makeURL(DEFAULT_CAPTIVE_PORTAL_HTTP_URL);
        }
        return makeURL(url);
    }

    private static URL makeURL(String url) {
        try {
            return new URL(url);
        } catch (MalformedURLException e) {
            Log.e(TAG, "Invalid URL " + url);
        }
        return null;
    }

    private static String host(URL url) {
        if (url == null) {
            return null;
        }
        return url.getHost();
    }

    private static String sanitizeURL(URL url) {
        // In non-Debug build, only show host to avoid leaking private info.
        return isDebuggable() ? Objects.toString(url) : host(url);
    }

    private static boolean isDebuggable() {
        return SystemProperties.getInt("ro.debuggable", 0) == 1;
    }

    private static boolean isDismissed(
            int httpResponseCode, String locationHeader, CaptivePortalProbeSpec probeSpec) {
        return (probeSpec != null)
                ? probeSpec.getResult(httpResponseCode, locationHeader).isSuccessful()
                : (httpResponseCode == 204);
    }

    @VisibleForTesting
    boolean hasVpnNetwork() {
        for (Network network : mCm.getAllNetworks()) {
            final NetworkCapabilities nc = mCm.getNetworkCapabilities(network);
            if (nc != null && nc.hasTransport(NetworkCapabilities.TRANSPORT_VPN)) {
                return true;
            }
        }

        return false;
    }

    @VisibleForTesting
    boolean isAlwaysOnVpnEnabled() {
        final ComponentName cn = new ComponentName(this, CaptivePortalLoginActivity.class);
        return mDpm.isAlwaysOnVpnLockdownEnabled(cn);
    }

    @VisibleForTesting
    class MyWebViewClient extends WebViewClient {
        private static final String INTERNAL_ASSETS = "file:///android_asset/";

        private final String mBrowserBailOutToken = Long.toString(new Random().nextLong());
        private final String mCertificateOutToken = Long.toString(new Random().nextLong());
        // How many Android device-independent-pixels per scaled-pixel
        // dp/sp = (px/sp) / (px/dp) = (1/sp) / (1/dp)
        private final float mDpPerSp = TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_SP, 1,
                    getResources().getDisplayMetrics()) /
                    TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_DIP, 1,
                    getResources().getDisplayMetrics());
        private int mPagesLoaded;
        private final ArraySet<String> mMainFrameUrls = new ArraySet<>();

        // If we haven't finished cleaning up the history, don't allow going back.
        public boolean allowBack() {
            return mPagesLoaded > 1;
        }

        private String mSslErrorTitle = null;
        private SslErrorHandler mSslErrorHandler = null;
        private SslError mSslError = null;

        @Override
        public void onPageStarted(WebView view, String urlString, Bitmap favicon) {
            if (urlString.contains(mBrowserBailOutToken)) {
                mLaunchBrowser = true;
                done(Result.WANTED_AS_IS);
                return;
            }
            // The first page load is used only to cause the WebView to
            // fetch the proxy settings.  Don't update the URL bar, and
            // don't check if the captive portal is still there.
            if (mPagesLoaded == 0) {
                return;
            }
            final URL url = makeURL(urlString);
            Log.d(TAG, "onPageStarted: " + sanitizeURL(url));
            // For internally generated pages, leave URL bar listing prior URL as this is the URL
            // the page refers to.
            if (!urlString.startsWith(INTERNAL_ASSETS)) {
                String subtitle = (url != null) ? getHeaderSubtitle(url) : urlString;
                getActionBar().setSubtitle(subtitle);
            }
            // getProgressBar() can't return null here because this method can only be
            // called in webview mode, not in custom tabs mode.
            getProgressBar().setVisibility(View.VISIBLE);
            mCaptivePortal.reevaluateNetwork();
        }

        @Override
        public void onPageFinished(WebView view, String url) {
            mPagesLoaded++;
            Log.d(TAG, "onPageFinished.");
            // getProgressBar() can't return null here because this method can only be
            // called in webview mode, not in custom tabs mode.
            getProgressBar().setVisibility(View.INVISIBLE);
            mSwipeRefreshLayout.setRefreshing(false);
            if (mPagesLoaded == 1) {
                // Now that WebView has loaded at least one page we know it has read in the proxy
                // settings.  Now prompt the WebView read the Network-specific proxy settings.
                setWebViewProxy();
                // Load the real page.
                view.loadUrl(mUrl.toString());
                return;
            } else if (mPagesLoaded == 2) {
                // Prevent going back to empty first page.
                // Fix for missing focus, see b/62449959 for details. Remove it once we get a
                // newer version of WebView (60.x.y).
                view.requestFocus();
                view.clearHistory();
            }
            mCaptivePortal.reevaluateNetwork();
        }

        // Convert Android scaled-pixels (sp) to HTML size.
        private String sp(int sp) {
            // Convert sp to dp's.
            float dp = sp * mDpPerSp;
            // Apply a scale factor to make things look right.
            dp *= 1.3;
            // Convert dp's to HTML size.
            // HTML px's are scaled just like dp's, so just add "px" suffix.
            return Integer.toString((int)dp) + "px";
        }

        // Check if webview is trying to load the main frame and record its url.
        @Override
        public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) {
            final String url = request.getUrl().toString();
            if (request.isForMainFrame()) {
                mMainFrameUrls.add(url);
            }
            // Be careful that two shouldOverrideUrlLoading methods are overridden, but
            // shouldOverrideUrlLoading(WebView view, String url) was deprecated in API level 24.
            // TODO: delete deprecated one ??
            return shouldOverrideUrlLoading(view, url);
        }

        // Record the initial main frame url. This is only called for the initial resource URL, not
        // any subsequent redirect URLs.
        @Override
        public WebResourceResponse shouldInterceptRequest(WebView view,
                WebResourceRequest request) {
            if (request.isForMainFrame()) {
                mMainFrameUrls.add(request.getUrl().toString());
            }
            return null;
        }

        // A web page consisting of a large broken lock icon to indicate SSL failure.
        @Override
        public void onReceivedSslError(WebView view, SslErrorHandler handler, SslError error) {
            final String strErrorUrl = error.getUrl();
            final URL errorUrl = makeURL(strErrorUrl);
            Log.d(TAG, String.format("SSL error: %s, url: %s, certificate: %s",
                    sslErrorName(error), sanitizeURL(errorUrl), error.getCertificate()));
            if (errorUrl == null
                    // Ignore SSL errors coming from subresources by comparing the
                    // main frame urls with SSL error url.
                    || (!mMainFrameUrls.contains(strErrorUrl))) {
                handler.cancel();
                return;
            }
            final String sslErrorPage = makeSslErrorPage();
            view.loadDataWithBaseURL(INTERNAL_ASSETS, sslErrorPage, "text/HTML", "UTF-8", null);
            mSslErrorTitle = view.getTitle() == null ? "" : view.getTitle();
            mSslErrorHandler = handler;
            mSslError = error;
        }

        private String makeHtmlTag() {
            if (getWebview().getLayoutDirection() == View.LAYOUT_DIRECTION_RTL) {
                return "<html dir=\"rtl\">";
            }

            return "<html>";
        }

        // If there is a VPN network or always-on VPN is enabled, there may be no way for user to
        // see the log-in page by browser. So, hide the link which is used to open the browser.
        @VisibleForTesting
        String getVpnMsgOrLinkToBrowser() {
            // Before Android R, CaptivePortalLogin cannot call the isAlwaysOnVpnLockdownEnabled()
            // to get the status of VPN always-on due to permission denied. So adding a version
            // check here to prevent CaptivePortalLogin crashes.
            if (hasVpnNetwork() || isAlwaysOnVpnEnabled()) {
                final String vpnWarning = getString(R.string.no_bypass_error_vpnwarning);
                return "  <div class=vpnwarning>" + vpnWarning + "</div><br>";
            }

            final String continueMsg = getString(R.string.error_continue_via_browser);
            return "  <a id=continue_link href=" + mBrowserBailOutToken + ">" + continueMsg
                    + "</a><br>";
        }

        private String makeErrorPage(@StringRes int warningMsgRes, @StringRes int exampleMsgRes,
                String extraLink) {
            final String warningMsg = getString(warningMsgRes);
            final String exampleMsg = getString(exampleMsgRes);
            return String.join("\n",
                    makeHtmlTag(),
                    "<head>",
                    "  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
                    "  <style>",
                    "    body {",
                    "      background-color:#fafafa;",
                    "      margin:auto;",
                    "      width:80%;",
                    "      margin-top: 96px",
                    "    }",
                    "    img {",
                    "      height:48px;",
                    "      width:48px;",
                    "    }",
                    "    div.warn {",
                    "      font-size:" + sp(16) + ";",
                    "      line-height:1.28;",
                    "      margin-top:16px;",
                    "      opacity:0.87;",
                    "    }",
                    "    div.example, div.vpnwarning {",
                    "      font-size:" + sp(14) + ";",
                    "      line-height:1.21905;",
                    "      margin-top:16px;",
                    "      opacity:0.54;",
                    "    }",
                    "    a {",
                    "      color:#4285F4;",
                    "      display:inline-block;",
                    "      font-size:" + sp(14) + ";",
                    "      font-weight:bold;",
                    "      margin-top:24px;",
                    "      text-decoration:none;",
                    "      text-transform:uppercase;",
                    "    }",
                    "  </style>",
                    "</head>",
                    "<body>",
                    "  <p><img src=quantum_ic_warning_amber_96.png><br>",
                    "  <div class=warn>" + warningMsg + "</div>",
                    "  <div class=example>" + exampleMsg + "</div>",
                    getVpnMsgOrLinkToBrowser(),
                    extraLink,
                    "</body>",
                    "</html>");
        }

        private String makeCustomSchemeErrorPage() {
            return makeErrorPage(R.string.custom_scheme_warning, R.string.custom_scheme_example,
                    "" /* extraLink */);
        }

        private String makeSslErrorPage() {
            final String certificateMsg = getString(R.string.ssl_error_view_certificate);
            return makeErrorPage(R.string.ssl_error_warning, R.string.ssl_error_example,
                    "<a id=cert_link href=" + mCertificateOutToken + ">" + certificateMsg
                            + "</a>");
        }

        @Override
        public boolean shouldOverrideUrlLoading (WebView view, String url) {
            if (url.startsWith("tel:")) {
                return startActivity(Intent.ACTION_DIAL, url);
            } else if (url.startsWith("sms:")) {
                return startActivity(Intent.ACTION_SENDTO, url);
            } else if (!url.startsWith("http:")
                    && !url.startsWith("https:") && !url.startsWith(INTERNAL_ASSETS)) {
                // If the page is not in a supported scheme (HTTP, HTTPS or internal page),
                // show an error page that informs the user that the page is not supported. The
                // user can bypass the warning and reopen the portal in browser if needed.
                // This is done as it is unclear whether third party applications can properly
                // handle multinetwork scenarios, if the scheme refers to a third party application.
                loadCustomSchemeErrorPage(view);
                return true;
            }
            if (url.contains(mCertificateOutToken) && mSslError != null) {
                showSslAlertDialog(mSslErrorHandler, mSslError, mSslErrorTitle);
                return true;
            }
            return false;
        }

        private boolean startActivity(String action, String uriData) {
            final Intent intent = new Intent(action, Uri.parse(uriData));
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            try {
                CaptivePortalLoginActivity.this.startActivity(intent);
                return true;
            } catch (ActivityNotFoundException e) {
                Log.e(TAG, "No activity found to handle captive portal intent", e);
                return false;
            }
        }

        protected void loadCustomSchemeErrorPage(WebView view) {
            final String errorPage = makeCustomSchemeErrorPage();
            view.loadDataWithBaseURL(INTERNAL_ASSETS, errorPage, "text/HTML", "UTF-8", null);
        }

        private void showSslAlertDialog(SslErrorHandler handler, SslError error, String title) {
            final LayoutInflater factory = LayoutInflater.from(CaptivePortalLoginActivity.this);
            final View sslWarningView = factory.inflate(R.layout.ssl_warning, null);

            // Set Security certificate
            setViewSecurityCertificate(sslWarningView.findViewById(R.id.certificate_layout), error);
            ((TextView) sslWarningView.findViewById(R.id.ssl_error_type))
                    .setText(sslErrorName(error));
            ((TextView) sslWarningView.findViewById(R.id.title)).setText(mSslErrorTitle);
            ((TextView) sslWarningView.findViewById(R.id.address)).setText(error.getUrl());

            AlertDialog sslAlertDialog = new AlertDialog.Builder(CaptivePortalLoginActivity.this)
                    .setTitle(R.string.ssl_security_warning_title)
                    .setView(sslWarningView)
                    .setPositiveButton(R.string.ok, (DialogInterface dialog, int whichButton) -> {
                        // handler.cancel is called via OnCancelListener.
                        dialog.cancel();
                    })
                    .setOnCancelListener((DialogInterface dialogInterface) -> handler.cancel())
                    .create();
            sslAlertDialog.show();
        }

        private void setViewSecurityCertificate(LinearLayout certificateLayout, SslError error) {
            ((TextView) certificateLayout.findViewById(R.id.ssl_error_msg))
                    .setText(sslErrorMessage(error));
            SslCertificate cert = error.getCertificate();
            // TODO: call the method directly once inflateCertificateView is @SystemApi
            try {
                final View certificateView = (View) SslCertificate.class.getMethod(
                        "inflateCertificateView", Context.class)
                        .invoke(cert, CaptivePortalLoginActivity.this);
                certificateLayout.addView(certificateView);
            } catch (ReflectiveOperationException | SecurityException e) {
                Log.e(TAG, "Could not create certificate view", e);
            }
        }
    }

    private class MyWebChromeClient extends WebChromeClient {
        @Override
        public void onProgressChanged(WebView view, int newProgress) {
            // getProgressBar() can't return null here because this method can only be
            // called in webview mode, not in custom tabs mode.
            getProgressBar().setProgress(newProgress);
        }
    }

    private class PortalDownloadListener implements DownloadListener {
        @Override
        public void onDownloadStart(String url, String userAgent, String contentDisposition,
                String mimetype, long contentLength) {
            final String normalizedType = Intent.normalizeMimeType(mimetype);
            // TODO: Need to sanitize the file name.
            final String displayName = URLUtil.guessFileName(
                    url, contentDisposition, normalizedType);

            String guessedMimetype = normalizedType;
            if (TextUtils.isEmpty(guessedMimetype)) {
                guessedMimetype = URLConnection.guessContentTypeFromName(displayName);
            }
            if (TextUtils.isEmpty(guessedMimetype)) {
                guessedMimetype = MediaStore.Downloads.CONTENT_TYPE;
            }

            Log.d(TAG, String.format("Starting download for %s, type %s with display name %s",
                    url, guessedMimetype, displayName));

            final int requestId;
            // WebView should call onDownloadStart from the UI thread, but to be extra-safe as
            // that is not documented behavior, access the download requests array with a lock.
            synchronized (mDownloadRequests) {
                requestId = mNextDownloadRequestId++;
                // Only bind the DownloadService for the first download. The request is put into
                // array later, so size == 0 with null mDownloadService means it's the first item.
                if (mDownloadService == null && mDownloadRequests.size() == 0) {
                    final Intent serviceIntent =
                            new Intent(CaptivePortalLoginActivity.this, DownloadService.class);
                    // To allow downloads to continue while the activity is closed, start service
                    // with a no-op intent, to make sure the service still gets put into started
                    // state.
                    startService(new Intent(getApplicationContext(), DownloadService.class));
                    bindService(serviceIntent, mDownloadServiceConn, Context.BIND_AUTO_CREATE);
                }
            }
            // Skip file picker for directly open MIME type, such as wifi Passpoint configuration
            // files. Fallback to generic design if the download process can not start successfully.
            if (isDirectlyOpenType(guessedMimetype)) {
                try {
                    startDirectlyOpenDownload(url, displayName, guessedMimetype, requestId);
                    return;
                } catch (IOException | ActivityNotFoundException e) {
                    // Fallthrough to show the file picker
                    Log.d(TAG, "Unable to do directly open on the file", e);
                }
            }

            synchronized (mDownloadRequests) {
                // outFile will be assigned after file is created.
                mDownloadRequests.put(requestId, new DownloadRequest(url, displayName,
                        guessedMimetype, null /* outFile */));
            }

            final Intent createFileIntent = DownloadService.makeCreateFileIntent(
                    guessedMimetype, displayName);
            try {
                startActivityForResult(createFileIntent, requestId);
            } catch (ActivityNotFoundException e) {
                // This could happen in theory if the device has no stock document provider (which
                // Android normally requires), or if the user disabled all of them, but
                // should be rare; the download cannot be started as no writeable file can be
                // created.
                Log.e(TAG, "No document provider found to create download file", e);
            }
        }

        private void startDirectlyOpenDownload(String url, String filename, String mimeType,
                int requestId) throws ActivityNotFoundException, IOException {
            ensureRunningOnMainThread();
            // Reject another directly open task if there is one task in progress. Using
            // mDirectlyOpenId here is ok because mDirectlyOpenId will not be updated to
            // non-NO_DIRECTLY_OPEN_TASK_ID until the new task is started.
            if (mDirectlyOpenId != NO_DIRECTLY_OPEN_TASK_ID) {
                Log.d(TAG, "Existing directly open task is in progress. Ignore this.");
                return;
            }

            final File downloadPath = new File(getFilesDir(), FILE_PROVIDER_DOWNLOAD_PATH);
            downloadPath.mkdirs();
            final File file = new File(downloadPath.getPath(), filename);

            final Uri uri = FileProvider.getUriForFile(
                    CaptivePortalLoginActivity.this, getFileProviderAuthority(), file);

            // Test if there is possible activity to handle this directly open file.
            final Intent testIntent = makeDirectlyOpenIntent(uri, mimeType);
            if (getPackageManager().resolveActivity(testIntent, 0 /* flag */) == null) {
                // No available activity is able to handle this.
                throw new ActivityNotFoundException("No available activity is able to handle "
                        + mimeType + " mime type file");
            }

            file.createNewFile();
            synchronized (mDownloadRequests) {
                mDownloadRequests.put(requestId, new DownloadRequest(url, filename, mimeType, uri));
            }

            maybeStartPendingDownloads();
        }
    }

    /**
     * Get the {@link androidx.core.content.FileProvider} authority for storing downloaded files.
     *
     * Useful for tests to override so they can use their own storage directories.
     */
    @VisibleForTesting
    String getFileProviderAuthority() {
        return FILE_PROVIDER_AUTHORITY;
    }

    @Nullable
    private ProgressBar getProgressBar() {
        return findViewById(R.id.progress_bar);
    }

    @Nullable
    private WebView getWebview() {
        return findViewById(R.id.webview);
    }

    @Nullable
    private FrameLayout getProgressLayout() {
        return findViewById(R.id.downloading_panel);
    }

    private String getHeaderTitle() {
        NetworkCapabilities nc = mCm.getNetworkCapabilities(mNetwork);
        final CharSequence networkName = getNetworkName(nc);
        if (TextUtils.isEmpty(networkName)
                || nc == null || !nc.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)) {
            return getString(R.string.action_bar_label);
        }
        return getString(R.string.action_bar_title, networkName);
    }

    private CharSequence getNetworkName(NetworkCapabilities nc) {
        // Use the venue friendly name if available
        if (!TextUtils.isEmpty(mVenueFriendlyName)) {
            return mVenueFriendlyName;
        }

        if (nc == null) {
            return null;
        }
        return removeDoubleQuotes(nc.getSsid());
    }



    private static String removeDoubleQuotes(String string) {
        if (string == null) return null;
        final int length = string.length();
        if ((length > 1) && (string.charAt(0) == '"') && (string.charAt(length - 1) == '"')) {
            return string.substring(1, length - 1);
        }
        return string;
    }

    private String getHeaderSubtitle(URL url) {
        String host = host(url);
        final String https = "https";
        if (https.equals(url.getProtocol())) {
            return https + "://" + host;
        }
        return host;
    }

    private static final SparseArray<String> SSL_ERRORS = new SparseArray<>();
    static {
        SSL_ERRORS.put(SslError.SSL_NOTYETVALID,  "SSL_NOTYETVALID");
        SSL_ERRORS.put(SslError.SSL_EXPIRED,      "SSL_EXPIRED");
        SSL_ERRORS.put(SslError.SSL_IDMISMATCH,   "SSL_IDMISMATCH");
        SSL_ERRORS.put(SslError.SSL_UNTRUSTED,    "SSL_UNTRUSTED");
        SSL_ERRORS.put(SslError.SSL_DATE_INVALID, "SSL_DATE_INVALID");
        SSL_ERRORS.put(SslError.SSL_INVALID,      "SSL_INVALID");
    }

    private static String sslErrorName(SslError error) {
        return SSL_ERRORS.get(error.getPrimaryError(), "UNKNOWN");
    }

    private static final SparseArray<Integer> SSL_ERROR_MSGS = new SparseArray<>();
    static {
        SSL_ERROR_MSGS.put(SslError.SSL_NOTYETVALID,  R.string.ssl_error_not_yet_valid);
        SSL_ERROR_MSGS.put(SslError.SSL_EXPIRED,      R.string.ssl_error_expired);
        SSL_ERROR_MSGS.put(SslError.SSL_IDMISMATCH,   R.string.ssl_error_mismatch);
        SSL_ERROR_MSGS.put(SslError.SSL_UNTRUSTED,    R.string.ssl_error_untrusted);
        SSL_ERROR_MSGS.put(SslError.SSL_DATE_INVALID, R.string.ssl_error_date_invalid);
        SSL_ERROR_MSGS.put(SslError.SSL_INVALID,      R.string.ssl_error_invalid);
    }

    private static Integer sslErrorMessage(SslError error) {
        return SSL_ERROR_MSGS.get(error.getPrimaryError(), R.string.ssl_error_unknown);
    }

    @Nullable
    private CaptivePortalData getCapportData() {
        final LinkProperties linkProperties = mCm.getLinkProperties(mNetwork);
        if (linkProperties == null) return null;
        return linkProperties.getCaptivePortalData();
    }

    private boolean isOptedInToCustomTabs(@NonNull final CaptivePortalData captivePortalData) {
        return captivePortalData.getUserPortalUrlSource()
                == CAPTIVE_PORTAL_DATA_SOURCE_CAPPORT_WITH_CUSTOM_TABS_OPTIN;
    }

    private CharSequence getVenueFriendlyName(@NonNull final CaptivePortalData captivePortalData) {
        // TODO: Use CaptivePortalData#getVenueFriendlyName when building with S
        // Use reflection for now
        final Class captivePortalDataClass = captivePortalData.getClass();
        try {
            final Method getVenueFriendlyNameMethod = captivePortalDataClass.getDeclaredMethod(
                    "getVenueFriendlyName");
            return (CharSequence) getVenueFriendlyNameMethod.invoke(captivePortalData);
        } catch (Exception e) {
            // Do nothing
        }
        return null;
    }
}
