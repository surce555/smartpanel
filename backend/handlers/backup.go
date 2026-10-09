package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/database"
)

type BackupHandler struct{}

func (h *BackupHandler) ExportBackup(c *gin.Context) {
	filename := fmt.Sprintf("smartpanel-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/zip")

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// 1. Add database file to zip
	dbFile, err := os.Open(config.AppConfig.DBPath)
	if err == nil {
		defer dbFile.Close()
		w, err := zipWriter.Create("smartpanel.db")
		if err == nil {
			_, _ = io.Copy(w, dbFile)
		}
	}

	// 2. Add uploads directory to zip
	_ = filepath.Walk(config.AppConfig.UploadsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(config.AppConfig.UploadsDir, path)
		if err != nil {
			return nil
		}

		zipEntryPath := filepath.ToSlash(filepath.Join("uploads", relPath))
		w, err := zipWriter.Create(zipEntryPath)
		if err != nil {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		_, _ = io.Copy(w, f)
		return nil
	})
}

func (h *BackupHandler) ImportBackup(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传有效的备份 ZIP 文件", "code": http.StatusBadRequest})
		return
	}

	tempZipPath := filepath.Join(config.AppConfig.DataDir, "temp-restore.zip")
	if err := c.SaveUploadedFile(file, tempZipPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败", "code": http.StatusInternalServerError})
		return
	}
	defer os.Remove(tempZipPath)

	r, err := zip.OpenReader(tempZipPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法解压缩备份文件: " + err.Error(), "code": http.StatusBadRequest})
		return
	}
	defer r.Close()

	// Extract files
	for _, f := range r.File {
		cleaned := filepath.Clean(f.Name)
		if strings.HasPrefix(cleaned, "..") {
			continue // Prevent Zip Slip vulnerability
		}

		if cleaned == "smartpanel.db" {
			// Restore DB
			dst := config.AppConfig.DBPath
			if err := extractZipFile(f, dst); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "还原数据库失败", "code": http.StatusInternalServerError})
				return
			}
		} else if strings.HasPrefix(cleaned, "uploads/") || strings.HasPrefix(cleaned, "uploads\\") {
			rel := strings.TrimPrefix(cleaned, "uploads/")
			rel = strings.TrimPrefix(rel, "uploads\\")
			dst := filepath.Join(config.AppConfig.UploadsDir, rel)
			_ = os.MkdirAll(filepath.Dir(dst), 0755)
			_ = extractZipFile(f, dst)
		}
	}

	// Reinitialize database connection to apply restored data
	_ = database.InitDB()

	c.JSON(http.StatusOK, gin.H{"message": "备份还原成功，数据已生效"})
}

func extractZipFile(f *zip.File, dst string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
