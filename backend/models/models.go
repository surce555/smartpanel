package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	Bookmarks []Bookmark `json:"bookmarks,omitempty"`
}

type Bookmark struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Icon              string       `json:"icon"`
	URLInternal       string       `json:"url_internal"`
	URLPublicTemplate string       `json:"url_public_template"`
	URLFallback       string       `json:"url_fallback"`
	GroupID           string       `json:"group_id"`
	Description       string       `json:"description"`
	OpenInNewTab      bool         `json:"open_in_new_tab"`
	IsPrivate         bool         `json:"is_private"`
	SortOrder         int          `json:"sort_order"`
	CreatedAt         time.Time    `json:"created_at"`
	Tags              []Tag        `json:"tags,omitempty"`
	TagIDs            []string     `json:"tag_ids,omitempty"`
	HealthCheck       *HealthCheck `json:"health_check,omitempty"`
}

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type BookmarkTag struct {
	BookmarkID string `json:"bookmark_id"`
	TagID      string `json:"tag_id"`
}

type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HealthCheck struct {
	ID             string     `json:"id"`
	BookmarkID     string     `json:"bookmark_id"`
	Status         string     `json:"status"` // 'online', 'offline', 'unknown'
	LastChecked    *time.Time `json:"last_checked"`
	ResponseTimeMS int        `json:"response_time_ms"`
}

// Request and Response DTOs
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type ReorderItem struct {
	ID        string `json:"id"`
	SortOrder int    `json:"sort_order"`
	GroupID   string `json:"group_id,omitempty"`
}

type NetworkInfo struct {
	CurrentIPv6      string `json:"current_ipv6"`
	DetectedHostIPv6 string `json:"detected_host_ipv6,omitempty"`
	Domain           string `json:"domain"`
	V6Domain         string `json:"v6domain"`
	DDNSEnabled      bool   `json:"ddns_enabled"`
	DDNSInterval     int    `json:"ddns_interval_minutes"`
	DDNSStatus       string `json:"ddns_status"` // 'idle', 'updating', 'success', 'failed'
	LastIPv6Check    string `json:"last_ipv6_check"`
	HasCFTunnel      bool   `json:"has_cf_tunnel"`
	CFZoneID         string `json:"cf_zone_id,omitempty"`
	CFRecordID       string `json:"cf_record_id,omitempty"`
	HasAPIToken      bool   `json:"has_api_token"`
}

type NetworkSettingsUpdateRequest struct {
	CFAPIToken          string `json:"cf_api_token"`
	CFZoneID            string `json:"cf_zone_id"`
	CFRecordID          string `json:"cf_record_id"`
	Domain              string `json:"domain"`
	V6Domain            string `json:"v6domain"`
	DDNSEnabled         bool   `json:"ddns_enabled"`
	DDNSIntervalMinutes int    `json:"ddns_interval_minutes"`
}

type SystemStatusResponse struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryTotalBytes   uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64  `json:"memory_used_bytes"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskTotalBytes     uint64  `json:"disk_total_bytes"`
	DiskUsedBytes      uint64  `json:"disk_used_bytes"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	HostOS             string  `json:"host_os"`
	HostArch           string  `json:"host_arch"`
	UptimeSeconds      uint64  `json:"uptime_seconds"`
}

type DockerContainerInfo struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Image   string   `json:"image"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
	Created int64    `json:"created"`
}
