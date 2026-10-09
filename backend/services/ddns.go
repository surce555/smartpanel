package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"smartpanel/config"
	"smartpanel/database"
	"smartpanel/models"
)

type DDNSService struct {
	sseClients   map[chan models.NetworkInfo]bool
	clientsMu    sync.RWMutex
	checkTrigger chan struct{}
	stopChan     chan struct{}
}

var (
	DDNS                   *DDNSService
	lastCapturedServerIPv6 string
	serverIPMu             sync.RWMutex
)

func InitDDNSService() *DDNSService {
	DDNS = &DDNSService{
		sseClients:   make(map[chan models.NetworkInfo]bool),
		checkTrigger: make(chan struct{}, 1),
		stopChan:     make(chan struct{}),
	}

	go DDNS.worker()

	// Trigger immediate check on application startup
	DDNS.TriggerCheck()

	return DDNS
}

func RecordServerIPv6FromHost(host string) {
	if host == "" {
		return
	}
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	h = strings.Trim(h, "[]")
	ip := net.ParseIP(h)
	if ip != nil && ip.To4() == nil && isGlobalUnicastIPv6(ip) {
		serverIPMu.Lock()
		lastCapturedServerIPv6 = ip.String()
		serverIPMu.Unlock()
	}
}

func GetCapturedServerIPv6() string {
	serverIPMu.RLock()
	defer serverIPMu.RUnlock()
	return lastCapturedServerIPv6
}

func (s *DDNSService) TriggerCheck() {
	select {
	case s.checkTrigger <- struct{}{}:
	default:
	}
}

func (s *DDNSService) Subscribe() chan models.NetworkInfo {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	ch := make(chan models.NetworkInfo, 5)
	s.sseClients[ch] = true
	return ch
}

func (s *DDNSService) Unsubscribe(ch chan models.NetworkInfo) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	delete(s.sseClients, ch)
	close(ch)
}

func (s *DDNSService) broadcast(info models.NetworkInfo) {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	for ch := range s.sseClients {
		select {
		case ch <- info:
		default:
		}
	}
}

func (s *DDNSService) worker() {
	intervalMinutes := 5
	intervalStr := database.GetSetting("ddns_interval_minutes")
	if val, err := strconv.Atoi(intervalStr); err == nil && val > 0 {
		intervalMinutes = val
	}

	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.CheckAndUpdateIPv6("")
		case <-s.checkTrigger:
			s.CheckAndUpdateIPv6("")
		case <-s.stopChan:
			return
		}
	}
}

// GetLocalInterfaceIPv6 finds the first valid public global unicast IPv6 on network interfaces
func GetLocalInterfaceIPv6() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
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
			if ip != nil && ip.To4() == nil && isGlobalUnicastIPv6(ip) {
				return ip.String()
			}
		}
	}
	return ""
}

// FetchPublicIPv6 probes external endpoints to get public IPv6 address
func FetchPublicIPv6() (string, error) {
	// 1. Try domestic and global fast IPv6 test endpoints
	endpoints := []string{
		"https://6.ipw.cn",                   // 国内高可用极速 IPv6 探针
		"https://speed.neu6.edu.cn/getIP.php", // 东北大学网络测速 IPv6 接口
		"https://v6.ident.me",
		"https://api64.ipify.org",
		"https://api6.ipify.org",
		"https://ipv6.icanhazip.com",
		"https://ifconfig.co/ip",
	}

	client := &http.Client{
		Timeout: 4 * time.Second,
	}

	for _, url := range endpoints {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		ipStr := strings.TrimSpace(string(body))
		ip := net.ParseIP(ipStr)
		if ip != nil && ip.To4() == nil {
			if isGlobalUnicastIPv6(ip) {
				return ipStr, nil
			}
		}
	}

	// 2. Check local network interfaces (e.g. host network mode)
	if localIP := GetLocalInterfaceIPv6(); localIP != "" {
		return localIP, nil
	}

	// 3. Check if server IPv6 was detected from incoming client Host header
	if capturedIP := GetCapturedServerIPv6(); capturedIP != "" {
		return capturedIP, nil
	}

	return "", fmt.Errorf("未能获取到有效的公网 IPv6 地址 (外部探针均不可达且本地未发现全局单播地址)")
}

