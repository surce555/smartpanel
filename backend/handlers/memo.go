package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"smartpanel/database"
)

type MemoHandler struct{}

func (h *MemoHandler) GetMemo(c *gin.Context) {
	content := database.GetSetting("quick_memo_content")
	updatedAt := database.GetSetting("quick_memo_updated_at")
	c.JSON(http.StatusOK, gin.H{
		"content":    content,
		"updated_at": updatedAt,
	})
}

func (h *MemoHandler) UpdateMemo(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	_ = database.SetSetting("quick_memo_content", req.Content)
	_ = database.SetSetting("quick_memo_updated_at", time.Now().Format("2006-01-02 15:04:05"))

	c.JSON(http.StatusOK, gin.H{
		"message":    "便签已保存",
		"updated_at": time.Now().Format("2006-01-02 15:04:05"),
	})
}
