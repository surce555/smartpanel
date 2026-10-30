package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/database"
	"smartpanel/models"
	"smartpanel/services"
)

type NetworkHandler struct{}

func isPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// Check ULA (Unique Local Address fc00::/7)
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}

func isSameSubnetIPv6(ip1, ip2 net.IP) bool {
	if ip1 == nil || ip2 == nil {
		return false
	}
	ip1_16 := ip1.To16()
	ip2_16 := ip2.To16()
	if ip1_16 == nil || ip2_16 == nil {
		return false
	}
	if ip1.To4() != nil || ip2.To4() != nil {
		return false
	}
	if !services.IsGlobalUnicastIPv6(ip1) || !services.IsGlobalUnicastIPv6(ip2) {
		return false
	}
	// Compare first 64 bits (first 8 bytes)
	for i := 0; i < 8; i++ {
		if ip1_16[i] != ip2_16[i] {
			return false
		}
	}
	return true
}

func isCGNAT(ip net.IP) bool {
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	return v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127
}

func isSameOrNearbyIPv4(ip1, ip2 net.IP) bool {
	if ip1 == nil || ip2 == nil {
		return false
	}
	v4_1 := ip1.To4()
	v4_2 := ip2.To4()
	if v4_1 == nil || v4_2 == nil {
		return false
	}
	if v4_1.Equal(v4_2) {
		return true
	}
	if isCGNAT(v4_1) && isCGNAT(v4_2) {
		return v4_1[0] == v4_2[0] && v4_1[1] == v4_2[1] && v4_1[2] == v4_2[2]
	}
	return false
}

func isIPInWhitelist(ip net.IP, whitelistStr string) bool {
	if ip == nil || strings.TrimSpace(whitelistStr) == "" {
		return false
	}
	entries := strings.Split(whitelistStr, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err == nil && ipNet.Contains(ip) {
				return true
			}
		} else {
			target := net.ParseIP(entry)
			if target != nil && target.Equal(ip) {
				return true
			}
		}
	}
	return false
}

func getHostLANIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil && ip.IsPrivate() && !ip.IsLoopback() {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}

// isProxiedByCloudflare reports whether this request reached us through the
// Cloudflare edge (Tunnel / proxied DNS) rather than a direct connection.
//
// This matters for routing: behind Cloudflare, `CF-Connecting-IP` carries the
// visitor's public address, so an IPv6 value there does NOT prove the client
// reached the NAS directly over IPv6 — Cloudflare could have proxied an IPv4
// client, or an IPv6 client over the tunnel. Only a direct connection proves
// the client can reach the NAS over IPv6.
func isProxiedByCloudflare(c *gin.Context) bool {
	if c.GetHeader("CF-Ray") != "" || c.GetHeader("CF-Connecting-IP") != "" {
		return true
	}
	if c.GetHeader("CDN-Loop") != "" {
		return true
	}
	return false
}

// resolveClientIP returns the address of the browser, not of whatever hop
// proxied the request to us.
//
// The router is configured with `SetTrustedProxies(nil)`, so `c.ClientIP()` only
// ever returns the TCP peer. Behind the Cloudflare Tunnel that peer is
// Cloudflare's own edge node, so every visitor looked like a public IP and
// `is_client_lan` was permanently false — which silently disabled LAN routing
// even when browsing from inside the house.
//
// When the edge tells us who the visitor really is (CF-Connecting-IP), use it:
// it is the only way to classify the client correctly through a tunnel. Without
// that header (direct access, cloudflared running with --no-tls-verify and no
// edge) fall back to the peer, which is then the real client.
func resolveClientIP(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); v != "" {
		if net.ParseIP(v) != nil {
			return v
		}
	}
	// X-Forwarded-For: first entry is the original client.
	if v := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); v != "" {
		first := strings.TrimSpace(strings.Split(v, ",")[0])
		if net.ParseIP(first) != nil {
			return first
		}
	}
	if v := strings.TrimSpace(c.GetHeader("X-Real-IP")); v != "" {
		if net.ParseIP(v) != nil {
			return v
		}
	}
	return c.ClientIP()
}

