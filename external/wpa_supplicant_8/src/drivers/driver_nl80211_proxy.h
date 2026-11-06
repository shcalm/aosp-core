/*
 * Proxy functions for nl80211 vendor driver commands
 * Copyright (c) 2025, Google Inc.
 *
 * This software may be distributed under the terms of the BSD license.
 * See README for more details.
 */

#ifndef DRIVER_NL80211_PROXY_H
#define DRIVER_NL80211_PROXY_H

#define PROXY_FUNC_BODY(op, fallback, ...)					\
	do {									\
		struct i802_bss *bss = priv;					\
		struct wpa_driver_nl80211_data *drv = bss ? bss->drv : NULL;	\
		if (drv && drv->vendor_ops && drv->vendor_ops->op) {		\
			wpa_printf(MSG_DEBUG, "nl80211: Dispatching " #op	\
				   " via vendor_ops (OUI 0x%x)",		\
				   drv->chip_vendor_id);			\
			return drv->vendor_ops->op(priv, ##__VA_ARGS__);	\
		}								\
		return fallback(priv, ##__VA_ARGS__);				\
	} while (0)

static inline int
wpa_driver_nl80211_driver_cmd_proxy(void *priv, char *cmd, char *buf,
				    size_t buf_len) {
	PROXY_FUNC_BODY(driver_cmd, wpa_driver_nl80211_driver_cmd, cmd, buf,
			buf_len);
}

static inline int
wpa_driver_set_p2p_noa_proxy(void *priv, u8 count, int start, int duration) {
	PROXY_FUNC_BODY(set_noa, wpa_driver_set_p2p_noa, count, start, duration);
}

static inline int
wpa_driver_get_p2p_noa_proxy(void *priv, u8 *buf, size_t buf_len) {
	PROXY_FUNC_BODY(get_noa, wpa_driver_get_p2p_noa, buf, buf_len);
}

static inline int
wpa_driver_set_p2p_ps_proxy(void *priv, int legacy_ps, int opp_ps, int ctwindow) {
	PROXY_FUNC_BODY(set_ps, wpa_driver_set_p2p_ps, legacy_ps, opp_ps,
			ctwindow);
}

static inline int
wpa_driver_set_ap_wps_p2p_ie_proxy(void *priv, const struct wpabuf *beacon,
				   const struct wpabuf *proberesp,
				   const struct wpabuf *assocresp) {
	PROXY_FUNC_BODY(set_ap_wps_ie, wpa_driver_set_ap_wps_p2p_ie, beacon,
			proberesp, assocresp);
}

#ifdef MAINLINE_SUPPLICANT
int wpa_driver_nl80211_driver_cmd(void *priv, char *cmd, char *buf,
				  size_t buf_len ) {
	return 0;
}

int wpa_driver_set_p2p_noa(void *priv, u8 count, int start, int duration) {
	return 0;
}

int wpa_driver_get_p2p_noa(void *priv, u8 *buf, size_t len) {
	return 0;
}

int wpa_driver_set_p2p_ps(void *priv, int legacy_ps, int opp_ps, int ctwindow) {
	return 0;
}

int wpa_driver_set_ap_wps_p2p_ie(void *priv, const struct wpabuf *beacon,
				 const struct wpabuf *proberesp,
				 const struct wpabuf *assocresp) {
	return 0;
}
#endif

#endif /* DRIVER_NL80211_PROXY_H */
