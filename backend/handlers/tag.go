package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartpanel/database"
	"smartpanel/models"
)

type TagHandler struct{}

func (h *TagHandler) GetTags(c *gin.Context) {
	rows, err := database.DB.Query("SELECT id, name, color FROM tags ORDER BY name ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签失败", "code": http.StatusInternalServerError})
		return
	}
	defer rows.Close()

	tags := make([]models.Tag, 0)
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color); err == nil {
			tags = append(tags, t)
		}
	}

	c.JSON(http.StatusOK, tags)
}

func (h *TagHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签名称不能为空", "code": http.StatusBadRequest})
		return
	}

	color := req.Color
	if color == "" {
		color = "#6366f1"
	}

	tag := models.Tag{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Color: color,
	}

	_, err := database.DB.Exec("INSERT INTO tags (id, name, color) VALUES (?, ?, ?)", tag.ID, tag.Name, tag.Color)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "创建标签失败或标签已存在", "code": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, tag)
}

func (h *TagHandler) DeleteTag(c *gin.Context) {
	id := c.Param("id")
	res, err := database.DB.Exec("DELETE FROM tags WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除标签失败", "code": http.StatusInternalServerError})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在", "code": http.StatusNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
