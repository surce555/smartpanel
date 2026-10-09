package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"smartpanel/database"
	"smartpanel/middleware"
	"smartpanel/models"
)

type AuthHandler struct{}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数格式错误", "code": http.StatusBadRequest})
		return
	}

	accountName := strings.TrimSpace(req.Email)

	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, email, password_hash, created_at FROM users WHERE email = ? OR (email = 'admin@smartpanel.local' AND ? = 'admin')",
		accountName, accountName,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "账号名称或密码错误", "code": http.StatusUnauthorized})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库查询失败", "code": http.StatusInternalServerError})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号名称或密码错误", "code": http.StatusUnauthorized})
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败", "code": http.StatusInternalServerError})
		return
	}

	mustChange := database.GetSetting("must_change_password") == "true"

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":                   user.ID,
			"email":                user.Email,
			"must_change_password": mustChange,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": http.StatusUnauthorized})
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, email, created_at FROM users WHERE id = ?",
		userID,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在", "code": http.StatusNotFound})
		return
	}

	mustChange := database.GetSetting("must_change_password") == "true"

	c.JSON(http.StatusOK, gin.H{
		"id":                   user.ID,
		"email":                user.Email,
		"must_change_password": mustChange,
	})
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": http.StatusUnauthorized})
		return
	}

	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号名称不能为空", "code": http.StatusBadRequest})
		return
	}

	newAccount := strings.TrimSpace(req.Email)
	if newAccount == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号名称不能为空", "code": http.StatusBadRequest})
		return
	}

	_, err := database.DB.Exec("UPDATE users SET email = ? WHERE id = ?", newAccount, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "账号名称已被使用或更新失败", "code": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "账号名称修改成功",
		"email":   newAccount,
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录", "code": http.StatusUnauthorized})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "code": http.StatusBadRequest})
		return
	}

	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码长度不能少于 6 位", "code": http.StatusBadRequest})
		return
	}

	var currentHash string
	err := database.DB.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&currentHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户不存在", "code": http.StatusInternalServerError})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原密码不正确", "code": http.StatusBadRequest})
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加密新密码失败", "code": http.StatusInternalServerError})
		return
	}

	_, err = database.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(newHash), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败", "code": http.StatusInternalServerError})
		return
	}

	_ = database.SetSetting("must_change_password", "false")

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}
