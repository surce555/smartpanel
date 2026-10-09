package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartpanel/database"
	"smartpanel/models"
)

type GroupHandler struct{}

func (h *GroupHandler) GetGroups(c *gin.Context) {
	isAuth, _ := c.Get("is_authenticated")
	isAuthenticated, _ := isAuth.(bool)

	if database.GetSetting("require_login") == "true" && !isAuthenticated {
		c.JSON(http.StatusOK, []models.Group{})
		return
	}

	rows, err := database.DB.Query("SELECT id, name, icon, sort_order, created_at FROM groups ORDER BY sort_order ASC, created_at ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分组失败", "code": http.StatusInternalServerError})
		return
	}
	defer rows.Close()

	groups := make([]models.Group, 0)
	for rows.Next() {
		var g models.Group
		var icon sqlNullString
		if err := rows.Scan(&g.ID, &g.Name, &icon, &g.SortOrder, &g.CreatedAt); err == nil {
			g.Icon = icon.String
			groups = append(groups, g)
		}
	}

	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Icon string `json:"icon"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分组名称不能为空", "code": http.StatusBadRequest})
		return
	}

	var maxSort int
	_ = database.DB.QueryRow("SELECT COALESCE(MAX(sort_order), -1) FROM groups").Scan(&maxSort)

	group := models.Group{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Icon:      req.Icon,
		SortOrder: maxSort + 1,
		CreatedAt: time.Now(),
	}

	_, err := database.DB.Exec(
		"INSERT INTO groups (id, name, icon, sort_order, created_at) VALUES (?, ?, ?, ?, ?)",
		group.ID, group.Name, group.Icon, group.SortOrder, group.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分组失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name      string `json:"name" binding:"required"`
		Icon      string `json:"icon"`
		SortOrder *int   `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "code": http.StatusBadRequest})
		return
	}

	query := "UPDATE groups SET name = ?, icon = ?"
	args := []interface{}{req.Name, req.Icon}

	if req.SortOrder != nil {
		query += ", sort_order = ?"
		args = append(args, *req.SortOrder)
	}

	query += " WHERE id = ?"
	args = append(args, id)

	res, err := database.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分组失败", "code": http.StatusInternalServerError})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "分组不存在", "code": http.StatusNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id := c.Param("id")

	// Set group_id to null for bookmarks in this group, or let foreign key handle it
	_, _ = database.DB.Exec("UPDATE bookmarks SET group_id = NULL WHERE group_id = ?", id)

	res, err := database.DB.Exec("DELETE FROM groups WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分组失败", "code": http.StatusInternalServerError})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "分组不存在", "code": http.StatusNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (h *GroupHandler) ReorderGroups(c *gin.Context) {
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

	stmt, err := tx.Prepare("UPDATE groups SET sort_order = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "准备语句失败", "code": http.StatusInternalServerError})
		return
	}
	defer stmt.Close()

	for _, item := range items {
		_, err = stmt.Exec(item.SortOrder, item.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新排序失败", "code": http.StatusInternalServerError})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序更新成功"})
}

// sqlNullString helper for nullable database strings
type sqlNullString struct {
	String string
	Valid  bool
}

func (s *sqlNullString) Scan(value interface{}) error {
	if value == nil {
		s.String, s.Valid = "", false
		return nil
	}
	switch v := value.(type) {
	case string:
		s.String, s.Valid = v, true
	case []byte:
		s.String, s.Valid = string(v), true
	}
	return nil
}
