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

func isSameSubnetIPv6(clientIP net.IP) bool {
	if clientIP == nil || clientIP.To4() != nil || clientIP.IsLoopback() {
		return false
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	client16 := clientIP.To16()
	if client16 == nil {
		return false
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
			if ip == nil || ip.To4() != nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			host16 := ip.To16()
			if host16 != nil && len(host16) == 16 && len(client16) == 16 {
				// Compare first 64 bits (8 bytes) of IPv6 address
				match := true
				for i := 0; i < 8; i++ {
					if host16[i] != client16[i] {
						match = false
						break
					}
				}
				if match {
					return true
				}
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

func evaluateLANEnvironment(c *gin.Context) (bool, string, []string, string) {
	rawClientIP := c.ClientIP()
	clientIP := net.ParseIP(rawClientIP)

	remoteHost, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	var remoteIP net.IP
	if err == nil {
		remoteIP = net.ParseIP(remoteHost)
	}

	isLAN := false
	if clientIP != nil && (isPrivateOrLocalIP(clientIP) || isSameSubnetIPv6(clientIP)) {
		isLAN = true
	} else if remoteIP != nil && (isPrivateOrLocalIP(remoteIP) || isSameSubnetIPv6(remoteIP)) {
		isLAN = true
	}

	// Check if request Host is a LAN IP or local domain
	hostOnly := c.Request.Host
	if h, _, err := net.SplitHostPort(hostOnly); err == nil {
		hostOnly = h
	}
	if hostIP := net.ParseIP(hostOnly); hostIP != nil && isPrivateOrLocalIP(hostIP) {
		isLAN = true
	}
	if strings.HasSuffix(hostOnly, ".local") || strings.HasSuffix(hostOnly, ".lan") || strings.HasSuffix(hostOnly, ".home.arpa") {
		isLAN = true
	}

	lanDomain := database.GetSetting("lan_domain")
	if lanDomain != "" && (hostOnly == lanDomain || strings.HasSuffix(hostOnly, "."+lanDomain)) {
		isLAN = true
	}

	hostLANIPs := getHostLANIPs()
	port := config.AppConfig.Port
	if port == "" {
		port = "5050"
	}

	return isLAN, rawClientIP, hostLANIPs, port
}

func (h *NetworkHandler) GetNetworkInfo(c *gin.Context) {
	// Record incoming Host header to detect server's IPv6 address
	services.RecordServerIPv6FromHost(c.Request.Host)

	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}

	isLAN, clientIPStr, hostIPs, srvPort := evaluateLANEnvironment(c)

	info := models.NetworkInfo{
		CurrentIPv6:      database.GetSetting("current_ipv6"),
		DetectedHostIPv6: services.GetCapturedServerIPv6(),
		Domain:           database.GetSetting("domain"),
		V6Domain:         database.GetSetting("v6domain"),
		LANDomain:        database.GetSetting("lan_domain"),
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
	isLAN, clientIPStr, hostIPs, srvPort := evaluateLANEnvironment(c)

	initialInfo := models.NetworkInfo{
		CurrentIPv6:      database.GetSetting("current_ipv6"),
		DetectedHostIPv6: services.GetCapturedServerIPv6(),
		Domain:           database.GetSetting("domain"),
		V6Domain:         database.GetSetting("v6domain"),
		LANDomain:        database.GetSetting("lan_domain"),
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
