/*
 * WPA Supplicant - Nan Iface Aidl interface
 * Copyright (c) 2025, Google Inc. All rights reserved.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */

#include "aidl_manager.h"
#include "aidl_return_util.h"
#include "driver_i.h"
#include "misc_utils.h"
#include "nan_iface.h"
#include "nan_supplicant.h"
#include "nan_utils.h"
#include "src/common/nan_de.h"
#include "src/common/nan_defs.h"

#include <iomanip>
#include <sstream>
#include <thread>

using aidl::android::hardware::wifi::supplicant::AidlManager;
using aidl::android::hardware::wifi::supplicant::aidl_return_util::validateAndCall;
using aidl::android::hardware::wifi::supplicant::misc_utils::convertVectorToWpaBuf;
using aidl::android::hardware::wifi::supplicant::misc_utils::createStatus;
using aidl::android::hardware::wifi::supplicant::misc_utils::ensureConfigFileExistsAtPath;
using aidl::android::hardware::wifi::supplicant::misc_utils::kIfaceDriverName;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::validateNanPublishConfig;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::validateNanSubscribeConfig;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::convertAidlNanPublishConfigToInternal;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::convertAidlNanSubscribeConfigToInternal;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::convertIntegerToRttBw;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::convertNanCipherSuiteTypeToSupplicantCipherSuiteType;
using aidl::android::system::wifi::mainline_supplicant::nan_utils::convertNanPairingSecurityTypeToInteger;
using NanPairingConfig = aidl::android::system::wifi::mainline_supplicant::NanPairingConfig;
using NanStatusCode = aidl::android::system::wifi::mainline_supplicant::NanStatus::NanStatusCode;
using NanPairingSecurityType =
	aidl::android::system::wifi::mainline_supplicant::
	NanPairingSecurityConfig::NanPairingSecurityType;
using NanCipherSuiteType =
	aidl::android::system::wifi::mainline_supplicant::NanCipherSuiteType;
using NanDataPathSecurityType =
	aidl::android::system::wifi::mainline_supplicant::
	NanDataPathSecurityConfig::NanDataPathSecurityType;
using NanDataPathSecurityConfig =
	aidl::android::system::wifi::mainline_supplicant::NanDataPathSecurityConfig;
using NanPairingRequestType =
	aidl::android::system::wifi::mainline_supplicant::NanPairingRequestType;
using NanPairingAkm =
	aidl::android::system::wifi::mainline_supplicant::NanPairingAkm;

const std::string kMainlineSupplicantConfigPath =
	"/apex/com.android.wifi/etc/wpa_supplicant_mainline.conf";
static constexpr int kNanIfaceCapMaxPublishSessions = 10;
static constexpr int kNanIfaceCapMaxSubscribeSessions = kNanIfaceCapMaxPublishSessions;
static constexpr int kNanMaxServiceNameLen = 255;
static constexpr int kNanMaxServiceSpecificInfoLen = 255;
static constexpr int kNanMaxMatchFilterLen = 255;
static constexpr int kNanMaxExtendedServiceSpecificInfoLen = 1280;
static constexpr bool kNanIfaceCapInstantCommunicationModeSupport = false;
static constexpr bool kNanIfaceCapSupportsPeriodicRanging = false;
static constexpr bool kNanIfaceCapSupportsSuspension = false;
static constexpr int kNanIfaceConfBufSize = 1024;
static constexpr int kNanIfaceConfBandDisableScan = 0;
static constexpr int kNanIfaceConfScanDwellTime = 150;
static constexpr int kNanIfaceConfScanPeriod = 20;
static constexpr int kNanIfaceConfBootstrapComebackTimeoutTus = 1024;
static constexpr uint16_t kNanIfaceConfAutoAcceptedBm = BIT(0);
static constexpr int kNanIfaceConfMaxBandwidth = 160;  // in MHz
static constexpr int kNanIfaceCapMaxNdiInterfaces = 2;
static constexpr int kNanIfaceCapMaxNdpSessions = 10;

template <typename... Args>
int setNanConfigParam(struct wpa_supplicant* wpa_s, const char* param, Args... args)
{
	char cmd[kNanIfaceConfBufSize];
	int ret = snprintf(cmd, kNanIfaceConfBufSize, param, args...);
	if (ret < 0 || ret >= sizeof(cmd)) {
		return 1;
	}
	return wpas_nan_set(wpa_s, cmd);
}

static std::string bytesToHexString(const uint8_t* data, size_t len) {
    if (!data || len == 0) return "";

    std::stringstream ss;
    ss << std::hex << std::setfill('0');

    for (size_t i = 0; i < len; ++i) {
        ss << std::setw(2) << static_cast<int>(data[i]);
    }

    return ss.str();
}

void NanIface::enqueue(std::function<void()> task) {
	{
		std::lock_guard<std::mutex> lock(mutex_);
		if (stop_worker_) {
			return;
		}
		task_queue_.push(std::move(task));
	}
	cv_.notify_one();
}

NanIface::NanIface(struct wpa_global* global, const std::string& ifname)
	: wpa_global_(global), ifname_(ifname), is_valid_(true), started_cluster_indication_(false),
	  joined_cluster_indication_(false)
{
	worker_thread_ = std::thread([this]() {
		while (true) {
			std::function<void()> task;
			{
				std::unique_lock<std::mutex> lock(mutex_);
				cv_.wait(lock, [this] {
					return !task_queue_.empty() || stop_worker_;
				});
				if (stop_worker_) {
					return;
				}
				task = std::move(task_queue_.front());
				task_queue_.pop();
			}
			task();
		}
	});
	wpa_printf(MSG_INFO, "NAN: Worker thread for iface %s started", ifname_.c_str());
}

NanIface::~NanIface() {
	{
		std::lock_guard<std::mutex> lock(mutex_);
		stop_worker_ = true;
	}
	cv_.notify_one();
	if (worker_thread_.joinable()) {
		worker_thread_.join();
	}
	task_queue_ = std::queue<std::function<void()>>();
	wpa_printf(MSG_INFO, "NAN: Worker thread for iface %s stopped", ifname_.c_str());
	invalidate();
}

