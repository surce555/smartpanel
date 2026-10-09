package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartpanel/database"
	"smartpanel/models"
)

type BookmarkHandler struct{}

func (h *BookmarkHandler) GetBookmarks(c *gin.Context) {
	isAuth, _ := c.Get("is_authenticated")
	isAuthenticated, _ := isAuth.(bool)

	if database.GetSetting("require_login") == "true" && !isAuthenticated {
		c.JSON(http.StatusOK, []models.Bookmark{})
		return
	}

	groupID := c.Query("group_id")
	tagID := c.Query("tag_id")
	search := c.Query("search")

	query := `
		SELECT b.id, b.name, b.icon, b.url_internal, b.url_public_template, b.url_fallback,
		       b.group_id, b.description, b.open_in_new_tab, b.is_private, b.sort_order, b.created_at,
		       hc.status, hc.last_checked, hc.response_time_ms
		FROM bookmarks b
		LEFT JOIN health_checks hc ON b.id = hc.bookmark_id
	`
	conditions := make([]string, 0)
	args := make([]interface{}, 0)

	if !isAuthenticated {
		conditions = append(conditions, "b.is_private = 0")
	}

	if groupID != "" {
		conditions = append(conditions, "b.group_id = ?")
		args = append(args, groupID)
	}

	if search != "" {
		conditions = append(conditions, "(b.name LIKE ? OR b.description LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	if tagID != "" {
		conditions = append(conditions, "b.id IN (SELECT bookmark_id FROM bookmark_tags WHERE tag_id = ?)")
		args = append(args, tagID)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY b.sort_order ASC, b.created_at ASC"

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取书签失败: " + err.Error(), "code": http.StatusInternalServerError})
		return
	}
	defer rows.Close()

	bookmarks := make([]models.Bookmark, 0)
	bookmarkIDs := make([]string, 0)

	for rows.Next() {
		var b models.Bookmark
		var icon, urlInternal, urlPublic, urlFallback, groupID, desc sqlNullString
		var hcStatus sqlNullString
		var hcLastChecked sql.NullTime
		var hcRespTime sql.NullInt64

		err := rows.Scan(
			&b.ID, &b.Name, &icon, &urlInternal, &urlPublic, &urlFallback,
			&groupID, &desc, &b.OpenInNewTab, &b.IsPrivate, &b.SortOrder, &b.CreatedAt,
			&hcStatus, &hcLastChecked, &hcRespTime,
		)
		if err != nil {
			continue
		}

		b.Icon = icon.String
		b.URLInternal = urlInternal.String
		b.URLPublicTemplate = urlPublic.String
		b.URLFallback = urlFallback.String
		b.GroupID = groupID.String
		b.Description = desc.String

		if hcStatus.Valid {
			var t *time.Time
			if hcLastChecked.Valid {
				t = &hcLastChecked.Time
			}
			b.HealthCheck = &models.HealthCheck{
				BookmarkID:     b.ID,
				Status:         hcStatus.String,
				LastChecked:    t,
				ResponseTimeMS: int(hcRespTime.Int64),
			}
		}

		bookmarks = append(bookmarks, b)
		bookmarkIDs = append(bookmarkIDs, b.ID)
	}

	// Fetch tags for all bookmarks
	tagMap := getTagsForBookmarks(bookmarkIDs)
	for i := range bookmarks {
		if tags, ok := tagMap[bookmarks[i].ID]; ok {
			bookmarks[i].Tags = tags
		} else {
			bookmarks[i].Tags = []models.Tag{}
		}
	}

	c.JSON(http.StatusOK, bookmarks)
}

func (h *BookmarkHandler) CreateBookmark(c *gin.Context) {
	var req struct {
		Name              string   `json:"name" binding:"required"`
		Icon              string   `json:"icon"`
		URLInternal       string   `json:"url_internal"`
		URLPublicTemplate string   `json:"url_public_template"`
		URLFallback       string   `json:"url_fallback"`
		GroupID           string   `json:"group_id"`
		Description       string   `json:"description"`
		OpenInNewTab      *bool    `json:"open_in_new_tab"`
		IsPrivate         *bool    `json:"is_private"`
		TagIDs            []string `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "书签名称不能为空", "code": http.StatusBadRequest})
		return
	}

	openInNewTab := true
	if req.OpenInNewTab != nil {
		openInNewTab = *req.OpenInNewTab
	}

	isPrivate := false
	if req.IsPrivate != nil {
		isPrivate = *req.IsPrivate
	}

	var maxSort int
	_ = database.DB.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM bookmarks WHERE group_id = ?", req.GroupID).Scan(&maxSort)

	bookmark := models.Bookmark{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Icon:              req.Icon,
		URLInternal:       req.URLInternal,
		URLPublicTemplate: req.URLPublicTemplate,
		URLFallback:       req.URLFallback,
		GroupID:           req.GroupID,
		Description:       req.Description,
		OpenInNewTab:      openInNewTab,
		IsPrivate:         isPrivate,
		SortOrder:         maxSort + 1,
		CreatedAt:         time.Now(),
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "事务创建失败", "code": http.StatusInternalServerError})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO bookmarks (id, name, icon, url_internal, url_public_template, url_fallback, group_id, description, open_in_new_tab, is_private, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, bookmark.ID, bookmark.Name, bookmark.Icon, bookmark.URLInternal, bookmark.URLPublicTemplate, bookmark.URLFallback,
		bookmark.GroupID, bookmark.Description, bookmark.OpenInNewTab, bookmark.IsPrivate, bookmark.SortOrder, bookmark.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存书签失败: " + err.Error(), "code": http.StatusInternalServerError})
		return
	}

	for _, tagID := range req.TagIDs {
		_, _ = tx.Exec("INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)", bookmark.ID, tagID)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, bookmark)
}

func (h *BookmarkHandler) UpdateBookmark(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name              string   `json:"name" binding:"required"`
		Icon              string   `json:"icon"`
		URLInternal       string   `json:"url_internal"`
		URLPublicTemplate string   `json:"url_public_template"`
		URLFallback       string   `json:"url_fallback"`
		GroupID           string   `json:"group_id"`
		Description       string   `json:"description"`
		OpenInNewTab      *bool    `json:"open_in_new_tab"`
		IsPrivate         *bool    `json:"is_private"`
		SortOrder         *int     `json:"sort_order"`
		TagIDs            []string `json:"tag_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "code": http.StatusBadRequest})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "事务失败", "code": http.StatusInternalServerError})
		return
	}
	defer tx.Rollback()

	query := `
		UPDATE bookmarks SET name = ?, icon = ?, url_internal = ?, url_public_template = ?, url_fallback = ?,
		                     group_id = ?, description = ?
	`
	args := []interface{}{req.Name, req.Icon, req.URLInternal, req.URLPublicTemplate, req.URLFallback, req.GroupID, req.Description}

	if req.OpenInNewTab != nil {
		query += ", open_in_new_tab = ?"
		args = append(args, *req.OpenInNewTab)
	}
	if req.IsPrivate != nil {
		query += ", is_private = ?"
		args = append(args, *req.IsPrivate)
	}
	if req.SortOrder != nil {
		query += ", sort_order = ?"
		args = append(args, *req.SortOrder)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	res, err := tx.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新书签失败: " + err.Error(), "code": http.StatusInternalServerError})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "书签不存在", "code": http.StatusNotFound})
		return
	}

	// Update tags
	_, _ = tx.Exec("DELETE FROM bookmark_tags WHERE bookmark_id = ?", id)
	for _, tagID := range req.TagIDs {
		_, _ = tx.Exec("INSERT OR IGNORE INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)", id, tagID)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (h *BookmarkHandler) DeleteBookmark(c *gin.Context) {
	id := c.Param("id")
	res, err := database.DB.Exec("DELETE FROM bookmarks WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除书签失败", "code": http.StatusInternalServerError})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "书签不存在", "code": http.StatusNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *BookmarkHandler) ReorderBookmarks(c *gin.Context) {
	var items []models.ReorderItem
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误", "code": http.StatusBadRequest})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动事务失败", "code": http.StatusInternalServerError})
		return
	}
	defer tx.Rollback()

	for _, item := range items {
		if item.GroupID != "" {
			_, err = tx.Exec("UPDATE bookmarks SET sort_order = ?, group_id = ? WHERE id = ?", item.SortOrder, item.GroupID, item.ID)
		} else {
			_, err = tx.Exec("UPDATE bookmarks SET sort_order = ? WHERE id = ?", item.SortOrder, item.ID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新书签排序失败", "code": http.StatusInternalServerError})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序更新成功"})
}

