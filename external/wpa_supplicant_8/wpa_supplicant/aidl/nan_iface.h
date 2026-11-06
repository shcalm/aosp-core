/*
 * WPA Supplicant - Nan Iface Aidl interface
 * Copyright (c) 2025, Google Inc. All rights reserved.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */

#ifndef SUPPLICANT_NAN_IFACE_H
#define SUPPLICANT_NAN_IFACE_H

#include <aidl/android/hardware/wifi/supplicant/SupplicantStatusCode.h>
#include <aidl/android/system/wifi/mainline_supplicant/BnSupplicantNanIface.h>

#include <condition_variable>
#include <mutex>
#include <queue>
#include <thread>

extern "C"
{
#include "utils/common.h"
#include "utils/includes.h"
#include "wpa_supplicant_i.h"
}

using aidl::android::system::wifi::mainline_supplicant::BnSupplicantNanIface;

class NanIface : public BnSupplicantNanIface
{
public:
	using ISupplicantNanIfaceEventCallback =
		::aidl::android::system::wifi::mainline_supplicant::ISupplicantNanIfaceEventCallback;
	using NanConfigRequest = ::aidl::android::system::wifi::mainline_supplicant::NanConfigRequest;
	using NanEnableRequest = ::aidl::android::system::wifi::mainline_supplicant::NanEnableRequest;
	using NanPublishRequest = ::aidl::android::system::wifi::mainline_supplicant::NanPublishRequest;
	using NanSubscribeRequest = ::aidl::android::system::wifi::mainline_supplicant::NanSubscribeRequest;
	using NanTransmitFollowupRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanTransmitFollowupRequest;
	using NanBootstrappingRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanBootstrappingRequest;
	using NanBootstrappingResponse =
		::aidl::android::system::wifi::mainline_supplicant::NanBootstrappingResponse;
	using NanPairingRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanPairingRequest;
	using NanRespondToPairingIndicationRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanRespondToPairingIndicationRequest;
	using NanInitiateDataPathRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanInitiateDataPathRequest;
	using NanRespondToDataPathIndicationRequest =
		::aidl::android::system::wifi::mainline_supplicant::NanRespondToDataPathIndicationRequest;
	using NanSchedule =
		::aidl::android::system::wifi::mainline_supplicant::NanSchedule;
	NanIface(struct wpa_global* global, const std::string& ifname);
	~NanIface() override;

	// Refer to |StaIface::invalidate()|.
	void invalidate();
	bool isValid();
	void enqueue(std::function<void()> task);
	bool isStartedClusterIndicationEnabled();
	bool isJoinedClusterIndicationEnabled();
	bool isDiscoveryTerminationIndicationEnabled();
	bool isMatchExpirationIndicationEnabled();
	bool isFollowupReceivedIndicationEnabled();

