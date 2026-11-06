/*
 * WPA Supplicant - Helper methods for Nan Interface.
 * Copyright (c) 2025, Google Inc. All rights reserved.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */

#ifndef MAINLINE_SUPPLICANT_NAN_UTILS_H
#define MAINLINE_SUPPLICANT_NAN_UTILS_H

#include "misc_utils.h"
#include "nan_supplicant.h"
#include "src/common/nan_de.h"
#include <aidl/android/system/wifi/mainline_supplicant/NanPublishRequest.h>
#include <aidl/android/system/wifi/mainline_supplicant/NanSubscribeRequest.h>
#include <aidl/android/system/wifi/mainline_supplicant/NanDiscoveryCommonConfig.h>
#include <aidl/android/system/wifi/mainline_supplicant/NanCapabilities.h>
#include <aidl/android/system/wifi/mainline_supplicant/NanPairingSecurityConfig.h>
#include <aidl/android/system/wifi/mainline_supplicant/NanCipherSuiteType.h>

namespace aidl {
namespace android {
namespace system {
namespace wifi {
namespace mainline_supplicant {
namespace nan_utils {

constexpr int32_t kMaxNanServiceNameLengthBytes = 255;
constexpr int32_t kMaxNanMatchFilterLengthBytes = 255;
constexpr int32_t kMaxNanServiceSpecificInfoLen = 255;
using aidl::android::hardware::wifi::supplicant::misc_utils::convertVectorToWpaBuf;

template <typename T>
inline bool checkContainerSize(const T& container, int maxSize)
{
	return container.size() <= maxSize;
}

template <typename T>
inline bool isValidEnumValue(T value, T enumRangeMin, T enumRangeMax)
{
	return static_cast<uint32_t>(value) >=
		   static_cast<uint32_t>(enumRangeMin) &&
	       static_cast<uint32_t>(value) <=
		   static_cast<uint32_t>(enumRangeMax);
}

inline bool validateNanBaseConfig(NanDiscoveryCommonConfig baseConfig)
{
	if (!checkContainerSize(
		baseConfig.serviceName, kMaxNanServiceNameLengthBytes)) {
		return false;
	}
	if (!checkContainerSize(
		baseConfig.serviceSpecificInfo,
		kMaxNanServiceSpecificInfoLen)) {
		return false;
	}
	if (!baseConfig.txMatchFilter.empty() &&
	    !checkContainerSize(
		baseConfig.txMatchFilter, kMaxNanMatchFilterLengthBytes)) {
		return false;
	}
	if (!baseConfig.rxMatchFilter.empty() &&
	    !checkContainerSize(
		baseConfig.rxMatchFilter, kMaxNanMatchFilterLengthBytes)) {
		return false;
	}
	return true;
}

inline bool validateNanPublishConfig(NanPublishRequest publishRequest)
{
	if (!validateNanBaseConfig(publishRequest.baseConfig)) {
		return false;
	}

	if (!isValidEnumValue(
		publishRequest.publishType, NanPublishRequest::NanPublishType::UNSOLICITED,
		NanPublishRequest::NanPublishType::SOLICITED)) {
		return false;
	}

	if (!isValidEnumValue(
		publishRequest.txType, NanPublishRequest::NanTxType::BROADCAST,
		NanPublishRequest::NanTxType::UNICAST)) {
		return false;
	}
	return true;
}

inline bool validateNanSubscribeConfig(NanSubscribeRequest subscribeRequest)
{
	if (!validateNanBaseConfig(subscribeRequest.baseConfig)) {
		return false;
	}

	if (!isValidEnumValue(
		subscribeRequest.subscribeType, NanSubscribeRequest::NanSubscribeType::PASSIVE,
		NanSubscribeRequest::NanSubscribeType::ACTIVE)) {
		return false;
	}
	return true;
}

inline struct nan_publish_params convertAidlNanPublishConfigToInternal(
    const NanPublishRequest& in_msg)
{
	struct nan_publish_params params{};
	params.unsolicited = in_msg.publishType == NanPublishRequest::NanPublishType::UNSOLICITED
		|| in_msg.publishType == NanPublishRequest::NanPublishType::UNSOLICITED_SOLICITED;
	params.solicited = in_msg.publishType == NanPublishRequest::NanPublishType::SOLICITED
		|| in_msg.publishType == NanPublishRequest::NanPublishType::UNSOLICITED_SOLICITED;
	params.solicited_multicast =
	    params.solicited && in_msg.txType == NanPublishRequest::NanTxType::BROADCAST;
	params.ttl = in_msg.baseConfig.ttlSec;
	params.disable_events =
	    in_msg.baseConfig.disableDiscoveryTerminationIndication &&
	    in_msg.baseConfig.disableMatchExpirationIndication &&
	    in_msg.baseConfig.disableFollowupReceivedIndication;
	params.fsd = true;  // further service discovery is necessary
	params.fsd_gas = false;
	params.freq = 0;  // NAN sync discovery reuses current cluster's channel
	params.freq_list = NULL;
	params.announcement_period = 0;	 // 0 = use default
	params.sync = true;
	params.close_proximity = in_msg.baseConfig.useRssiThreshold;

	return params;
}

inline struct nan_subscribe_params convertAidlNanSubscribeConfigToInternal(
    const NanSubscribeRequest& in_msg)
{
	struct nan_subscribe_params params{};
	params.active = in_msg.subscribeType == NanSubscribeRequest::NanSubscribeType::ACTIVE;
	params.ttl = in_msg.baseConfig.ttlSec;
	params.freq = 0;  // NAN sync discovery reuses current cluster's channel
	params.freq_list = NULL;  // NAN sync discovery doesn't need set
	params.query_period = 0;
	params.sync = true;
	params.srf_include = false;
	params.srf_mac_list = NULL;
	params.srf_bf_len = 0;
	params.srf_bf_idx = 0;
	params.close_proximity = in_msg.baseConfig.useRssiThreshold;

	return params;
}

inline NanCapabilities::RttBw convertIntegerToRttBw(int32_t bandwidth)
{
	switch (bandwidth) {
	case 5:
		return NanCapabilities::RttBw::BW_5MHZ;
	case 10:
		return NanCapabilities::RttBw::BW_10MHZ;
	case 20:
		return NanCapabilities::RttBw::BW_20MHZ;
	case 40:
		return NanCapabilities::RttBw::BW_40MHZ;
	case 80:
		return NanCapabilities::RttBw::BW_80MHZ;
	case 160:
		return NanCapabilities::RttBw::BW_160MHZ;
	case 320:
		return NanCapabilities::RttBw::BW_320MHZ;
	default:
		return NanCapabilities::RttBw::BW_UNSPECIFIED;
	}
}

inline int convertNanPairingSecurityTypeToInteger(
	NanPairingSecurityConfig::NanPairingSecurityType security_type)
{
	switch (security_type) {
	case NanPairingSecurityConfig::NanPairingSecurityType::OPPORTUNISTIC:
		return 0;
	case NanPairingSecurityConfig::NanPairingSecurityType::PASSPHRASE:
		return 1;
	case NanPairingSecurityConfig::NanPairingSecurityType::PMK:
		return 2;
	default:
		return -1;
	}
}

inline int convertNanCipherSuiteTypeToSupplicantCipherSuiteType(
	NanCipherSuiteType cipher_suite)
{
	u16 cipher = static_cast<uint16_t>(cipher_suite);

	if (cipher == static_cast<uint16_t>(NanCipherSuiteType::NONE))
		return NAN_CS_NONE;
	/* check sequence based on cipher suite priority */
	/* NCS-PK-PASN-256 (using a password) or NCS-PK-2WDH-256 */
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::PUBLIC_KEY_PASN_256_MASK))
		return NAN_CS_PK_PASN_256;
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::PUBLIC_KEY_2WDH_256_MASK))
		return NAN_CS_PK_2WDH_256;

	/* NCS-PK-PASN-128 (using a password) or NCS-PK-2WDH-128 */
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::PUBLIC_KEY_PASN_128_MASK))
		return NAN_CS_PK_PASN_128;
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::PUBLIC_KEY_2WDH_128_MASK))
		return NAN_CS_PK_2WDH_128;

	/* NCS-SK-256 (using a PSK/Passphrase) */
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::SHARED_KEY_256_MASK))
		return NAN_CS_SK_GCM_256;

	/* NCS-SK-128 (using a PSK/Passphrase) */
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::SHARED_KEY_128_MASK))
		return NAN_CS_SK_CCM_128;
	
	/* Group key */
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::GROUP_KEY_GCMP_256_MASK))
		return NAN_CS_GTK_GCMP_256;
	if (cipher & static_cast<uint16_t>(NanCipherSuiteType::GROUP_KEY_CCMP_128_MASK))
		return NAN_CS_GTK_CCMP_128;

	return -1;
}

}  // namespace nan_utils
}  // namespace supplicant
}  // namespace wifi
}  // namespace hardware
}  // namespace android
}  // namespace aidl

#endif	// MAINLINE_SUPPLICANT_NAN_UTILS_H
