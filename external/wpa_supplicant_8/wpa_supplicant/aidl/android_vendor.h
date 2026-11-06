/*
 * WPA Supplicant - Google Android OUI and vendor specific assignments
 * Copyright (c) 2025, Google Inc. All rights reserved.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */
#ifndef ANDROID_VENDOR_H
#define ANDROID_VENDOR_H

/*
 * This file contains the attributes used for Android-specific vendor commands
 * and events within the Linux Wireless system. These commands and events are
 * designed to interact with and control specialized features implemented in
 * wireless chips, extending beyond the standard functionalities available
 * through the generic nl80211 interface.
 *
 * Every event and command detailed here is associated with the Google
 * Organizationally Unique Identifier (OUI) F4:F5:E8
 */

#define ANDROID_OUI 0xF4F5E8

/* Subcommands */
#define ANDROID_NL80211_SUBCMD_GET_PWRSTATS 1

/**
 * @def WPOWER_STATS_MAX_CORES
 * Maximum number of Wi-Fi cores for which power statistics can be reported.
 */
#define POWER_STATS_MAX_NUM_CORES 4
/**
 * @def POWER_STATS_MAX_NUM_RX_LISTENING_LEVELS
 * Maximum number of distinct RX listening power levels tracked.
 */
#define POWER_STATS_MAX_NUM_RX_LISTENING_LEVELS 16
/**
 * @def POWER_STATS_MAX_NUM_SLEEP_LEVELS
 * Maximum number of distinct sleep power levels tracked.
 */
#define POWER_STATS_MAX_NUM_SLEEP_LEVELS 4


/**
 * @enum WIFI_PWR_STAT_ATTRIBUTE - Attributes used in the nl80211 vendor command
 * for getting power statistics.
 */
typedef enum {
	POWER_STATS_ATTRIBUTE_INVALID = 0,
	POWER_STATS_ATTRIBUTE_NUM_WIFI_CORE = 1,
	POWER_STATS_ATTRIBUTE_RADIO_STATS = 2,
	POWER_STATS_ATTRIBUTE_TX_RATE_INFO = 3,
	POWER_STATS_ATTRIBUTE_RX_RATE_INFO = 4,
	POWER_STATS_ATTRIBUTE_CHIP_POWER_STATS = 5,
	POWER_STATS_ATTRIBUTE_STATS_MAX = 6
} POWER_STAT_ATTRIBUTE;


/**
 * @struct wifi_rate_info - Single rate entry with core index and direction.
 * This structure is the element type for the top-level Flexible Array Member (FAM).
 */
typedef struct {
    u32 rate_index;     // Data rate index (CCK/OFDM, MCS)
    u32 band;           // Operating band (0: 2.4 GHz, 1: 5 GHz, etc.)
    u32 bw;             // Bandwidth (20, 40, 80 MHz, etc.)
    u32 nss;            // Number of spatial streams
    u32 count;          // Number of times this rate was used.
} wifi_rate_info;


/**
 * @struct wifi_tx_rate_stats - TX rate statistics.
 */
typedef struct {
    u32 core_index;
    u32 num_rates;
    wifi_rate_info rates[];
} wifi_tx_rate_stats;


/**
 * @struct wifi_rx_rate_stats - RX rate statistics.
 */
typedef struct {
    u32 core_index;
    u32 num_rates;
    wifi_rate_info rates[];
} wifi_rx_rate_stats;


/**
 * @struct wifi_chip_power_state - Overall power state of the Wi-Fi chip.
 */
typedef struct {
    u32 wlan_pwr_on_time;       // Total time the chip is powered on (unit: ms).
    u32 sleep_levels_num;       // Number of sleep levels reported
    u32 sleep_time_per_levels[POWER_STATS_MAX_NUM_SLEEP_LEVELS]; // Time spent in each level (unit: ms).
} wifi_chip_power_state;


/**
 * @struct wifi_core_radio_stats - Radio statistics for a single Wi-Fi core.
 */
typedef struct {
    u32 core_index;                             // Index of the core.
    u32 radio_on_time;                          // Duration radio is on (unit: ms).
    u32 tx_time;                                // Duration in transmit state (unit: ms).
    u32 rx_time;                                // Duration in receiving state (unit: ms).

    u32 rx_listening_levels_num;                // Number of rx listening levels
    u32 rx_listening_time_per_levels[POWER_STATS_MAX_NUM_RX_LISTENING_LEVELS]; // Rx listening duration in each level (unit: ms).
} wifi_core_radio_stats;

#endif // ANDROID_VENDOR_H