	::ndk::ScopedAStatus registerEventCallback(
		const std::shared_ptr<ISupplicantNanIfaceEventCallback>&
		in_callback) override;
	::ndk::ScopedAStatus getCapabilitiesRequest(char16_t in_cmdId) override;
	::ndk::ScopedAStatus configRequest(
		char16_t in_cmdId, const NanConfigRequest& in_request) override;
	::ndk::ScopedAStatus enableRequest(
		char16_t in_cmdId, const NanEnableRequest& in_msg1,
		const NanConfigRequest& in_msg2) override;
	::ndk::ScopedAStatus disableRequest(char16_t in_cmdId) override;
	::ndk::ScopedAStatus createDataInterfaceRequest(
		char16_t, const std::string& in_ifaceName,
		const std::array<uint8_t, 6>& macAddr) override;
	::ndk::ScopedAStatus deleteDataInterfaceRequest(
		char16_t, const std::string& in_ifaceName) override;
	::ndk::ScopedAStatus getName(std::string* _aidl_return) override;
	::ndk::ScopedAStatus startPublishRequest(
		char16_t in_cmdId, const NanPublishRequest& in_msg) override;
	::ndk::ScopedAStatus stopPublishRequest(
		char16_t in_cmdId, int8_t in_sessionId) override;
	::ndk::ScopedAStatus startSubscribeRequest(
		char16_t in_cmdId, const NanSubscribeRequest& in_msg) override;
	::ndk::ScopedAStatus stopSubscribeRequest(
		char16_t in_cmdId, int8_t in_sessionId) override;
	::ndk::ScopedAStatus transmitFollowupRequest(
		char16_t in_cmdId,
		const NanTransmitFollowupRequest& in_msg) override;
	::ndk::ScopedAStatus initiateBootstrappingRequest(
		char16_t in_cmdId,
		const NanBootstrappingRequest& in_msg) override;
	::ndk::ScopedAStatus respondToBootstrappingIndicationRequest(
		char16_t in_cmdId,
		const NanBootstrappingResponse& in_msg) override;
	::ndk::ScopedAStatus initiatePairingRequest(
		char16_t in_cmdId,
		const NanPairingRequest& in_msg) override;
	::ndk::ScopedAStatus respondToPairingIndicationRequest(
		char16_t in_cmdId,
		const NanRespondToPairingIndicationRequest& in_msg) override;
	::ndk::ScopedAStatus terminatePairingRequest(
		char16_t in_cmdId,
		int32_t in_pairingInstanceId,
		const std::array<uint8_t, 6>& in_peerDiscMacAddr) override;
	::ndk::ScopedAStatus initiateDataPathRequest(
		char16_t in_cmdId,
		const NanInitiateDataPathRequest& in_msg) override;
	::ndk::ScopedAStatus respondToDataPathIndicationRequest(
		char16_t in_cmdId,
		const NanRespondToDataPathIndicationRequest& in_msg) override;
	::ndk::ScopedAStatus terminateDataPathRequest(
		char16_t in_cmdId,
		int32_t in_ndpInstanceId,
		const std::array<uint8_t, 6>& in_peerDiscMacAddr,
		const std::array<uint8_t, 6>& in_ndiInitMac) override;
	::ndk::ScopedAStatus setSchedule(
		char16_t in_cmdId,
		const std::vector<NanSchedule>& in_schedule) override;

private:
	::ndk::ScopedAStatus registerEventCallbackInternal(
		const std::shared_ptr<ISupplicantNanIfaceEventCallback>&
		in_callback);
	::ndk::ScopedAStatus getCapabilitiesRequestInternal(
		char16_t cmdId);
	::ndk::ScopedAStatus configRequestInternal(
		char16_t cmdId, const NanConfigRequest& request);
	::ndk::ScopedAStatus enableRequestInternal(
		char16_t cmdId, const NanEnableRequest& msg1,
		const NanConfigRequest& msg2);
	::ndk::ScopedAStatus disableRequestInternal(char16_t cmdId);
	::ndk::ScopedAStatus createDataInterfaceRequestInternal(
		char16_t, const std::string& ifaceName,
		const std::array<uint8_t, 6>& macAddr);
	::ndk::ScopedAStatus deleteDataInterfaceRequestInternal(
		char16_t, const std::string& ifaceName);
	::ndk::ScopedAStatus startPublishRequestInternal(
		char16_t cmdId, const NanPublishRequest& msg);
	::ndk::ScopedAStatus stopPublishRequestInternal(
		char16_t cmdId, int8_t sessionId);
	::ndk::ScopedAStatus startSubscribeRequestInternal(
		char16_t cmdId, const NanSubscribeRequest& msg);
	::ndk::ScopedAStatus stopSubscribeRequestInternal(
		char16_t cmdId, int8_t sessionId);
	::ndk::ScopedAStatus transmitFollowupRequestInternal(
		char16_t cmdId, const NanTransmitFollowupRequest& msg);
	::ndk::ScopedAStatus initiateBootstrappingRequestInternal(
		char16_t cmdId, const NanBootstrappingRequest& msg);
	::ndk::ScopedAStatus respondToBootstrappingIndicationRequestInternal(
		char16_t cmdId, const NanBootstrappingResponse& msg);
	::ndk::ScopedAStatus initiatePairingRequestInternal(
		char16_t cmdId, const NanPairingRequest& msg);
	::ndk::ScopedAStatus respondToPairingIndicationRequestInternal(
		char16_t cmdId,
		const NanRespondToPairingIndicationRequest& msg);
	::ndk::ScopedAStatus terminatePairingRequestInternal(
		char16_t cmdId, int32_t pairingInstanceId,
		const std::array<uint8_t, 6>& peerDiscMacAddr);
	::ndk::ScopedAStatus initiateDataPathRequestInternal(
		char16_t cmdId, const NanInitiateDataPathRequest& msg);
	::ndk::ScopedAStatus respondToDataPathIndicationRequestInternal(
		char16_t cmdId,
		const NanRespondToDataPathIndicationRequest& msg);
	::ndk::ScopedAStatus terminateDataPathRequestInternal(
		char16_t cmdId, int32_t ndpInstanceId,
		const std::array<uint8_t, 6>& peerDiscMacAddr,
		const std::array<uint8_t, 6>& in_ndiInitMac);
	::ndk::ScopedAStatus setScheduleInternal(
		char16_t in_cmdId,
		const std::vector<NanSchedule>& in_schedule);

	struct wpa_global* wpa_global_;
	// Name of the iface this aidl object controls
	const std::string ifname_;
	std::thread worker_thread_;
	std::mutex mutex_;
	std::condition_variable cv_;
	std::queue<std::function<void()>> task_queue_;
	bool stop_worker_ = false;
	bool is_valid_;
	bool started_cluster_indication_;
	bool joined_cluster_indication_;
	bool discovery_termination_indication_;
	bool match_expiration_indication_;
	bool followup_received_indication_;
};

#endif	// SUPPLICANT_NAN_IFACE_H