void NanIface::invalidate() { is_valid_ = false; }
bool NanIface::isValid()
{
	return (
		is_valid_ && (wpa_supplicant_get_iface(
				  wpa_global_, ifname_.c_str()) != nullptr));
}

::ndk::ScopedAStatus NanIface::registerEventCallback(
	const std::shared_ptr<ISupplicantNanIfaceEventCallback>& in_callback)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::registerEventCallbackInternal, in_callback);
}

::ndk::ScopedAStatus NanIface::getCapabilitiesRequest(char16_t in_cmdId)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::getCapabilitiesRequestInternal, in_cmdId);
}

::ndk::ScopedAStatus NanIface::enableRequest(
	char16_t in_cmdId, const NanEnableRequest& in_msg1,
	const NanConfigRequest& in_msg2)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::enableRequestInternal, in_cmdId, in_msg1, in_msg2);
}

::ndk::ScopedAStatus NanIface::configRequest(
	char16_t in_cmdId, const NanConfigRequest& in_request)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::configRequestInternal, in_cmdId, in_request);
}

::ndk::ScopedAStatus NanIface::disableRequest(char16_t in_cmdId)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::disableRequestInternal, in_cmdId);
}

::ndk::ScopedAStatus NanIface::createDataInterfaceRequest(
	char16_t in_cmdId, const std::string& in_ifaceName,
	const std::array<uint8_t, 6>& macaddr)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::createDataInterfaceRequestInternal, in_cmdId, in_ifaceName, macaddr
	);
}

::ndk::ScopedAStatus NanIface::deleteDataInterfaceRequest(
	char16_t in_cmdId, const std::string& in_ifaceName)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::deleteDataInterfaceRequestInternal, in_cmdId, in_ifaceName
	);
}

::ndk::ScopedAStatus NanIface::getName(std::string* _aidl_return)
{
	*_aidl_return = ifname_;
	return  createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::startPublishRequest(
	char16_t in_cmdId, const NanPublishRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::startPublishRequestInternal, in_cmdId, in_msg
	);
}

::ndk::ScopedAStatus NanIface::stopPublishRequest(
	char16_t in_cmdId, int8_t in_sessionId)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::stopPublishRequestInternal, in_cmdId, in_sessionId
	);
}

::ndk::ScopedAStatus NanIface::startSubscribeRequest(
	char16_t in_cmdId, const NanSubscribeRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::startSubscribeRequestInternal, in_cmdId, in_msg
	);
}

::ndk::ScopedAStatus NanIface::stopSubscribeRequest(
	char16_t in_cmdId, int8_t in_sessionId)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::stopSubscribeRequestInternal, in_cmdId, in_sessionId
	);
}

::ndk::ScopedAStatus NanIface::transmitFollowupRequest(
	char16_t in_cmdId, const NanTransmitFollowupRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::transmitFollowupRequestInternal, in_cmdId, in_msg
	);
}

::ndk::ScopedAStatus NanIface::initiateBootstrappingRequest(
	char16_t in_cmdId, const NanBootstrappingRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::initiateBootstrappingRequestInternal, in_cmdId, in_msg);
}

::ndk::ScopedAStatus NanIface::respondToBootstrappingIndicationRequest(
	char16_t in_cmdId, const NanBootstrappingResponse& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::respondToBootstrappingIndicationRequestInternal, in_cmdId,
		in_msg);
}

::ndk::ScopedAStatus NanIface::initiatePairingRequest(
	char16_t in_cmdId, const NanPairingRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::initiatePairingRequestInternal, in_cmdId, in_msg);
}

::ndk::ScopedAStatus NanIface::respondToPairingIndicationRequest(
	char16_t in_cmdId, const NanRespondToPairingIndicationRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::respondToPairingIndicationRequestInternal, in_cmdId,
		in_msg);
}

::ndk::ScopedAStatus NanIface::terminatePairingRequest(
	char16_t in_cmdId, int32_t in_pairingInstanceId,
	const std::array<uint8_t, 6>& in_peerDiscMacAddr)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::terminatePairingRequestInternal, in_cmdId,
		in_pairingInstanceId, in_peerDiscMacAddr);
}

::ndk::ScopedAStatus NanIface::initiateDataPathRequest(
	char16_t in_cmdId, const NanInitiateDataPathRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::initiateDataPathRequestInternal, in_cmdId, in_msg);
}

::ndk::ScopedAStatus NanIface::respondToDataPathIndicationRequest(
	char16_t in_cmdId, const NanRespondToDataPathIndicationRequest& in_msg)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::respondToDataPathIndicationRequestInternal, in_cmdId,
		in_msg);
}

::ndk::ScopedAStatus NanIface::terminateDataPathRequest(
	char16_t in_cmdId, int32_t in_ndpInstanceId,
	const std::array<uint8_t, 6>& in_peerDiscMacAddr,
	const std::array<uint8_t, 6>& in_ndiInitMac)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::terminateDataPathRequestInternal, in_cmdId,
		in_ndpInstanceId, in_peerDiscMacAddr, in_ndiInitMac);
}

::ndk::ScopedAStatus NanIface::setSchedule(
	char16_t in_cmdId,
	const std::vector<NanSchedule>& in_schedule)
{
	return validateAndCall(
		this, SupplicantStatusCode::FAILURE_IFACE_INVALID,
		&NanIface::setScheduleInternal, in_cmdId, in_schedule);
}

