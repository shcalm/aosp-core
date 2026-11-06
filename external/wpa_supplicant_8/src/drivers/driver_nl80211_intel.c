#include <netlink/genl/genl.h>
#include <sys/types.h>
#include "includes.h"

#include "common.h"
#include "driver_nl80211.h"
#ifndef HOSTAPD
#include "wpa_supplicant_i.h"
#endif /* HOSTAPD */
#ifdef ANDROID
#include "android_drv.h"
#endif

struct wiphy_idx_data {
	int wiphy_idx;
	enum nl80211_iftype nlmode;
	u8* macaddr;
	u8 use_4addr;
};

static int netdev_info_handler(struct nl_msg* msg, void* arg) {
	struct nlattr* tb[NL80211_ATTR_MAX + 1];
	struct genlmsghdr* gnlh = nlmsg_data(nlmsg_hdr(msg));
	struct wiphy_idx_data* info = arg;

	nla_parse(tb, NL80211_ATTR_MAX, genlmsg_attrdata(gnlh, 0), genlmsg_attrlen(gnlh, 0), NULL);

	if (tb[NL80211_ATTR_WIPHY]) info->wiphy_idx = nla_get_u32(tb[NL80211_ATTR_WIPHY]);

	if (tb[NL80211_ATTR_IFTYPE]) info->nlmode = nla_get_u32(tb[NL80211_ATTR_IFTYPE]);

	if (tb[NL80211_ATTR_MAC] && info->macaddr)
		os_memcpy(info->macaddr, nla_data(tb[NL80211_ATTR_MAC]), ETH_ALEN);

	if (tb[NL80211_ATTR_4ADDR]) info->use_4addr = nla_get_u8(tb[NL80211_ATTR_4ADDR]);

	return NL_SKIP;
}

static int nl80211_get_macaddr(struct i802_bss* bss) {
	struct nl_msg* msg;
	struct wiphy_idx_data data = {
			.macaddr = bss->addr,
	};

	if (!(msg = nl80211_cmd_msg(bss, 0, NL80211_CMD_GET_INTERFACE))) return -1;

	return send_and_recv_resp(bss->drv, msg, netdev_info_handler, &data);
}

int wpa_driver_nl80211_driver_cmd_intel(void* priv, char* cmd, char* buf, size_t buf_len) {
	struct i802_bss* bss = priv;
	struct wpa_driver_nl80211_data* drv = bss->drv;
	int ret = 0;

	// The AL (Desktop) WiFi drivers do not currently support the following private commands.
	if (os_strncasecmp(cmd, "MIRACAST", 8) == 0 || os_strncasecmp(cmd, "BTCOEXMODE", 10) == 0 ||
		os_strncasecmp(cmd, "BTCOEXSCAN-START", 16) == 0 ||
		os_strncasecmp(cmd, "BTCOEXSCAN-STOP", 15) == 0 ||
		os_strncasecmp(cmd, "SETSUSPENDMODE", 14) == 0)
		return 0;

#ifndef HOSTAPD
	if (bss->ifindex <= 0 && bss->wdev_id > 0) {
		/* DRIVER CMD received on the DEDICATED P2P Interface which doesn't
		 * have a NETDEVICE associated with it. So we have to re-route the
		 * command to the parent NETDEVICE
		 */
		struct wpa_supplicant* wpa_s = (struct wpa_supplicant*)(drv->ctx);

		wpa_printf(MSG_DEBUG, "Re-routing DRIVER cmd to parent iface");
		if (wpa_s && wpa_s->parent) {
			/* Update the nl80211 pointers corresponding to parent iface */
			bss = wpa_s->parent->drv_priv;
			drv = bss->drv;
			wpa_printf(MSG_DEBUG,
					   "Re-routing command to iface: %s"
					   " cmd (%s)",
					   bss->ifname, cmd);
		}
	}
#endif /* HOSTAPD */

	if (os_strncasecmp(cmd, "COUNTRY", 7) == 0) {
		char alpha2[3];
		struct nl_msg* msg;
		msg = nlmsg_alloc();
		if (!msg) ret = -ENOMEM;

		// Example: cmd = "COUNTRY US"
		memcpy(alpha2, cmd + strlen("COUNTRY") + 1, strlen(cmd) - strlen("COUNTRY") - 1);
		alpha2[2] = '\0';
		/* Self-managed regulatory: firmware/driver controls regdom, cfg80211 inputs are ignored,
		 * therefore check the WPA_DRIVER_FLAGS_SELF_MANAGED_REGULATORY flag before calling the driver
		 * for reloading the regulatory db
		 */
		if (!(drv->capa.flags & WPA_DRIVER_FLAGS_SELF_MANAGED_REGULATORY)) {
			if (!nl80211_cmd(drv, msg, 0, NL80211_CMD_RELOAD_REGDB)) {
				wpa_printf(MSG_ERROR, "debug for fw reload\n");
				nlmsg_free(msg);
				return -EINVAL;
			}
			if (send_and_recv_msgs(drv, msg, NULL, NULL, NULL, NULL)) return -EINVAL;
		}

		msg = nlmsg_alloc();
		if (!msg) ret = -ENOMEM;

		if (!nl80211_cmd(drv, msg, 0, NL80211_CMD_REQ_SET_REG) ||
			nla_put_string(msg, NL80211_ATTR_REG_ALPHA2, alpha2)) {
			nlmsg_free(msg);
			return -EINVAL;
		}
		if (send_and_recv_msgs(drv, msg, NULL, NULL, NULL, NULL)) ret = -EINVAL;
	} else if (os_strcasecmp(cmd, "MACADDR") == 0) {
		ret = nl80211_get_macaddr(bss);
		if (!ret) ret = os_snprintf(buf, buf_len, "Macaddr = " MACSTR "\n", MAC2STR(bss->addr));
	}

	return ret;
}

int wpa_driver_set_p2p_noa_intel(void* priv, u8 count, int start, int duration) {
	return 0;
}

int wpa_driver_get_p2p_noa_intel(void* priv __unused, u8* buf __unused, size_t len __unused) {
	return 0;
}

int wpa_driver_set_p2p_ps_intel(void* priv, int legacy_ps, int opp_ps, int ctwindow) {
	return 0;
}

int wpa_driver_set_ap_wps_p2p_ie_intel(void* priv, const struct wpabuf* beacon,
								 const struct wpabuf* proberesp, const struct wpabuf* assocresp) {
	return 0;
}

const struct wpa_driver_nl80211_vendor_ops intel_vendor_ops = {
	.driver_cmd = wpa_driver_nl80211_driver_cmd_intel,
	.set_noa = wpa_driver_set_p2p_noa_intel,
	.get_noa = wpa_driver_get_p2p_noa_intel,
	.set_ps = wpa_driver_set_p2p_ps_intel,
	.set_ap_wps_ie = wpa_driver_set_ap_wps_p2p_ie_intel,
};