func (h *BookmarkHandler) ExportBookmarks(c *gin.Context) {
	format := c.DefaultQuery("format", "json")

	// Get all groups and bookmarks
	rows, err := database.DB.Query("SELECT id, name, icon, sort_order FROM groups ORDER BY sort_order ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询分组失败", "code": http.StatusInternalServerError})
		return
	}
	defer rows.Close()

	groups := make([]models.Group, 0)
	for rows.Next() {
		var g models.Group
		var icon sqlNullString
		_ = rows.Scan(&g.ID, &g.Name, &icon, &g.SortOrder)
		g.Icon = icon.String
		groups = append(groups, g)
	}

	// Fetch all bookmarks
	bRows, err := database.DB.Query(`
		SELECT id, name, icon, url_internal, url_public_template, url_fallback, group_id, description, open_in_new_tab, is_private, sort_order
		FROM bookmarks ORDER BY sort_order ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询书签失败", "code": http.StatusInternalServerError})
		return
	}
	defer bRows.Close()

	bookmarks := make([]models.Bookmark, 0)
	for bRows.Next() {
		var b models.Bookmark
		var icon, urlInternal, urlPublic, urlFallback, groupID, desc sqlNullString
		_ = bRows.Scan(&b.ID, &b.Name, &icon, &urlInternal, &urlPublic, &urlFallback, &groupID, &desc, &b.OpenInNewTab, &b.IsPrivate, &b.SortOrder)
		b.Icon = icon.String
		b.URLInternal = urlInternal.String
		b.URLPublicTemplate = urlPublic.String
		b.URLFallback = urlFallback.String
		b.GroupID = groupID.String
		b.Description = desc.String
		bookmarks = append(bookmarks, b)
	}

	if format == "html" {
		// Output Netscape bookmark HTML format
		var sb strings.Builder
		sb.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n")
		sb.WriteString("<!-- This is an automatically generated file. -->\n")
		sb.WriteString("<META HTTP-EQUIV=\"Content-Type\" CONTENT=\"text/html; charset=UTF-8\">\n")
		sb.WriteString("<TITLE>SmartPanel Bookmarks</TITLE>\n")
		sb.WriteString("<H1>SmartPanel Bookmarks</H1>\n")
		sb.WriteString("<DL><p>\n")

		groupMap := make(map[string][]models.Bookmark)
		for _, b := range bookmarks {
			groupMap[b.GroupID] = append(groupMap[b.GroupID], b)
		}

		for _, g := range groups {
			sb.WriteString(fmt.Sprintf("  <DT><H3 ADD_DATE=\"%d\">%s</H3>\n  <DL><p>\n", time.Now().Unix(), html.EscapeString(g.Name)))
			for _, b := range groupMap[g.ID] {
				targetURL := b.URLPublicTemplate
				if targetURL == "" {
					targetURL = b.URLInternal
				}
				if targetURL == "" {
					targetURL = b.URLFallback
				}
				sb.WriteString(fmt.Sprintf("    <DT><A HREF=\"%s\" ADD_DATE=\"%d\">%s</A>\n", html.EscapeString(targetURL), time.Now().Unix(), html.EscapeString(b.Name)))
			}
			sb.WriteString("  </DL><p>\n")
		}
		sb.WriteString("</DL><p>\n")

		c.Header("Content-Disposition", "attachment; filename=smartpanel-bookmarks.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(sb.String()))
		return
	}

	// JSON format
	c.Header("Content-Disposition", "attachment; filename=smartpanel-bookmarks.json")
	c.JSON(http.StatusOK, gin.H{
		"version":   "1.0",
		"timestamp": time.Now().Format(time.RFC3339),
		"groups":    groups,
		"bookmarks": bookmarks,
	})
}

func (h *BookmarkHandler) ImportBookmarks(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传有效的导入文件", "code": http.StatusBadRequest})
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取文件失败", "code": http.StatusBadRequest})
		return
	}

	contentStr := string(content)
	importedCount := 0

	// Check if Netscape HTML format
	if strings.Contains(strings.ToUpper(contentStr), "<!DOCTYPE NETSCAPE-BOOKMARK-FILE-1>") || strings.Contains(contentStr, "<H3") {
		importedCount, err = importHTMLBookmarks(contentStr)
	} else {
		// Try JSON format
		importedCount, err = importJSONBookmarks(content)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析导入数据失败: " + err.Error(), "code": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("成功导入 %d 个书签", importedCount),
		"imported_count": importedCount,
	})
}

func importHTMLBookmarks(htmlContent string) (int, error) {
	// Simple Netscape HTML parser extracting folders and links
	reFolder := regexp.MustCompile(`(?i)<H3[^>]*>(.*?)</H3>`)
	reLink := regexp.MustCompile(`(?i)<A\s+HREF="([^"]+)"[^>]*>(.*?)</A>`)

	defaultGroupID := uuid.New().String()
	_, _ = database.DB.Exec("INSERT OR IGNORE INTO groups (id, name, sort_order) VALUES (?, ?, ?)", defaultGroupID, "导入的书签", 999)

	matches := reLink.FindAllStringSubmatch(htmlContent, -1)
	count := 0

	for _, match := range matches {
		if len(match) >= 3 {
			url := strings.TrimSpace(match[1])
			name := html.UnescapeString(strings.TrimSpace(match[2]))
			if name == "" {
				name = url
			}

			bookmarkID := uuid.New().String()
			_, err := database.DB.Exec(`
				INSERT INTO bookmarks (id, name, url_internal, group_id, open_in_new_tab, is_private, sort_order)
				VALUES (?, ?, ?, ?, 1, 0, ?)
			`, bookmarkID, name, url, defaultGroupID, count)
			if err == nil {
				count++
			}
		}
	}

	_ = reFolder
	return count, nil
}

