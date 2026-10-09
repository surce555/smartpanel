package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/database"
)

type SettingsHandler struct{}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings, err := database.GetAllSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取系统设置失败", "code": http.StatusInternalServerError})
		return
	}

	// Security: Do not expose raw encrypted token, instead return flag
	encToken := settings["cf_api_token_enc"]
	hasToken := encToken != ""
	delete(settings, "cf_api_token_enc")
	settings["has_api_token"] = "false"
	if hasToken {
		settings["has_api_token"] = "true"
	}

	isAuth, _ := c.Get("is_authenticated")
	isAuthenticated, _ := isAuth.(bool)
	if !isAuthenticated {
		delete(settings, "cf_zone_id")
		delete(settings, "cf_record_id")
		delete(settings, "cf_email")
	}

	c.JSON(http.StatusOK, settings)
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var payload map[string]string
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误", "code": http.StatusBadRequest})
		return
	}

	for k, v := range payload {
		if k == "cf_api_token" {
			if v != "" {
				enc, err := config.Encrypt(v)
				if err == nil {
					_ = database.SetSetting("cf_api_token_enc", enc)
				}
			}
			continue
		}
		_ = database.SetSetting(k, v)
	}

	c.JSON(http.StatusOK, gin.H{"message": "设置已保存"})
}

func (h *SettingsHandler) GetThemeSettings(c *gin.Context) {
	themeKeys := []string{
		"theme_mode", "card_style", "card_border_radius", "card_shadow",
		"icon_size", "grid_cols_desktop", "grid_cols_tablet", "grid_cols_mobile",
		"wallpaper_type", "wallpaper_blur", "wallpaper_mask", "wallpaper_interval",
		"wallpaper_url", "wallpaper_custom_css", "custom_css", "custom_js",
		"login_wallpaper_type", "login_wallpaper_url", "login_wallpaper_blur", "login_wallpaper_mask",
		"font_family", "font_size", "title_font_family",
	}

	res := make(map[string]string)
	for _, k := range themeKeys {
		res[k] = database.GetSetting(k)
	}

	c.JSON(http.StatusOK, res)
}

func (h *SettingsHandler) UpdateThemeSettings(c *gin.Context) {
	var payload map[string]string
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误", "code": http.StatusBadRequest})
		return
	}

	for k, v := range payload {
		_ = database.SetSetting(k, v)
	}

	c.JSON(http.StatusOK, gin.H{"message": "外观设置已更新"})
}