func evaluateLANEnvironment(c *gin.Context) (bool, string, []string, string, string) {
	rawClientIP := resolveClientIP(c)
	clientIP := net.ParseIP(rawClientIP)

	// Whether this request was relayed by the Cloudflare edge. Matters below: a
	// tunnel terminates on the NAS itself, so the TCP peer is loopback.
	proxied := isProxiedByCloudflare(c)

	remoteHost, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	var remoteIP net.IP
	if err == nil {
		remoteIP = net.ParseIP(remoteHost)
	}

	isLAN := false
	lanReason := ""

	// 1. Direct private or local loopback visitor IP
	if clientIP != nil && isPrivateOrLocalIP(clientIP) {
		isLAN = true
		lanReason = "客户端私网 IP 直连 (" + rawClientIP + ")"
	} else if !proxied && remoteIP != nil && isPrivateOrLocalIP(remoteIP) {
		isLAN = true
		lanReason = "局域网对端直连 (" + remoteIP.String() + ")"
	}

	// 2. IP Whitelist check (user configured in settings)
	whitelist := database.GetSetting("lan_ip_whitelist")
	if !isLAN && clientIP != nil && isIPInWhitelist(clientIP, whitelist) {
		isLAN = true
		lanReason = "匹配内网 IP 白名单 (" + rawClientIP + ")"
	}

	// 3. IPv6 Subnet matching (same /64 home broadband prefix)
	if !isLAN && clientIP != nil && clientIP.To4() == nil && services.IsGlobalUnicastIPv6(clientIP) {
		allV6 := services.GetAllServerIPv6s()
		for _, v6Str := range allV6 {
			nasV6 := net.ParseIP(v6Str)
			if nasV6 != nil && isSameSubnetIPv6(clientIP, nasV6) {
				isLAN = true
				lanReason = "与 NAS 处于同一 IPv6 /64 子网 (" + rawClientIP + ")"
				break
			}
		}
	}

	// 4. IPv4 Egress matching (same home NAT router egress)
	if !isLAN && clientIP != nil && clientIP.To4() != nil {
		serverIPv4 := services.GetCapturedServerIPv4()
		if serverIPv4 == "" {
			serverIPv4 = database.GetSetting("server_public_ipv4")
		}
		if serverIPv4 != "" {
			sIP := net.ParseIP(serverIPv4)
			if sIP != nil && isSameOrNearbyIPv4(clientIP, sIP) {
				isLAN = true
				lanReason = "与 NAS 共享相同的家庭公网出口 IPv4 (" + serverIPv4 + ")"
			}
		}
	}

	// 5. Check if request Host is a LAN IP or local domain
	hostOnly := c.Request.Host
	if h, _, err := net.SplitHostPort(hostOnly); err == nil {
		hostOnly = h
	}
	if !isLAN {
		if hostIP := net.ParseIP(hostOnly); hostIP != nil && isPrivateOrLocalIP(hostIP) {
			isLAN = true
			lanReason = "通过内网 IP 访问 (" + hostOnly + ")"
		} else if strings.HasSuffix(hostOnly, ".local") || strings.HasSuffix(hostOnly, ".lan") || strings.HasSuffix(hostOnly, ".home.arpa") {
			isLAN = true
			lanReason = "通过本地域名访问 (" + hostOnly + ")"
		} else {
			lanDomain := database.GetSetting("lan_domain")
			if lanDomain != "" {
				cleanLan := strings.TrimPrefix(strings.TrimSpace(lanDomain), "*.")
				if hostOnly == cleanLan || strings.HasSuffix(hostOnly, "."+cleanLan) {
					isLAN = true
					lanReason = "匹配自定义内网域名 (" + lanDomain + ")"
				}
			}
		}
	}

	hostLANIPs := getHostLANIPs()
	port := config.AppConfig.Port
	if port == "" {
		port = "5050"
	}

	return isLAN, rawClientIP, hostLANIPs, port, lanReason
}

func (h *NetworkHandler) GetNetworkInfo(c *gin.Context) {
	// Record incoming Host header to detect server's IPv6 address
	services.RecordServerIPv6FromHost(c.Request.Host)

	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}

	isLAN, clientIPStr, hostIPs, srvPort, lanReason := evaluateLANEnvironment(c)

	// Whether this page was loaded over a direct (non-proxied) connection, and
	// whether that direct connection was IPv6.
	viaCloudflare := isProxiedByCloudflare(c)
	remoteHost, _, splitErr := net.SplitHostPort(c.Request.RemoteAddr)
	directIPv6 := false
	if splitErr == nil && !viaCloudflare {
		if rip := net.ParseIP(remoteHost); rip != nil && rip.To4() == nil {
			directIPv6 = true
		}
	}

	// "The client has IPv6 connectivity at all", which is a SEPARATE question
	// from "the client reached the NAS over IPv6 just now".
	//
	// Behind the Cloudflare edge the visitor's real address arrives in
	// CF-Connecting-IP. If that address is IPv6, the browser demonstrably has a
	// working IPv6 stack (it used IPv6 to reach Cloudflare). That is exactly the
	// signal the frontend needs to decide whether to hand out `{ipv6}` links —
	// and it is far more reliable than probing third-party hosts from the page,
	// which fails whenever the user's proxy/VPN has no IPv6 egress even though
	// the NAS itself is reachable over IPv6.
	clientHasIPv6 := directIPv6
	if !clientHasIPv6 {
		if cip := net.ParseIP(clientIPStr); cip != nil && cip.To4() == nil && !isPrivateOrLocalIP(cip) {
			clientHasIPv6 = true
		}
	}

	info := models.NetworkInfo{
		CurrentIPv6:      database.GetSetting("current_ipv6"),
		DetectedHostIPv6: services.GetCapturedServerIPv6(),
		Domain:           database.GetSetting("domain"),
		V6Domain:         database.GetSetting("v6domain"),
		LANDomain:        database.GetSetting("lan_domain"),
		ServerPublicIPv4: database.GetSetting("server_public_ipv4"),
		LANIPWhitelist:   database.GetSetting("lan_ip_whitelist"),
		LANReason:        lanReason,
		DDNSEnabled:      database.GetSetting("ddns_enabled") == "true",
		DDNSInterval:     interval,
		DDNSStatus:       database.GetSetting("ddns_status"),
		LastIPv6Check:    database.GetSetting("last_ipv6_check"),
		CFZoneID:         database.GetSetting("cf_zone_id"),
		CFRecordID:       database.GetSetting("cf_record_id"),
		HasAPIToken:      database.GetSetting("cf_api_token_enc") != "",
		IsClientLAN:      isLAN,
		ClientIP:         clientIPStr,
		HostLANIPs:       hostIPs,
		ServerPort:       srvPort,
		ViaCloudflare:    viaCloudflare,
		DirectIPv6:       directIPv6,
		ClientHasIPv6:    clientHasIPv6,
	}

	c.JSON(http.StatusOK, info)
}

