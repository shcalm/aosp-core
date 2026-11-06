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

#include "supplicant_wifi_rtt_controller.h"
#include "aidl_manager.h"
#include "aidl_return_util.h"
#include "misc_utils.h"

namespace aidl {
namespace android {
namespace hardware {
namespace wifi {
namespace supplicant {
using aidl_return_util::validateAndCall;
using misc_utils::createStatus;

std::shared_ptr<SupplicantWifiRttController> SupplicantWifiRttController::create(
	struct wpa_global* wpa_global, const char* ifname)
{
	std::shared_ptr<SupplicantWifiRttController> ptr =
		ndk::SharedRefBase::make<SupplicantWifiRttController>(
			wpa_global, ifname);
	std::weak_ptr<SupplicantWifiRttController> weak_ptr_this(ptr);
	ptr->setWeakPtr(weak_ptr_this);
	return ptr;
}

SupplicantWifiRttController::SupplicantWifiRttController(
	struct wpa_global* wpa_global, const char* ifname)
	: wpa_global_(wpa_global), ifname_(ifname), is_valid_(true)
{
}

void SupplicantWifiRttController::invalidate()
{
	wpa_global_ = nullptr;
	is_valid_ = false;
};

bool SupplicantWifiRttController::isValid() { return is_valid_; }

void SupplicantWifiRttController::setWeakPtr(
	std::weak_ptr<SupplicantWifiRttController> ptr)
{
	weak_ptr_this_ = ptr;
}

::ndk::ScopedAStatus SupplicantWifiRttController::getName(
	std::string* _aidl_return)
{
	*_aidl_return = ifname_;
	return ndk::ScopedAStatus::ok();
}

struct wpa_supplicant* SupplicantWifiRttController::retrieveIfacePtr()
{
	return wpa_supplicant_get_iface(wpa_global_, ifname_.c_str());
}

::ndk::ScopedAStatus SupplicantWifiRttController::getCapabilities(
	RttCapabilities* _aidl_return)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::getCapabilitiesInternal, _aidl_return);
}

::ndk::ScopedAStatus SupplicantWifiRttController::setProximityRangingDeviceName(
	const std::string& in_name)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::setProximityRangingDeviceNameInternal,
		in_name);
}

::ndk::ScopedAStatus SupplicantWifiRttController::setProximityRangingMacAddress(
	const std::array<uint8_t, 6>& in_macAddress)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::setProximityRangingMacAddressInternal,
		in_macAddress);
}

::ndk::ScopedAStatus SupplicantWifiRttController::getProximityRangingMacAddress(
	std::array<uint8_t, 6>* _aidl_return)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::getProximityRangingMacAddressInternal,
		_aidl_return);
}

::ndk::ScopedAStatus SupplicantWifiRttController::rangeRequest(
	int32_t in_cmdId, const std::vector<RttConfig>& in_rttConfigs)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::rangeRequestInternal, in_cmdId,
		in_rttConfigs);
}

::ndk::ScopedAStatus SupplicantWifiRttController::rangeCancel(
	int32_t in_cmdId, const std::vector<MacAddress>& in_addrs)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::rangeCancelInternal, in_cmdId,
		in_addrs);
}

::ndk::ScopedAStatus SupplicantWifiRttController::registerEventCallback(
	const std::shared_ptr<ISupplicantWifiRttControllerEventCallback>&
		in_callback)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_UNKNOWN,
		&SupplicantWifiRttController::registerEventCallbackInternal,
		in_callback);
}

// Internal implementations
std::pair<RttCapabilities, ndk::ScopedAStatus>
SupplicantWifiRttController::getCapabilitiesInternal()
{
	// TODO: implement
	return {RttCapabilities{}, ndk::ScopedAStatus::ok()};
}

ndk::ScopedAStatus
SupplicantWifiRttController::setProximityRangingDeviceNameInternal(
	const std::string& name)
{
	// TODO: implement
	return ndk::ScopedAStatus::ok();
}

ndk::ScopedAStatus
SupplicantWifiRttController::setProximityRangingMacAddressInternal(
	const std::array<uint8_t, 6>& macAddress)
{
	// TODO: implement
	return ndk::ScopedAStatus::ok();
}

std::pair<std::array<uint8_t, 6>, ndk::ScopedAStatus>
SupplicantWifiRttController::getProximityRangingMacAddressInternal()
{
	// TODO: implement
	return {std::array<uint8_t, 6>(), ndk::ScopedAStatus::ok()};
}

ndk::ScopedAStatus SupplicantWifiRttController::rangeRequestInternal(
	int32_t cmdId, const std::vector<RttConfig>& rttConfigs)
{
	// TODO: implement
	return ndk::ScopedAStatus::ok();
}

ndk::ScopedAStatus SupplicantWifiRttController::rangeCancelInternal(
	int32_t cmdId, const std::vector<MacAddress>& addrs)
{
	// TODO: implement
	return ndk::ScopedAStatus::ok();
}

ndk::ScopedAStatus SupplicantWifiRttController::registerEventCallbackInternal(
	const std::shared_ptr<ISupplicantWifiRttControllerEventCallback>&
		callback)
{
	AidlManager *aidl_manager = AidlManager::getInstance();
	if (!aidl_manager ||
		    aidl_manager->addWifiRttControllerEventCallbackAidlObject(ifname_, callback)) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}
	return ndk::ScopedAStatus::ok();
}

}  // namespace supplicant
}  // namespace wifi
}  // namespace hardware
}  // namespace android
}  // namespace aidl
