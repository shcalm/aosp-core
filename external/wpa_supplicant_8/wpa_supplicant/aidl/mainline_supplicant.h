/*
 * WPA Supplicant - Root mainline supplicant interface
 * Copyright (c) 2025, Google Inc. All rights reserved.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */

#ifndef MAINLINE_SUPPLICANT_H
#define MAINLINE_SUPPLICANT_H

#include <map>

#include <aidl/android/hardware/wifi/supplicant/SupplicantStatusCode.h>

#include <aidl/android/system/wifi/mainline_supplicant/BnMainlineSupplicant.h>

#include "supplicant.h"

extern "C"
{
#include "utils/common.h"
#include "utils/includes.h"
#include "utils/wpa_debug.h"
#include "wpa_supplicant_i.h"
#include "scan.h"
}

using aidl::android::hardware::wifi::supplicant::ISupplicant;
using aidl::android::hardware::wifi::supplicant::Supplicant;
using aidl::android::hardware::wifi::supplicant::SupplicantStatusCode;

using aidl::android::system::wifi::mainline_supplicant::BnMainlineSupplicant;
using aidl::android::system::wifi::mainline_supplicant::ISupplicantNanIface;

class MainlineSupplicant : public BnMainlineSupplicant {
    public:
        MainlineSupplicant(struct wpa_global* global);
        ~MainlineSupplicant() override = default;

	bool isValid();
        // Aidl methods exposed.
        ::ndk::ScopedAStatus getVendorSupplicant(
                std::shared_ptr<ISupplicant>* _aidl_return) override;
        ::ndk::ScopedAStatus addNanInterface(const std::string& in_ifaceName,
                std::shared_ptr<ISupplicantNanIface>* _aidl_return) override;
        ::ndk::ScopedAStatus removeNanInterface(const std::string& in_ifaceName)
                override;
        ::ndk::ScopedAStatus setCurrentUserIdentity(int32_t in_userId) override;

    private:
        std::pair<std::shared_ptr<ISupplicant>, ndk::ScopedAStatus> getVendorSupplicantInternal();
        std::pair<std::shared_ptr<ISupplicantNanIface>, ndk::ScopedAStatus> addNanInterfaceInternal(
                const std::string& ifaceName);
        ndk::ScopedAStatus setCurrentUserIdentityInternal(uint32_t userId);
        struct wpa_global* wpa_global_;
        std::shared_ptr<Supplicant> vendor_supplicant_;
        std::map<std::string, std::shared_ptr<ISupplicantNanIface>> active_nan_ifaces_;
};

#endif // MAINLINE_SUPPLICANT_H