func (h *NetworkHandler) CheckNetwork(c *gin.Context) {
	services.RecordServerIPv6FromHost(c.Request.Host)

	var req struct {
		ManualIPv6 string `json:"manual_ipv6"`
	}
	_ = c.ShouldBindJSON(&req)

	go services.DDNS.CheckAndUpdateIPv6(req.ManualIPv6)

	c.JSON(http.StatusOK, gin.H{
		"message": "已触发 IPv6 检测与 DDNS 同步任务",
	})
}

func (h *NetworkHandler) StreamNetworkUpdates(c *gin.Context) {
	services.RecordServerIPv6FromHost(c.Request.Host)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	ch := services.DDNS.Subscribe()
	defer services.DDNS.Unsubscribe(ch)

	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}
	isLAN, clientIPStr, hostIPs, srvPort, lanReason := evaluateLANEnvironment(c)

	viaCloudflare := isProxiedByCloudflare(c)
	remoteHostSSE, _, sseSplitErr := net.SplitHostPort(c.Request.RemoteAddr)
	directIPv6 := false
	if sseSplitErr == nil && !viaCloudflare {
		if rip := net.ParseIP(remoteHostSSE); rip != nil && rip.To4() == nil {
			directIPv6 = true
		}
	}
	clientHasIPv6 := directIPv6
	if !clientHasIPv6 {
		if cip := net.ParseIP(clientIPStr); cip != nil && cip.To4() == nil && !isPrivateOrLocalIP(cip) {
			clientHasIPv6 = true
		}
	}

	initialInfo := models.NetworkInfo{
		CurrentIPv6:      database.GetSetting("current_ipv6"),
		DetectedHostIPv6: services.GetCapturedServerIPv6(),
		Domain:           database.GetSetting("domain"),
		V6Domain:         database.GetSetting("v6domain"),
		LANDomain:        database.GetSetting("lan_domain"),
		ServerPublicIPv4: database.GetSetting("server_public_ipv4"),
		LANIPWhitelist:   database.GetSetting("lan_ip_whitelist"),
		LANReason:        lanReason,
		DDNSEnabled:      database.GetSetting("ddns_enabled") == "true",
		DDNSInterval:     interval,
		DDNSStatus:       database.GetSetting("ddns_status"),
		LastIPv6Check:    database.GetSetting("last_ipv6_check"),
		CFZoneID:         database.GetSetting("cf_zone_id"),
		CFRecordID:       database.GetSetting("cf_record_id"),
		HasAPIToken:      database.GetSetting("cf_api_token_enc") != "",
		IsClientLAN:      isLAN,
		ClientIP:         clientIPStr,
		HostLANIPs:       hostIPs,
		ServerPort:       srvPort,
		ViaCloudflare:    viaCloudflare,
		DirectIPv6:       directIPv6,
		ClientHasIPv6:    clientHasIPv6,
	}
	dataBytes, _ := json.Marshal(initialInfo)
	_, _ = fmt.Fprintf(c.Writer, "event: network_update\ndata: %s\n\n", string(dataBytes))
	c.Writer.Flush()

	c.Stream(func(w io.Writer) bool {
		select {
		case info, ok := <-ch:
			if !ok {
				return false
			}
			info.IsClientLAN = isLAN
			info.ClientIP = clientIPStr
			info.HostLANIPs = hostIPs
			info.ServerPort = srvPort
			info.LANDomain = database.GetSetting("lan_domain")
			info.ServerPublicIPv4 = database.GetSetting("server_public_ipv4")
			info.LANIPWhitelist = database.GetSetting("lan_ip_whitelist")
			info.LANReason = lanReason
			// Keep the per-connection facts on every push. The broadcast payload
			// originates from the DDNS service and does not know them; without
			// this the frontend would see them reset to false mid-session.
			info.ViaCloudflare = viaCloudflare
			info.DirectIPv6 = directIPv6
			info.ClientHasIPv6 = clientHasIPv6
			bytes, err := json.Marshal(info)
			if err != nil {
				return true
			}
			_, _ = fmt.Fprintf(w, "event: network_update\ndata: %s\n\n", string(bytes))
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