func isGlobalUnicastIPv6(ip net.IP) bool {
	if len(ip) != 16 {
		return false
	}
	// 2000::/3 means first byte is 0x20 to 0x3F
	firstByte := ip[0]
	return firstByte >= 0x20 && firstByte <= 0x3F
}

func (s *DDNSService) CheckAndUpdateIPv6(manualIPv6 string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	_ = database.SetSetting("last_ipv6_check", now)

	var ipv6 string
	var err error

	if manualIPv6 != "" {
		cleaned := strings.Trim(strings.TrimSpace(manualIPv6), "[]")
		ip := net.ParseIP(cleaned)
		if ip != nil && ip.To4() == nil && isGlobalUnicastIPv6(ip) {
			ipv6 = ip.String()
		} else {
			err = fmt.Errorf("手动指定的 IPv6 地址格式不正确")
		}
	} else {
		ipv6, err = FetchPublicIPv6()
	}

	if err != nil {
		log.Printf("[DDNS] IPv6 检测失败: %v", err)
		_ = database.SetSetting("ddns_status", "detect_failed: "+err.Error())
		s.notifyCurrentNetwork()
		return
	}

	prevIPv6 := database.GetSetting("current_ipv6")
	_ = database.SetSetting("current_ipv6", ipv6)

	ddnsEnabled := database.GetSetting("ddns_enabled") == "true"
	if ddnsEnabled && ipv6 != prevIPv6 {
		log.Printf("[DDNS] 检测到 IPv6 发生变更: 旧=%s, 新=%s", prevIPv6, ipv6)
		cfErr := updateCloudflareDNSRecord(ipv6)
		if cfErr != nil {
			log.Printf("[DDNS] 更新 Cloudflare DNS 失败: %v", cfErr)
			_ = database.SetSetting("ddns_status", "cf_error: "+cfErr.Error())
		} else {
			log.Printf("[DDNS] Cloudflare AAAA 记录成功更新为: %s", ipv6)
			_ = database.SetSetting("ddns_status", "success")
		}
	} else if ddnsEnabled {
		_ = database.SetSetting("ddns_status", "success (ip unchanged)")
	} else {
		_ = database.SetSetting("ddns_status", "ddns_disabled")
	}

	s.notifyCurrentNetwork()
}

func (s *DDNSService) notifyCurrentNetwork() {
	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}

	info := models.NetworkInfo{
		CurrentIPv6:      database.GetSetting("current_ipv6"),
		DetectedHostIPv6: GetCapturedServerIPv6(),
		Domain:           database.GetSetting("domain"),
		V6Domain:         database.GetSetting("v6domain"),
		DDNSEnabled:      database.GetSetting("ddns_enabled") == "true",
		DDNSInterval:     interval,
		DDNSStatus:       database.GetSetting("ddns_status"),
		LastIPv6Check:    database.GetSetting("last_ipv6_check"),
		CFZoneID:         database.GetSetting("cf_zone_id"),
		CFRecordID:       database.GetSetting("cf_record_id"),
		HasAPIToken:      database.GetSetting("cf_api_token_enc") != "",
	}
	s.broadcast(info)
}

func updateCloudflareDNSRecord(newIPv6 string) error {
	encToken := database.GetSetting("cf_api_token_enc")
	if encToken == "" {
		return fmt.Errorf("Cloudflare API Token 未配置")
	}

	token, err := config.Decrypt(encToken)
	if err != nil {
		return fmt.Errorf("解密 Cloudflare Token 失败: %w", err)
	}

	zoneID := database.GetSetting("cf_zone_id")
	recordID := database.GetSetting("cf_record_id")
	v6Domain := database.GetSetting("v6domain")

	if zoneID == "" || recordID == "" {
		return fmt.Errorf("Zone ID 或 Record ID 未配置")
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)

	payload := map[string]interface{}{
		"type":    "AAAA",
		"name":    v6Domain,
		"content": newIPv6,
		"ttl":     120,
		"proxied": false, // 小黄云必须关闭 (仅 DNS)
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("CF API 错误 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var cfResp struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &cfResp); err == nil && !cfResp.Success {
		errMsg := "unknown CF error"
		if len(cfResp.Errors) > 0 {
			errMsg = cfResp.Errors[0].Message
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}
