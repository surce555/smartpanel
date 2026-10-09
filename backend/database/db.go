package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"smartpanel/config"
	"smartpanel/models"
)

var DB *sql.DB

func InitDB() error {
	var err error
	dbPath := config.AppConfig.DBPath

	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite performance & concurrency
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := DB.Exec(pragma); err != nil {
			log.Printf("Warning executing pragma %s: %v", pragma, err)
		}
	}

	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := seedDefaultData(); err != nil {
		return fmt.Errorf("failed to seed default data: %w", err)
	}

	log.Printf("SQLite database initialized successfully at: %s", dbPath)
	return nil
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		icon TEXT,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS bookmarks (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		icon TEXT,
		url_internal TEXT,
		url_public_template TEXT,
		url_fallback TEXT,
		group_id TEXT REFERENCES groups(id) ON DELETE SET NULL,
		description TEXT,
		open_in_new_tab BOOLEAN DEFAULT 1,
		is_private BOOLEAN DEFAULT 0,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tags (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		color TEXT DEFAULT '#6366f1'
	);

	CREATE TABLE IF NOT EXISTS bookmark_tags (
		bookmark_id TEXT REFERENCES bookmarks(id) ON DELETE CASCADE,
		tag_id TEXT REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (bookmark_id, tag_id)
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS health_checks (
		id TEXT PRIMARY KEY,
		bookmark_id TEXT REFERENCES bookmarks(id) ON DELETE CASCADE,
		status TEXT CHECK(status IN ('online', 'offline', 'unknown')),
		last_checked DATETIME,
		response_time_ms INTEGER
	);
	`

	_, err := DB.Exec(schema)
	return err
}

func seedDefaultData() error {
	// Seed default admin user
	var userCount int
	err := DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return err
	}

	if userCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		adminID := uuid.New().String()
		_, err = DB.Exec(
			"INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)",
			adminID, "admin@smartpanel.local", string(hash),
		)
		if err != nil {
			return err
		}

		// Mark first login must change password
		_ = SetSetting("must_change_password", "true")
		log.Println("Default admin user created: admin@smartpanel.local / admin123")
	}

	// Seed default settings if not exists
	defaultSettings := map[string]string{
		"theme_mode":            "system", // light, dark, system
		"card_style":            "glass",  // glass, solid, transparent, minimal
		"card_border_radius":    "12",     // px
		"card_shadow":           "md",     // none, sm, md, lg
		"icon_size":             "medium", // small, medium, large, detail
		"grid_cols_desktop":     "6",      // 4-8
		"grid_cols_tablet":      "3",
		"grid_cols_mobile":      "2",
		"wallpaper_type":        "gradient", // none, upload, gradient, unsplash
		"wallpaper_blur":        "0",
		"wallpaper_mask":        "20",
		"wallpaper_interval":    "60",     // seconds
		"search_default_engine": "bing",   // bing, google, baidu, custom
		"search_custom_url":     "https://www.google.com/search?q=%s",
		"footer_text":           "SmartPanel - NAS Navigation Dashboard",
		"show_clock":            "true",
		"clock_format_24":       "true",
		"custom_css":            "",
		"custom_js":             "",
		"ddns_enabled":          "false",
		"ddns_interval_minutes": "5",
		"current_ipv6":          "",
		"last_ipv6_check":       "",
		"domain":                "",
		"v6domain":              "",
		"cf_zone_id":            "",
		"cf_record_id":          "",
		"ddns_status":           "idle",
	}

	for k, v := range defaultSettings {
		var exists int
		_ = DB.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", k).Scan(&exists)
		if exists == 0 {
			_ = SetSetting(k, v)
		}
	}

	// Seed default sample group and bookmarks if groups is empty
	var groupCount int
	_ = DB.QueryRow("SELECT COUNT(*) FROM groups").Scan(&groupCount)
	if groupCount == 0 {
		groupID1 := uuid.New().String()
		_, _ = DB.Exec("INSERT INTO groups (id, name, icon, sort_order) VALUES (?, ?, ?, ?)", groupID1, "常用服务", "tabler:star", 0)

		groupID2 := uuid.New().String()
		_, _ = DB.Exec("INSERT INTO groups (id, name, icon, sort_order) VALUES (?, ?, ?, ?)", groupID2, "NAS 与存储", "tabler:server", 1)

		sampleBookmarks := []models.Bookmark{
			{
				ID:                uuid.New().String(),
				Name:              "SmartPanel 管理",
				Icon:              "tabler:dashboard",
				URLInternal:       "http://localhost:5666",
				URLPublicTemplate: "https://[{ipv6}]:5666",
				URLFallback:       "https://{domain}",
				GroupID:           groupID1,
				Description:       "SmartPanel 控制面板管理后台",
				OpenInNewTab:      true,
				IsPrivate:         false,
				SortOrder:         0,
			},
			{
				ID:                uuid.New().String(),
				Name:              "路由器管理",
				Icon:              "tabler:router",
				URLInternal:       "http://192.168.1.1",
				URLPublicTemplate: "",
				URLFallback:       "",
				GroupID:           groupID2,
				Description:       "家庭网关与局域网管理",
				OpenInNewTab:      true,
				IsPrivate:         true,
				SortOrder:         0,
			},
		}

		for _, b := range sampleBookmarks {
			_, _ = DB.Exec(`
				INSERT INTO bookmarks (id, name, icon, url_internal, url_public_template, url_fallback, group_id, description, open_in_new_tab, is_private, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, b.ID, b.Name, b.Icon, b.URLInternal, b.URLPublicTemplate, b.URLFallback, b.GroupID, b.Description, b.OpenInNewTab, b.IsPrivate, b.SortOrder)
		}
	}

	return nil
}

func GetSetting(key string) string {
	var val string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

func SetSetting(key, value string) error {
	_, err := DB.Exec(`
		INSERT INTO settings (key, value, updated_at) 
		VALUES (?, ?, ?) 
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now())
	return err
}

func GetAllSettings() (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			settings[k] = v
		}
	}
	return settings, nil
}
