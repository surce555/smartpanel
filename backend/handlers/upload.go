package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"smartpanel/config"
)

type UploadHandler struct{}

func (h *UploadHandler) UploadIcon(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的图标文件", "code": http.StatusBadRequest})
		return
	}

	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图标文件大小不能超过 2MB", "code": http.StatusBadRequest})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".png": true, ".svg": true, ".webp": true, ".jpg": true, ".jpeg": true, ".ico": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 PNG, SVG, WebP, JPG, ICO 格式图标", "code": http.StatusBadRequest})
		return
	}

	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dst := filepath.Join(config.AppConfig.UploadsDir, "icons", fileName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      fmt.Sprintf("/uploads/icons/%s", fileName),
		"filename": fileName,
	})
}

func (h *UploadHandler) UploadWallpaper(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的壁纸文件", "code": http.StatusBadRequest})
		return
	}

	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "壁纸文件大小不能超过 10MB", "code": http.StatusBadRequest})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".png": true, ".webp": true, ".jpg": true, ".jpeg": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG, PNG, WebP 格式壁纸", "code": http.StatusBadRequest})
		return
	}

	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dst := filepath.Join(config.AppConfig.UploadsDir, "wallpapers", fileName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败", "code": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":      fmt.Sprintf("/uploads/wallpapers/%s", fileName),
		"filename": fileName,
	})
}

func (h *UploadHandler) UploadFont(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择字体文件", "code": http.StatusBadRequest})
		return
	}

	if file.Size > 15*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "字体文件大小不能超过 15MB", "code": http.StatusBadRequest})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{".ttf": true, ".woff2": true, ".woff": true, ".otf": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 TTF, WOFF2, WOFF, OTF 格式字体", "code": http.StatusBadRequest})
		return
	}

	fileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	dst := filepath.Join(config.AppConfig.UploadsDir, "fonts", fileName)

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存字体失败", "code": http.StatusInternalServerError})
		return
	}

	fontFamily := strings.TrimSuffix(file.Filename, ext)
	c.JSON(http.StatusOK, gin.H{
		"url":         fmt.Sprintf("/uploads/fonts/%s", fileName),
		"filename":    fileName,
		"font_family": fontFamily,
	})
}

func (h *UploadHandler) ListUploadedFiles(c *gin.Context) {
	category := c.DefaultQuery("category", "wallpapers")
	targetDir := filepath.Join(config.AppConfig.UploadsDir, category)

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		c.JSON(http.StatusOK, []string{})
		return
	}

	files := make([]gin.H, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			files = append(files, gin.H{
				"name":    entry.Name(),
				"url":     fmt.Sprintf("/uploads/%s/%s", category, entry.Name()),
				"size":    info.Size(),
				"modtime": info.ModTime(),
			})
		}
	}

	c.JSON(http.StatusOK, files)
}

func (h *UploadHandler) DeleteUploadedFile(c *gin.Context) {
	filename := c.Param("filename")
	category := c.DefaultQuery("category", "wallpapers")

	// Prevent directory traversal
	cleaned := filepath.Base(filename)
	filePath := filepath.Join(config.AppConfig.UploadsDir, category, cleaned)

	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在或删除失败", "code": http.StatusNotFound})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "文件已删除"})
}
