package handlers

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartpanel/database"
	"smartpanel/models"
)

type HealthHandler struct{}

func (h *HealthHandler) CheckBookmarkHealth(c *gin.Context) {
	bookmarkID := c.Param("bookmarkId")

	var urlInternal, urlPublic, urlFallback string
	err := database.DB.QueryRow(`
		SELECT url_internal, url_public_template, url_fallback FROM bookmarks WHERE id = ?
	`, bookmarkID).Scan(&urlInternal, &urlPublic, &urlFallback)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "书签不存在", "code": http.StatusNotFound})
		return
	}

	targetURL := urlInternal
	if targetURL == "" {
		targetURL = urlFallback
	}
	if targetURL == "" {
		targetURL = urlPublic
	}

	res := probeURL(bookmarkID, targetURL)
	saveHealthCheck(res)

	c.JSON(http.StatusOK, res)
}

func (h *HealthHandler) CheckAllHealth(c *gin.Context) {
	rows, err := database.DB.Query("SELECT id, url_internal, url_fallback, url_public_template FROM bookmarks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询书签失败", "code": http.StatusInternalServerError})
		return
	}
	defer rows.Close()

	type task struct {
		id  string
		url string
	}
	tasks := make([]task, 0)

	for rows.Next() {
		var id, uInt, uFall, uPub sqlNullString
		if err := rows.Scan(&id, &uInt, &uFall, &uPub); err == nil {
			targetURL := uInt.String
			if targetURL == "" {
				targetURL = uFall.String
			}
			if targetURL == "" {
				targetURL = uPub.String
			}
			if targetURL != "" {
				tasks = append(tasks, task{id: id.String, url: targetURL})
			}
		}
	}

	results := make([]models.HealthCheck, len(tasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // Limit to 8 concurrent probes

	for i, t := range tasks {
		wg.Add(1)
		go func(idx int, tk task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := probeURL(tk.id, tk.url)
			saveHealthCheck(res)
			results[idx] = res
		}(i, t)
	}

	wg.Wait()
	c.JSON(http.StatusOK, results)
}

func probeURL(bookmarkID, targetURL string) models.HealthCheck {
	now := time.Now()
	res := models.HealthCheck{
		BookmarkID:  bookmarkID,
		Status:      "unknown",
		LastChecked: &now,
	}

	if targetURL == "" {
		return res
	}

	// InsecureSkipVerify for self-signed certificates on internal NAS networks
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   3 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(targetURL)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		res.Status = "offline"
		res.ResponseTimeMS = int(duration)
		return res
	}
	defer resp.Body.Close()

	res.Status = "online"
	res.ResponseTimeMS = int(duration)
	return res
}

func saveHealthCheck(hc models.HealthCheck) {
	checkID := uuid.New().String()
	_, _ = database.DB.Exec(`
		INSERT INTO health_checks (id, bookmark_id, status, last_checked, response_time_ms)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(bookmark_id) DO UPDATE SET 
			status = excluded.status, 
			last_checked = excluded.last_checked, 
			response_time_ms = excluded.response_time_ms
	`, checkID, hc.BookmarkID, hc.Status, hc.LastChecked, hc.ResponseTimeMS)
}