#ifdef CONFIG_NAN
::ndk::ScopedAStatus NanIface::registerEventCallbackInternal(
	const std::shared_ptr<ISupplicantNanIfaceEventCallback>& callback)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager ||
		aidl_manager->addNanIfaceCallbackAidlObject(ifname_, callback)) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::getCapabilitiesRequestInternal(char16_t cmdId)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::SUCCESS;
		NanCapabilities aidl_caps = {};
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global_, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanCapabilitiesResponse(ifname, cmdId,
				nan_status, aidl_caps);
			return;
		}

		aidl_caps.maxPublishes = kNanIfaceCapMaxPublishSessions;
		aidl_caps.maxSubscribes = kNanIfaceCapMaxSubscribeSessions;
		aidl_caps.maxServiceNameLen = kNanMaxServiceNameLen;
		aidl_caps.maxServiceSpecificInfoLen = kNanMaxServiceSpecificInfoLen;
		aidl_caps.maxMatchFilterLen = kNanMaxMatchFilterLen;
		aidl_caps.maxExtendedServiceSpecificInfoLen =
			kNanMaxExtendedServiceSpecificInfoLen;
		aidl_caps.instantCommunicationModeSupportFlag =
			kNanIfaceCapInstantCommunicationModeSupport;
		aidl_caps.supportsSuspension = kNanIfaceCapSupportsSuspension;
		aidl_caps.supportsPeriodicRanging = kNanIfaceCapSupportsPeriodicRanging;
		aidl_caps.maxSupportedBandwidth = convertIntegerToRttBw(kNanIfaceConfMaxBandwidth);
		aidl_caps.maxNdiInterfaces = kNanIfaceCapMaxNdiInterfaces;
		aidl_caps.maxNdpSessions = kNanIfaceCapMaxNdpSessions;
		aidl_caps.supportedCipherSuites = wpa_s->nan_supported_csids;
#ifdef CONFIG_PASN
		if (wpa_s->nan_capa.drv_flags & WPA_DRIVER_FLAGS_NAN_SUPPORT_NDP) {
			aidl_caps.supportsPairing = true;
		}
