package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"smartpanel/database"
	"smartpanel/models"
	"smartpanel/services"
)

type NetworkHandler struct{}

func (h *NetworkHandler) GetNetworkInfo(c *gin.Context) {
	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}

	info := models.NetworkInfo{
		CurrentIPv6:   database.GetSetting("current_ipv6"),
		Domain:        database.GetSetting("domain"),
		V6Domain:      database.GetSetting("v6domain"),
		DDNSEnabled:   database.GetSetting("ddns_enabled") == "true",
		DDNSInterval:  interval,
		DDNSStatus:    database.GetSetting("ddns_status"),
		LastIPv6Check: database.GetSetting("last_ipv6_check"),
		CFZoneID:      database.GetSetting("cf_zone_id"),
		CFRecordID:    database.GetSetting("cf_record_id"),
		HasAPIToken:   database.GetSetting("cf_api_token_enc") != "",
	}

	c.JSON(http.StatusOK, info)
}

func (h *NetworkHandler) CheckNetwork(c *gin.Context) {
	// Trigger immediate check synchronously or trigger background
	go services.DDNS.CheckAndUpdateIPv6()

	c.JSON(http.StatusOK, gin.H{
		"message": "已触发 IPv6 检测与 DDNS 同步任务",
	})
}

func (h *NetworkHandler) StreamNetworkUpdates(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	ch := services.DDNS.Subscribe()
	defer services.DDNS.Unsubscribe(ch)

	// Send initial state immediately
	interval, _ := strconv.Atoi(database.GetSetting("ddns_interval_minutes"))
	if interval <= 0 {
		interval = 5
	}
	initialInfo := models.NetworkInfo{
		CurrentIPv6:   database.GetSetting("current_ipv6"),
		Domain:        database.GetSetting("domain"),
		V6Domain:      database.GetSetting("v6domain"),
		DDNSEnabled:   database.GetSetting("ddns_enabled") == "true",
		DDNSInterval:  interval,
		DDNSStatus:    database.GetSetting("ddns_status"),
		LastIPv6Check: database.GetSetting("last_ipv6_check"),
		CFZoneID:      database.GetSetting("cf_zone_id"),
		CFRecordID:    database.GetSetting("cf_record_id"),
		HasAPIToken:   database.GetSetting("cf_api_token_enc") != "",
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
