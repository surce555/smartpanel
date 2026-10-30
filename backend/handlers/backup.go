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

	// 1. Add a consistent snapshot of the database to the zip.
	//    Copying smartpanel.db directly is unsafe in WAL mode: recently committed
	//    transactions may still live in smartpanel.db-wal, so the raw file can be
	//    missing data. VACUUM INTO produces a complete, standalone copy.
	if err := writeDBSnapshotToZip(zipWriter); err != nil {
		// Fall back to copying the raw file so the export still contains something.
		if dbFile, openErr := os.Open(config.AppConfig.DBPath); openErr == nil {
			if w, createErr := zipWriter.Create("smartpanel.db"); createErr == nil {
				_, _ = io.Copy(w, dbFile)
			}
			_ = dbFile.Close()
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

		_, _ = io.Copy(w, f)
		// Close per file: `defer` inside filepath.Walk would pile up every file
		// handle until the whole archive finished, exhausting the FD limit on
		// NAS boxes with large wallpaper/font folders.
		_ = f.Close()
		return nil
	})
}

// writeDBSnapshotToZip exports a consistent copy of the SQLite database using
// VACUUM INTO, which flushes WAL contents into a single self-contained file.
func writeDBSnapshotToZip(zipWriter *zip.Writer) error {
	snapshotPath := filepath.Join(config.AppConfig.DataDir, "backup-snapshot.db")
	_ = os.Remove(snapshotPath)
	defer os.Remove(snapshotPath)

	if _, err := database.DB.Exec("VACUUM INTO ?", snapshotPath); err != nil {
		return err
	}

	snapshot, err := os.Open(snapshotPath)
	if err != nil {
		return err
	}
	defer snapshot.Close()

	w, err := zipWriter.Create("smartpanel.db")
	if err != nil {
		return err
	}
	_, err = io.Copy(w, snapshot)
	return err
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

	// Extract uploads first; the database is restored separately at the end so
	// the live connection is closed and swapped atomically.
	var dbEntry *zip.File
	for _, f := range r.File {
		cleaned := filepath.Clean(f.Name)
		if strings.HasPrefix(cleaned, "..") {
			continue // Prevent Zip Slip vulnerability
		}

		if cleaned == "smartpanel.db" {
			dbEntry = f
			continue
		}

		if strings.HasPrefix(cleaned, "uploads/") || strings.HasPrefix(cleaned, "uploads\\") {
			rel := strings.TrimPrefix(cleaned, "uploads/")
			rel = strings.TrimPrefix(rel, "uploads\\")
			dst := filepath.Join(config.AppConfig.UploadsDir, rel)
			// Re-check containment after joining: `uploads/../../x` would
			// otherwise still escape the uploads directory.
			if !isPathInside(config.AppConfig.UploadsDir, dst) {
				continue
			}
			_ = os.MkdirAll(filepath.Dir(dst), 0755)
			_ = extractZipFile(f, dst)
		}
	}

	// Restore the database through a staged file + atomic rename, and only after
	// closing the live connection. Writing straight over smartpanel.db while
	// SQLite (WAL mode) still had it open corrupted the restore.
	if dbEntry != nil {
		if err := restoreDatabase(dbEntry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "还原数据库失败: " + err.Error(), "code": http.StatusInternalServerError})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "备份还原成功，数据已生效"})
}

// isPathInside reports whether target stays within baseDir after cleaning.
func isPathInside(baseDir, target string) bool {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

// restoreDatabase swaps in the backup DB file safely:
//  1. extract to a staging file next to the real DB
//  2. close the live connection (and its WAL sidecars)
//  3. atomically rename the staging file into place
//  4. reopen
func restoreDatabase(entry *zip.File) error {
	dbPath := config.AppConfig.DBPath
	stagingPath := dbPath + ".restore"

	if err := extractZipFile(entry, stagingPath); err != nil {
		return err
	}
	defer os.Remove(stagingPath)

	// Close the live handle first — SQLite must not hold the file while we
	// replace it, otherwise the restored bytes are overwritten by the old WAL.
	if database.DB != nil {
		_ = database.DB.Close()
		database.DB = nil
	}
	for _, sidecar := range []string{dbPath + "-wal", dbPath + "-shm"} {
		_ = os.Remove(sidecar)
	}

	if err := os.Rename(stagingPath, dbPath); err != nil {
		// Rename across the same directory should succeed; fall back to copy.
		if err := copyFile(stagingPath, dbPath); err != nil {
			_ = database.InitDB()
			return err
		}
	}

	return database.InitDB()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
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