#endif
		// the upper nibble represents RX
		aidl_caps.maxNumRxChainsSupported = (wpa_s->nan_capa.num_antennas >> 4) & 0x0f;
		aidl_manager->notifyNanCapabilitiesResponse(
			ifname, cmdId, nan_status, aidl_caps);
	});
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::configRequestInternal(
	char16_t cmdId, const NanConfigRequest& request)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanConfigResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::enableRequestInternal(
	char16_t cmdId, const NanEnableRequest& msg1,
	const NanConfigRequest& msg2)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanEnableResponse(ifname, cmdId, nan_status);
			return;
		}

		// Default with 2.4G band
		int8_t dual_band =
			(msg1.operateInBand[1] || msg1.operateInBand[2]) ? 1 : 0;
		int ret = 0;
		ret |= setNanConfigParam(wpa_s, "master_pref %d", msg2.masterPref);
		ret |= setNanConfigParam(wpa_s, "dual_band %d", dual_band);
		ret |= setNanConfigParam(
			wpa_s, "cluster_id " MACSTR, MAC2STR(msg2.clusterId));
		ret |= setNanConfigParam(
			wpa_s, "max_bw %d",  kNanIfaceConfMaxBandwidth
		);

		if (ret != 0) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanEnableResponse(
				ifname, cmdId, nan_status);
			return;
		}

		ret = wpas_nan_start(wpa_s);
		if (ret != 0) {
			aidl_manager->notifyNanEnableResponse(
				ifname, cmdId, nan_status);
			return;
		}
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanEnableResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::disableRequestInternal(char16_t cmdId)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanDisableResponse(ifname, cmdId, nan_status);
			return;
		}

		int ret = wpas_nan_stop(wpa_s);
		if (ret != 0) {
			aidl_manager->notifyNanDisableResponse(
				ifname, cmdId, nan_status);
			return;
		}
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanDisableResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::createDataInterfaceRequestInternal(
	char16_t cmdId, const std::string& ifaceName, const std::array<uint8_t, 6>& macAddr)
{
	u8 allocate_if_addr[ETH_ALEN];

	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_]  {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		// Get the wpa supplicant of NMI
		struct wpa_supplicant *wpa_s = wpa_supplicant_get_iface(wpa_global, ifname.c_str());

		if (!wpa_s) {
			wpa_printf(MSG_ERROR, "nl80211 parent interface not found");
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanCreateDataInterfaceResponse(ifname, cmdId, nan_status);
			return;
		}

		if (ensureConfigFileExistsAtPath(kMainlineSupplicantConfigPath) != 0) {
			wpa_printf(MSG_ERROR, "Conf file does not exists: %s",
				kMainlineSupplicantConfigPath.c_str());
			aidl_manager->notifyNanCreateDataInterfaceResponse(ifname, cmdId, nan_status);
			return;
		}

		if (wpa_drv_if_add(wpa_s, WPA_IF_NAN_DATA, ifaceName.c_str(), macAddr.data(),
						   NULL, NULL, (u8 *)allocate_if_addr, NULL) < 0)
		{
			wpa_printf(MSG_ERROR, "Failed to create NAN data iface");
			aidl_manager->notifyNanCreateDataInterfaceResponse(ifname, cmdId, nan_status);
			return;
		}

		struct wpa_interface iface_params = {};
		iface_params.driver = kIfaceDriverName;
		iface_params.ifname = ifaceName.c_str();
		iface_params.confname = kMainlineSupplicantConfigPath.c_str();
		iface_params.nan_data = true;

		if (!wpa_supplicant_add_iface(wpa_global, &iface_params, wpa_s)) {
			wpa_printf(MSG_ERROR, "Failed to add NAN data iface");
			wpa_drv_if_remove(wpa_s, WPA_IF_NAN, ifaceName.c_str());
			aidl_manager->notifyNanCreateDataInterfaceResponse(ifname, cmdId, nan_status);
			return;
		}
		wpa_s->added_vif = true;
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanCreateDataInterfaceResponse(ifname, cmdId, nan_status);
	});
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::deleteDataInterfaceRequestInternal(
	char16_t cmdId, const std::string& ifaceName)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	NanStatus nan_status;
	nan_status.status = NanStatusCode::INTERNAL_FAILURE;
	if (ifaceName.empty()) {
		wpa_printf(MSG_ERROR, "Empty nan data iface name provided");
		nan_status.status = NanStatusCode::INVALID_ARGS;
		aidl_manager->notifyNanDeleteDataInterfaceResponse(ifname_, cmdId, nan_status);
		return createStatus(SupplicantStatusCode::FAILURE_ARGS_INVALID);
	}

	struct wpa_supplicant *wpa_s = wpa_supplicant_get_iface(wpa_global_, ifaceName.c_str());
	if (!wpa_s) {
		wpa_printf(MSG_ERROR, "Failed to find NAN data iface.");
		nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
		aidl_manager->notifyNanDeleteDataInterfaceResponse(ifname_, cmdId, nan_status);
		return createStatus(SupplicantStatusCode::FAILURE_IFACE_UNKNOWN);
	}

	if (wpa_supplicant_remove_iface(wpa_global_, wpa_s, 0)) {
		wpa_printf(MSG_ERROR, "Failed to remove NAN data iface %s", ifaceName.c_str());
		aidl_manager->notifyNanDeleteDataInterfaceResponse(ifname_, cmdId, nan_status);
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	if (wpa_drv_if_remove(wpa_global_->ifaces, WPA_IF_NAN_DATA, ifaceName.c_str())) {
		wpa_printf(MSG_ERROR, "Failed to remove NAN data iface %s in WPA driver",
			ifaceName.c_str());
		aidl_manager->notifyNanDeleteDataInterfaceResponse(ifname_, cmdId, nan_status);
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	nan_status.status = NanStatusCode::SUCCESS;
	aidl_manager->notifyNanDeleteDataInterfaceResponse(ifname_, cmdId, nan_status);
	wpa_printf(MSG_INFO, "Interface %s was removed successfully", ifaceName.c_str());
	return ndk::ScopedAStatus::ok();
}

static std::string vectorToHexString(const std::vector<uint8_t>& input) {
    std::stringstream ss;
    ss << std::hex << std::setfill('0');

    for (uint8_t byte : input) {
        // std::setw(2) guarantees exactly two characters per byte
        // This ensures the total length is always even.
        ss << std::setw(2) << static_cast<int>(byte);
    }

    return ss.str();
}

::ndk::ScopedAStatus NanIface::startPublishRequestInternal(
	char16_t cmdId, const NanPublishRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanStartPublishResponse(
				ifname, cmdId, nan_status, -1);
			return;
		}

		if (!validateNanPublishConfig(msg)) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanStartPublishResponse(
				ifname, cmdId, nan_status, -1);
			return;
		}
		auto ssi_buf = convertVectorToWpaBuf(msg.baseConfig.serviceSpecificInfo);
		if (msg.baseConfig.sessionId != 0) {
			// update publish
			int ret = wpas_nan_update_publish(wpa_s, msg.baseConfig.sessionId, ssi_buf.get());
			if (ret >= 0) {
				nan_status.status = NanStatusCode::SUCCESS;
			}
			aidl_manager->notifyNanStartPublishResponse(ifname, cmdId, nan_status,
				msg.baseConfig.sessionId);
			return;
		}

                /* Hardcoded the schedule map config, remove when b/484354184 is done */
                char cmd[] = "map_id=1 5745:feffffff";
                if (wpas_nan_sched_config_map(wpa_s, cmd) < 0) {
                     wpa_printf(MSG_ERROR, "Failed to set hardcoded schedule map");
                }

		// Setup Bootstrapping & Pairing Config
		NanPairingConfig pairing_config = msg.pairingConfig;
		setNanConfigParam(wpa_s, "pairing_setup %d", pairing_config.enablePairingSetup ? 1 : 0);
		setNanConfigParam(wpa_s, "npk_caching %d", pairing_config.enablePairingCache ? 1 : 0);
		setNanConfigParam(wpa_s, "pairing_verification %d", pairing_config.enablePairingVerification ? 1 : 0);
		setNanConfigParam(wpa_s, "bootstrap_config %hx,%hx,%hu",
			static_cast<uint16_t>(pairing_config.supportedBootstrappingMethods),
			kNanIfaceConfAutoAcceptedBm, kNanIfaceConfBootstrapComebackTimeoutTus);
		if (!msg.identityKey.empty()) {
			setNanConfigParam(wpa_s, "nik %s", bytesToHexString(msg.identityKey.data(),
				msg.identityKey.size()).c_str());
		}

		struct nan_publish_params params =
			convertAidlNanPublishConfigToInternal(msg);
		// To prevent hex_matching_xx is freed before passing it to wpas_nan_publish,
		// it has to be assigned outside the helper function.
		std::string hex_matching_tx = vectorToHexString(msg.baseConfig.txMatchFilter);
		std::string hex_matching_rx = vectorToHexString(msg.baseConfig.rxMatchFilter);
		params.match_filter_tx = hex_matching_tx.c_str();
		params.match_filter_rx = hex_matching_rx.c_str();
		params.pbm = static_cast<uint16_t>(pairing_config.supportedBootstrappingMethods);

		discovery_termination_indication_ =
			!msg.baseConfig.disableDiscoveryTerminationIndication;
		match_expiration_indication_ =
			!msg.baseConfig.disableMatchExpirationIndication;
		followup_received_indication_ =
			!msg.baseConfig.disableFollowupReceivedIndication;

		// service name is guaranteed to be UTF-8, so it can be convert directly
		std::string srv_name(msg.baseConfig.serviceName.begin(), msg.baseConfig.serviceName.end());
		int publish_id = 0;
		publish_id = wpas_nan_publish(
			wpa_s,
			srv_name.c_str(),
			NAN_SRV_PROTO_GENERIC, ssi_buf.get(), &params, false);
		if (publish_id < 0) {
			aidl_manager->notifyNanStartPublishResponse(
				ifname, cmdId, nan_status, -1);
			return;
		}

		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanStartPublishResponse(ifname, cmdId, nan_status, publish_id);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::stopPublishRequestInternal(
	char16_t cmdId, int8_t sessionId)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::SUCCESS;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname_.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanStopPublishResponse(ifname, cmdId, nan_status);
			return;
		}

		wpas_nan_cancel_publish(wpa_s, sessionId);
		aidl_manager->notifyNanStopPublishResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::startSubscribeRequestInternal(
	char16_t cmdId, const NanSubscribeRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanStartSubscribeResponse(
				ifname, cmdId, nan_status, 0);
			return;
		}

		if (!validateNanSubscribeConfig(msg)) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanStartSubscribeResponse(
				ifname, cmdId, nan_status, 0);
			return;
		}

                /* Hardcoded the schedule map config, remove when b/484354184 is done */
                char cmd[] = "map_id=1 5745:feffffff";
                if (wpas_nan_sched_config_map(wpa_s, cmd) < 0) {
                    wpa_printf(MSG_ERROR, "Failed to set hardcoded schedule map");
                }

		if (msg.baseConfig.sessionId != 0) {
			// Ignore subscribe update, this feature unsupported.
			nan_status.status = NanStatusCode::SUCCESS;
			aidl_manager->notifyNanStartSubscribeResponse(ifname, cmdId, nan_status,
				msg.baseConfig.sessionId);
			return;
		}

		// Setup Bootstrapping & Pairing Config
		NanPairingConfig pairing_config = msg.pairingConfig;
		setNanConfigParam(wpa_s, "pairing_setup %d", pairing_config.enablePairingSetup ? 1 : 0);
		setNanConfigParam(wpa_s, "npk_caching %d", pairing_config.enablePairingCache ? 1 : 0);
		setNanConfigParam(wpa_s, "pairing_verification %d", pairing_config.enablePairingVerification ? 1 : 0);
		setNanConfigParam(wpa_s, "bootstrap_config %hx,%hx,%hu",
			static_cast<uint16_t>(pairing_config.supportedBootstrappingMethods),
			kNanIfaceConfAutoAcceptedBm, kNanIfaceConfBootstrapComebackTimeoutTus);
		if (!msg.identityKey.empty()) {
			setNanConfigParam(wpa_s, "nik %s", bytesToHexString(msg.identityKey.data(),
				msg.identityKey.size()).c_str());
		}

		struct nan_subscribe_params params =
			convertAidlNanSubscribeConfigToInternal(msg);
		// To prevent hex_matching_xx is freed before passing it to wpas_nan_publish,
		// it has to be assigned outside the helper function.
		std::string hex_matching_tx = vectorToHexString(msg.baseConfig.txMatchFilter);
		std::string hex_matching_rx = vectorToHexString(msg.baseConfig.rxMatchFilter);
		params.match_filter_tx = hex_matching_tx.c_str();
		params.match_filter_rx = hex_matching_rx.c_str();

		discovery_termination_indication_ =
			!msg.baseConfig.disableDiscoveryTerminationIndication;
		match_expiration_indication_ =
			!msg.baseConfig.disableMatchExpirationIndication;
		followup_received_indication_ =
			!msg.baseConfig.disableFollowupReceivedIndication;

		// service name is guaranteed to be UTF-8, so it can be convert directly
		std::string srv_name(msg.baseConfig.serviceName.begin(), msg.baseConfig.serviceName.end());

		auto ssi_buf =
			convertVectorToWpaBuf(msg.baseConfig.serviceSpecificInfo);
		int subscribe_id = 0;
		subscribe_id = wpas_nan_subscribe(
			wpa_s,
			srv_name.c_str(),
			NAN_SRV_PROTO_GENERIC, ssi_buf.get(), &params, false);
		if (subscribe_id < 0) {
			aidl_manager->notifyNanStartSubscribeResponse(
				ifname, cmdId, nan_status, 0);
			return;
		}
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanStartSubscribeResponse(ifname, cmdId, nan_status, subscribe_id);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::stopSubscribeRequestInternal(
	char16_t cmdId, int8_t sessionId)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::SUCCESS;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanStopSubscribeResponse(ifname, cmdId, nan_status);
			return;
		}
		wpas_nan_cancel_subscribe(wpa_s, sessionId);
		aidl_manager->notifyNanStopSubscribeResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::transmitFollowupRequestInternal(
	char16_t cmdId, const NanTransmitFollowupRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;

		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanTransmitFollowupResponse(ifname, cmdId, nan_status);
			return;
		}

		auto ssi_buf = convertVectorToWpaBuf(msg.serviceSpecificInfo);
		auto elems = convertVectorToWpaBuf(msg.extendedServiceSpecificInfo);
		int ret = wpas_nan_transmit(
			wpa_supplicant_get_iface(wpa_global, ifname.c_str()),
			msg.discoverySessionId, ssi_buf.get(), elems.get(),
			msg.addr.data(), msg.peerId);
		if (ret < 0) {
			aidl_manager->notifyNanTransmitFollowupResponse(
				ifname, cmdId, nan_status);
			return;
		}
		nan_status.status = NanStatusCode::SUCCESS;
		aidl_manager->notifyNanTransmitFollowupResponse(ifname, cmdId, nan_status);

		/* Transmit Followup Event when it is transmitted
		   remove it when b/488629386 is implemented */
		aidl_manager->notifyNanTransmitFollowup(wpa_s, cmdId, 0);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::initiateBootstrappingRequestInternal(
	char16_t cmdId, const NanBootstrappingRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;

		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanInitiateBootstrappingResponse(ifname, cmdId, nan_status, 0);
			return;
		}
		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			MACSTR " handle=%d req_instance_id=%d method=%d",
			MAC2STR(msg.peerDiscMacAddr), msg.discoverySessionId,
			msg.peerId, msg.requestBootstrappingMethod);
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanInitiateBootstrappingResponse(
				ifname, cmdId, nan_status, 0);
			return;
		}
		if (wpas_nan_bootstrap_request(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		}
		// supplicant reuses the discvoery session id as bootstrapping session id
		aidl_manager->notifyNanInitiateBootstrappingResponse(ifname, cmdId,
			nan_status, msg.discoverySessionId);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::respondToBootstrappingIndicationRequestInternal(
	char16_t cmdId, const NanBootstrappingResponse& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;

		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanRespondToBootstrappingIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}
		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			MACSTR " handle=%d method=%d auth",
			MAC2STR(msg.peerDiscMacAddr), msg.discoverySessionId,
			msg.responseBootstrappingMethod);
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanRespondToBootstrappingIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}
		if (wpas_nan_bootstrap_request(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		}
		aidl_manager->notifyNanRespondToBootstrappingIndicationResponse(
			ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::initiatePairingRequestInternal(
	char16_t cmdId, const NanPairingRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INVALID_ARGS;

		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
				nan_status, msg.peerId);
			return;
		}

		setNanConfigParam(wpa_s, "npk_caching %d", msg.enablePairingCache ? 1 : 0);
		if (!msg.pairingIdentityKey.empty()) {
			setNanConfigParam(wpa_s, "nik %s", bytesToHexString(msg.pairingIdentityKey.data(),
				msg.pairingIdentityKey.size()).c_str());
		}

		// Update pairing credentials for verification
		if (msg.requestType == NanPairingRequestType::NAN_PAIRING_VERIFICATION) {
			if (!msg.peerIdentityKey) {
				wpa_printf(MSG_ERROR, "No peer identity key provided");
				aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
					nan_status, msg.peerId);
				return;
			}
			int akmp  = 0;
			if (msg.securityConfig.akm == NanPairingAkm::PASN) {
				akmp = WPA_KEY_MGMT_PASN;
			} else {
				akmp = WPA_KEY_MGMT_SAE;
			}
			if (wpas_nan_update_pairing_credentials(
					wpa_s, msg.peerIdentityKey->data(), msg.peerIdentityKey->size(),
					NAN_NIRA_CIPHER_VER_128, akmp, msg.securityConfig.pmk.data(),
					msg.securityConfig.pmk.size()) < 0) {
				wpa_printf(MSG_ERROR, "Failed to update pairing credentials");

			}
		}

		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			MACSTR " handle=%d peer_instance_id=%d auth=%d",
			MAC2STR(msg.peerDiscMacAddr), msg.discoverySessionId,
			msg.peerId, convertNanPairingSecurityTypeToInteger(
				msg.securityConfig.securityType));
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
				nan_status, msg.peerId);
			return;
		}
		if (msg.securityConfig.securityType == NanPairingSecurityType::PASSPHRASE &&
			msg.securityConfig.passphrase.size() > 0) {
			cnt += snprintf(
				cmd + cnt, kNanIfaceConfBufSize - cnt,
				" password=%s",
				std::string(msg.securityConfig.passphrase.begin(),
				msg.securityConfig.passphrase.end()).c_str());
			if (cnt < 0 || cnt >= sizeof(cmd)) {
				aidl_manager->notifyNanInitiatePairingResponse(
					ifname, cmdId, nan_status, msg.peerId);
				return;
			}
		}
		// Only support PUBLIC_KEY_PASN_128_MASK and PUBLIC_KEY_PASN_256_MASK for pairing
		if (msg.securityConfig.cipherType == NanCipherSuiteType::PUBLIC_KEY_PASN_128_MASK) {
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt, " cipher=CCMP");
		} else if (msg.securityConfig.cipherType ==
				NanCipherSuiteType::PUBLIC_KEY_PASN_256_MASK) {
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt,
				" cipher=GCMP-256");
		} else {
			wpa_printf(MSG_ERROR, "Invalid cipher suite type");
			aidl_manager->notifyNanInitiatePairingResponse(
				ifname, cmdId, nan_status, msg.peerId);
			return;
		}
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
				nan_status, msg.peerId);
			return;
		}
		if (wpas_nan_pairing_start(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		} else {
			nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		}
		aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
			nan_status, msg.peerId);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::respondToPairingIndicationRequestInternal(
	char16_t cmdId, const NanRespondToPairingIndicationRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INVALID_ARGS;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanRespondToPairingIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}

		if (!msg.acceptRequest) {
			char cmd[kNanIfaceConfBufSize];
			int ret = snprintf(cmd, kNanIfaceConfBufSize,
				MACSTR, MAC2STR(msg.peerDiscMacAddr));
			if (ret > 0 && ret < kNanIfaceConfBufSize &&
				wpas_nan_pairing_abort(wpa_s, cmd) >= 0) {
				nan_status.status = NanStatusCode::SUCCESS;
			} else {
				nan_status.status = NanStatusCode::INTERNAL_FAILURE;
			}
			aidl_manager->notifyNanRespondToPairingIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}

		setNanConfigParam(wpa_s, "npk_caching %d", msg.enablePairingCache ? 1 : 0);
		if (!msg.pairingIdentityKey.empty()) {
			setNanConfigParam(wpa_s, "nik %s", bytesToHexString(msg.pairingIdentityKey.data(),
				msg.pairingIdentityKey.size()).c_str());
		}

		// Update pairing credentials for verification
		if (msg.requestType == NanPairingRequestType::NAN_PAIRING_VERIFICATION) {
			if (!msg.peerIdentityKey) {
				wpa_printf(MSG_ERROR, "No peer identity key provided");
				aidl_manager->notifyNanRespondToPairingIndicationResponse(
						ifname, cmdId, nan_status);
				return;
			}
			int akmp  = 0;
			if (msg.securityConfig.akm == NanPairingAkm::PASN) {
				akmp = WPA_KEY_MGMT_PASN;
			} else {
				akmp = WPA_KEY_MGMT_SAE;
			}
			if (wpas_nan_update_pairing_credentials(
					wpa_s, msg.peerIdentityKey->data(), msg.peerIdentityKey->size(),
					NAN_NIRA_CIPHER_VER_128, akmp, msg.securityConfig.pmk.data(),
					msg.securityConfig.pmk.size()) < 0) {
				wpa_printf(MSG_ERROR, "Failed to update pairing credentials");
			}
		}

		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			MACSTR " handle=%d peer_instance_id=%d auth=%d responder",
			MAC2STR(msg.peerDiscMacAddr), msg.discoverySessionId,
			msg.pairingInstanceId, convertNanPairingSecurityTypeToInteger(
				msg.securityConfig.securityType));
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
				nan_status, msg.pairingInstanceId);
			return;
		}
		if (msg.securityConfig.securityType == NanPairingSecurityType::PASSPHRASE)  {
			cnt += snprintf(
				cmd + cnt, kNanIfaceConfBufSize - cnt,
				" password=%s",
				std::string(msg.securityConfig.passphrase.begin(),
				msg.securityConfig.passphrase.end()).c_str());
			if (cnt < 0 || cnt >= sizeof(cmd)) {
				aidl_manager->notifyNanInitiatePairingResponse(
					ifname, cmdId, nan_status, msg.pairingInstanceId);
				return;
			}
		}
		// Only support PUBLIC_KEY_PASN_128_MASK and PUBLIC_KEY_PASN_256_MASK for pairing
		if (msg.securityConfig.cipherType == NanCipherSuiteType::PUBLIC_KEY_PASN_128_MASK) {
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt, " cipher=CCMP");
		} else if (msg.securityConfig.cipherType ==
				NanCipherSuiteType::PUBLIC_KEY_PASN_256_MASK) {
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt,
				" cipher=GCMP-256");
		} else {
			wpa_printf(MSG_ERROR, "Invalid cipher suite type");
			aidl_manager->notifyNanInitiatePairingResponse(
				ifname, cmdId, nan_status, msg.pairingInstanceId);
			return;
		}
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiatePairingResponse(ifname, cmdId,
				nan_status, msg.pairingInstanceId);
			return;
		}
		if (wpas_nan_pairing_start(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		} else {
			nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		}
		aidl_manager->notifyNanRespondToPairingIndicationResponse(
			ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::terminatePairingRequestInternal(
	char16_t cmdId, int32_t pairingInstanceId, const std::array<uint8_t, 6>& peerDiscMacAddr)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanTerminatePairingResponse(ifname, cmdId, nan_status);
			return;
		}
		char cmd[kNanIfaceConfBufSize];
		int ret = snprintf(cmd, kNanIfaceConfBufSize, MACSTR, MAC2STR(peerDiscMacAddr));
		if (ret > 0 && ret < kNanIfaceConfBufSize && wpas_nan_pairing_abort(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		}
		aidl_manager->notifyNanTerminatePairingResponse(ifname, cmdId, nan_status);
	});

	return ndk::ScopedAStatus::ok();
}