func importJSONBookmarks(data []byte) (int, error) {
	var payload struct {
		Groups    []models.Group    `json:"groups"`
		Bookmarks []models.Bookmark `json:"bookmarks"`
	}

	if err := json.Unmarshal(data, &payload); err == nil && (len(payload.Groups) > 0 || len(payload.Bookmarks) > 0) {
		for _, g := range payload.Groups {
			_, _ = database.DB.Exec("INSERT OR IGNORE INTO groups (id, name, icon, sort_order) VALUES (?, ?, ?, ?)", g.ID, g.Name, g.Icon, g.SortOrder)
		}
		count := 0
		for _, b := range payload.Bookmarks {
			_, err := database.DB.Exec(`
				INSERT OR REPLACE INTO bookmarks (id, name, icon, url_internal, url_public_template, url_fallback, group_id, description, open_in_new_tab, is_private, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, b.ID, b.Name, b.Icon, b.URLInternal, b.URLPublicTemplate, b.URLFallback, b.GroupID, b.Description, b.OpenInNewTab, b.IsPrivate, b.SortOrder)
			if err == nil {
				count++
			}
		}
		return count, nil
	}

	return 0, fmt.Errorf("未能识别的 JSON 结构")
}

func getTagsForBookmarks(bookmarkIDs []string) map[string][]models.Tag {
	res := make(map[string][]models.Tag)
	if len(bookmarkIDs) == 0 {
		return res
	}

	placeholders := make([]string, len(bookmarkIDs))
	args := make([]interface{}, len(bookmarkIDs))
	for i, id := range bookmarkIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT bt.bookmark_id, t.id, t.name, t.color
		FROM bookmark_tags bt
		JOIN tags t ON bt.tag_id = t.id
		WHERE bt.bookmark_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return res
	}
	defer rows.Close()

	for rows.Next() {
		var bookmarkID string
		var t models.Tag
		if err := rows.Scan(&bookmarkID, &t.ID, &t.Name, &t.Color); err == nil {
			res[bookmarkID] = append(res[bookmarkID], t)
		}
	}

	return res
}

func (h *BookmarkHandler) FetchFavicon(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url 参数不能为空"})
		return
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 URL 格式"})
		return
	}

	host := parsed.Host
	hostname := parsed.Hostname()
	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "http"
	}

	// Smart keyword mapping to popular Tabler icons
	lowerURL := strings.ToLower(rawURL)
	suggestedIcon := "tabler:world"
	if strings.Contains(lowerURL, "github") {
		suggestedIcon = "tabler:brand-github"
	} else if strings.Contains(lowerURL, "emby") || strings.Contains(lowerURL, "jellyfin") || strings.Contains(lowerURL, "plex") {
		suggestedIcon = "tabler:movie"
	} else if strings.Contains(lowerURL, "qbittorrent") || strings.Contains(lowerURL, "aria2") || strings.Contains(lowerURL, "transmission") {
		suggestedIcon = "tabler:download"
	} else if strings.Contains(lowerURL, "router") || strings.Contains(lowerURL, "openwrt") || strings.Contains(lowerURL, "ikuai") {
		suggestedIcon = "tabler:router"
	} else if strings.Contains(lowerURL, "pve") || strings.Contains(lowerURL, "proxmox") || strings.Contains(lowerURL, "esxi") {
		suggestedIcon = "tabler:server"
	} else if strings.Contains(lowerURL, "docker") || strings.Contains(lowerURL, "portainer") {
		suggestedIcon = "tabler:brand-docker"
	} else if strings.Contains(lowerURL, "homeassistant") || strings.Contains(lowerURL, "hass") {
		suggestedIcon = "tabler:home-2"
	} else if strings.Contains(lowerURL, "vaultwarden") || strings.Contains(lowerURL, "bitwarden") {
		suggestedIcon = "tabler:shield-lock"
	} else if strings.Contains(lowerURL, "nas") || strings.Contains(lowerURL, "synology") || strings.Contains(lowerURL, "nextcloud") {
		suggestedIcon = "tabler:cloud"
	} else if strings.Contains(lowerURL, "bilibili") {
		suggestedIcon = "tabler:brand-bilibili"
	} else if strings.Contains(lowerURL, "youtube") {
		suggestedIcon = "tabler:brand-youtube"
	}

	iconURL := ""
	title := ""

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, reqErr := http.NewRequest("GET", rawURL, nil)
	if reqErr == nil {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
			bodyStr := string(bodyBytes)

			// Extract title
			reTitle := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
			if m := reTitle.FindStringSubmatch(bodyStr); len(m) > 1 {
				title = strings.TrimSpace(html.UnescapeString(m[1]))
			}

			// Extract favicon link
			reIcon := regexp.MustCompile(`(?i)<link[^>]+rel=["'](?:shortcut )?icon["'][^>]+href=["']([^"']+)["']`)
			if m := reIcon.FindStringSubmatch(bodyStr); len(m) > 1 {
				foundHref := strings.TrimSpace(m[1])
				if strings.HasPrefix(foundHref, "http://") || strings.HasPrefix(foundHref, "https://") {
					iconURL = foundHref
				} else if strings.HasPrefix(foundHref, "//") {
					iconURL = scheme + ":" + foundHref
				} else if strings.HasPrefix(foundHref, "/") {
					iconURL = fmt.Sprintf("%s://%s%s", scheme, host, foundHref)
				} else {
					iconURL = fmt.Sprintf("%s://%s/%s", scheme, host, foundHref)
				}
			}
		}
	}

	// Fallback: Direct host /favicon.ico
	if iconURL == "" {
		iconURL = fmt.Sprintf("%s://%s/favicon.ico", scheme, host)
	}

	// High-res public domain favicon fallback
	googleFavicon := ""
	if strings.Contains(hostname, ".") && !strings.HasPrefix(hostname, "192.168.") && !strings.HasPrefix(hostname, "10.") && !strings.HasPrefix(hostname, "172.") && hostname != "localhost" {
		googleFavicon = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", hostname)
	}

	c.JSON(http.StatusOK, gin.H{
		"title":          title,
		"icon_url":       iconURL,
		"google_favicon": googleFavicon,
		"suggested_icon": suggestedIcon,
	})
}

