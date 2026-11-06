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

import android.app.Service;
import android.content.Intent;
import android.os.Binder;
import android.os.IBinder;

/**
 * An empty service that is bound to by the custom tab.
 *
 * <p>If the user goes back to their home screen while the custom tab is open, the captive portal
 * app will be frozen by the cached app freezer. Even if the user navigates back to the custom tab,
 * this will not take the captive portal app out of the freezer since only the browser process was
 * brought to the foreground.
 *
 * <p>This service fixes the problem by using the undocumented intent extra
 * android.support.customtabs.extra.KEEP_ALIVE on the CustomTabsIntent. This specifies a service
 * that the browser app can use to bind to its calling app, so that when the browser is in the
 * foreground, the calling app is also taken out of the freezer (since it is bound by a foreground
 * app at that point).
 */
public class CustomTabsKeepAliveService extends Service {
    private final Binder mBinder = new Binder();
    @Override
    public IBinder onBind(Intent intent) {
        return mBinder;
    }
}