static int appendSecurityConfigToCmd(
	char* cmd, size_t cmd_size, size_t cnt,
	const NanDataPathSecurityConfig& security_config)
{
	cnt += snprintf(
		cmd + cnt, cmd_size - cnt,
		" csid=%d", convertNanCipherSuiteTypeToSupplicantCipherSuiteType(
			security_config.cipherType)
	);
	if (security_config.securityType == NanDataPathSecurityType::PASSPHRASE) {
		cnt += snprintf(cmd + cnt, cmd_size - cnt,
			" password=%s", std::string(security_config.passphrase.begin(),
			security_config.passphrase.end()).c_str());
	} else if (security_config.securityType == NanDataPathSecurityType::PMK) {
		std::string pmk_str = bytesToHexString(security_config.pmk.data(), security_config.pmk.size());
		cnt += snprintf(cmd + cnt, cmd_size - cnt,
			" pmk=%s", pmk_str.c_str());
	}
	return cnt;
}

::ndk::ScopedAStatus NanIface::initiateDataPathRequestInternal(
	char16_t cmdId, const NanInitiateDataPathRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INVALID_ARGS;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanInitiateDataPathResponse(ifname, cmdId,
				nan_status, msg.peerId);
			return;
		}
		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			"handle=%d ndi=%s peer_nmi=" MACSTR " peer_id=%d",
			msg.discoverySessionId, msg.ifaceName.c_str(),
			MAC2STR(msg.peerDiscMacAddr), msg.peerId);
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiateDataPathResponse(
				ifname, cmdId, nan_status, msg.peerId);
			return;
		}
		if (msg.appInfo.size() > 0) {
			std::string app_info_str = vectorToHexString(msg.appInfo);
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt,
				" ssi=%s", app_info_str.c_str());
			if (cnt < 0 || cnt >= sizeof(cmd)) {
				aidl_manager->notifyNanInitiateDataPathResponse(
					ifname, cmdId, nan_status, msg.peerId);
				return;
			}
		}
		cnt = appendSecurityConfigToCmd(
			cmd, kNanIfaceConfBufSize, cnt, msg.securityConfig);
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanInitiateDataPathResponse(
				ifname, cmdId, nan_status, msg.peerId);
			return;
		}
		if (wpas_nan_ndp_request(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		} else {
			nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		}
		aidl_manager->notifyNanInitiateDataPathResponse(ifname, cmdId,
			nan_status, msg.peerId);
	});
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::respondToDataPathIndicationRequestInternal(
	char16_t cmdId, const NanRespondToDataPathIndicationRequest& msg)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INVALID_ARGS;

		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanRespondToDataPathIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}
		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
			"%s ndi=%s handle=%d ndp_id=%d peer_nmi=" MACSTR " init_ndi=" MACSTR,
			msg.acceptRequest ? "accept" : "reject",
			msg.ifaceName.c_str(), msg.discoverySessionId, msg.ndpInstanceId,
			MAC2STR(msg.peerDiscMacAddr), MAC2STR(msg.ndiInitMac));
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanRespondToDataPathIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}
		if (msg.appInfo.size() > 0) {
			std::string app_info_str = vectorToHexString(msg.appInfo);
			cnt += snprintf(cmd + cnt, kNanIfaceConfBufSize - cnt,
				" ssi=%s", app_info_str.c_str());
			if (cnt < 0 || cnt >= sizeof(cmd)) {
				aidl_manager->notifyNanRespondToDataPathIndicationResponse(
					ifname, cmdId, nan_status);
				return;
			}
		}
		cnt = appendSecurityConfigToCmd(
			cmd, kNanIfaceConfBufSize, cnt, msg.securityConfig);
		if (cnt < 0 || cnt >= sizeof(cmd)) {
			aidl_manager->notifyNanRespondToDataPathIndicationResponse(
				ifname, cmdId, nan_status);
			return;
		}

		if (wpas_nan_ndp_response(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		} else {
			nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		}
		aidl_manager->notifyNanRespondToDataPathIndicationResponse(
			ifname, cmdId, nan_status);
	});
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::terminateDataPathRequestInternal(
	char16_t cmdId, int32_t ndpInstanceId, const std::array<uint8_t, 6>& peerDiscMacAddr,
	const std::array<uint8_t, 6>& in_ndiInitMac)
{
	AidlManager* aidl_manager = AidlManager::getInstance();
	if (!aidl_manager) {
		return createStatus(SupplicantStatusCode::FAILURE_UNKNOWN);
	}

	enqueue([=, ifname = ifname_, wpa_global = wpa_global_] {
		NanStatus nan_status;
		nan_status.status = NanStatusCode::INTERNAL_FAILURE;
		struct wpa_supplicant* wpa_s =
			wpa_supplicant_get_iface(wpa_global, ifname.c_str());
		if (!wpa_s) {
			nan_status.status = NanStatusCode::NAN_NOT_ALLOWED;
			aidl_manager->notifyNanTerminateDataPathResponse(ifname, cmdId, nan_status);
			return;
		}

		char cmd[kNanIfaceConfBufSize];
		int cnt = snprintf(cmd, kNanIfaceConfBufSize,
				"peer_nmi=" MACSTR " ndp_id=%d" " init_ndi=" MACSTR,
				MAC2STR(peerDiscMacAddr), ndpInstanceId, MAC2STR(in_ndiInitMac));

		if (cnt < 0 || cnt >= sizeof(cmd)) {
			nan_status.status = NanStatusCode::INVALID_ARGS;
			aidl_manager->notifyNanTerminateDataPathResponse(ifname, cmdId, nan_status);
			return;
		}
		if (wpas_nan_ndp_terminate(wpa_s, cmd) >= 0) {
			nan_status.status = NanStatusCode::SUCCESS;
		}
		aidl_manager->notifyNanTerminateDataPathResponse(ifname, cmdId, nan_status);
	});
	return ndk::ScopedAStatus::ok();
}

::ndk::ScopedAStatus NanIface::setScheduleInternal(
	char16_t in_cmdId,
	const std::vector<NanSchedule>& in_schedule)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}
#else
::ndk::ScopedAStatus NanIface::registerEventCallbackInternal(
	const std::shared_ptr<ISupplicantNanIfaceEventCallback>& callback)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::getCapabilitiesRequestInternal(char16_t cmdId)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::configRequestInternal(
	char16_t cmdId, const NanConfigRequest& request)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::enableRequestInternal(
	char16_t cmdId, const NanEnableRequest& msg1,
	const NanConfigRequest& msg2)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::disableRequestInternal(char16_t cmdId)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::createDataInterfaceRequestInternal(
	char16_t cmdId, const std::string& ifaceName, const std::array<uint8_t, 6>& macAddr)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::deleteDataInterfaceRequestInternal(
	char16_t cmdId, const std::string& ifaceName)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::startPublishRequestInternal(
	char16_t cmdId, const NanPublishRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::stopPublishRequestInternal(
	char16_t cmdId, int8_t sessionId)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::startSubscribeRequestInternal(
	char16_t cmdId, const NanSubscribeRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::stopSubscribeRequestInternal(
	char16_t cmdId, int8_t sessionId)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::transmitFollowupRequestInternal(
	char16_t cmdId, const NanTransmitFollowupRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::initiateBootstrappingRequestInternal(
	char16_t cmdId, const NanBootstrappingRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::respondToBootstrappingIndicationRequestInternal(
	char16_t cmdId, const NanBootstrappingResponse& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::initiatePairingRequestInternal(
	char16_t cmdId, const NanPairingRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::respondToPairingIndicationRequestInternal(
	char16_t cmdId, const NanRespondToPairingIndicationRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::terminatePairingRequestInternal(
	char16_t cmdId, int32_t pairingInstanceId,
	const std::array<uint8_t, 6>& peerDiscMacAddr)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::initiateDataPathRequestInternal(
	char16_t cmdId, const NanInitiateDataPathRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::respondToDataPathIndicationRequestInternal(
	char16_t cmdId, const NanRespondToDataPathIndicationRequest& msg)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::terminateDataPathRequestInternal(
	char16_t cmdId, int32_t ndpInstanceId,
	const std::array<uint8_t, 6>& peerDiscMacAddr,
	const std::array<uint8_t, 6>& in_ndiInitMac)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

::ndk::ScopedAStatus NanIface::setScheduleInternal(
	char16_t in_cmdId,
	const std::vector<NanSchedule>& in_schedule)
{
	return createStatus(SupplicantStatusCode::FAILURE_UNSUPPORTED);
}

#endif

bool NanIface::isStartedClusterIndicationEnabled()
{
	return started_cluster_indication_;
}

bool NanIface::isJoinedClusterIndicationEnabled()
{
	return joined_cluster_indication_;
}

bool NanIface::isDiscoveryTerminationIndicationEnabled()
{
	return discovery_termination_indication_;
}

bool NanIface::isMatchExpirationIndicationEnabled()
{
	return match_expiration_indication_;
}

bool NanIface::isFollowupReceivedIndicationEnabled()
{
	return followup_received_indication_;
}
