package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/database"
	"smartpanel/handlers"
	"smartpanel/middleware"
	"smartpanel/services"
)

func main() {
	cfg := config.InitConfig()

	if err := database.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	services.InitDDNSService()

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Static files for uploads (icons, wallpapers, fonts)
	r.Static("/uploads", cfg.UploadsDir)

	// Handlers
	authH := &handlers.AuthHandler{}
	groupH := &handlers.GroupHandler{}
	bookmarkH := &handlers.BookmarkHandler{}
	tagH := &handlers.TagHandler{}
	settingsH := &handlers.SettingsHandler{}
	uploadH := &handlers.UploadHandler{}
	systemH := &handlers.SystemHandler{}
	dockerH := &handlers.DockerHandler{}
	healthH := &handlers.HealthHandler{}
	networkH := &handlers.NetworkHandler{}
	backupH := &handlers.BackupHandler{}

	api := r.Group("/api")
	{
		// Health probe ping
		api.GET("/health/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "panel": "smartpanel"})
		})

		// Public/Optional Auth routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/login", authH.Login)
			authGroup.POST("/logout", authH.Logout)
			authGroup.GET("/me", middleware.AuthMiddleware(), authH.GetMe)
			authGroup.PUT("/password", middleware.AuthMiddleware(), authH.ChangePassword)
			authGroup.PUT("/profile", middleware.AuthMiddleware(), authH.UpdateProfile)
		}

		// Bookmarks & Groups (readable by guests, filtered by is_private)
		api.GET("/groups", middleware.OptionalAuthMiddleware(), groupH.GetGroups)
		api.GET("/bookmarks", middleware.OptionalAuthMiddleware(), bookmarkH.GetBookmarks)
		api.GET("/tags", tagH.GetTags)
		api.GET("/settings", middleware.OptionalAuthMiddleware(), settingsH.GetSettings)
		api.GET("/settings/theme", settingsH.GetThemeSettings)
		api.GET("/system/network", networkH.GetNetworkInfo)
		api.GET("/system/network/stream", networkH.StreamNetworkUpdates)
		api.GET("/health/all", healthH.CheckAllHealth)
		api.GET("/health/check/:bookmarkId", healthH.CheckBookmarkHealth)

		// Protected Admin Routes
		admin := api.Group("")
		admin.Use(middleware.AuthMiddleware())
		{
			// Groups management
			admin.POST("/groups", groupH.CreateGroup)
			admin.PUT("/groups/:id", groupH.UpdateGroup)
			admin.DELETE("/groups/:id", groupH.DeleteGroup)
			admin.PUT("/groups/reorder", groupH.ReorderGroups)

			// Bookmarks management
			admin.POST("/bookmarks", bookmarkH.CreateBookmark)
			admin.PUT("/bookmarks/:id", bookmarkH.UpdateBookmark)
			admin.DELETE("/bookmarks/:id", bookmarkH.DeleteBookmark)
			admin.PUT("/bookmarks/reorder", bookmarkH.ReorderBookmarks)
			admin.POST("/bookmarks/import", bookmarkH.ImportBookmarks)
			admin.GET("/bookmarks/export", bookmarkH.ExportBookmarks)

			// Tags management
			admin.POST("/tags", tagH.CreateTag)
			admin.DELETE("/tags/:id", tagH.DeleteTag)

			// Settings
			admin.PUT("/settings", settingsH.UpdateSettings)
			admin.PUT("/settings/theme", settingsH.UpdateThemeSettings)

			// Uploads
			admin.POST("/upload/icon", uploadH.UploadIcon)
			admin.POST("/upload/wallpaper", uploadH.UploadWallpaper)
			admin.POST("/upload/font", uploadH.UploadFont)
			admin.GET("/upload/list", uploadH.ListUploadedFiles)
			admin.DELETE("/upload/:filename", uploadH.DeleteUploadedFile)

			// System & Docker monitoring
			admin.GET("/system/status", systemH.GetSystemStatus)
			admin.GET("/system/docker", dockerH.GetDockerContainers)
			admin.POST("/system/network/check", networkH.CheckNetwork)

			// Backup & Restore
			admin.POST("/backup/export", backupH.ExportBackup)
			admin.POST("/backup/import", backupH.ImportBackup)
		}
	}

	// Frontend SPA serving: check possible dist directories
	distCandidates := []string{"./dist", "../frontend/dist", "./frontend/dist", "/app/dist"}
	var frontendDist string
	for _, d := range distCandidates {
		if stat, err := os.Stat(d); err == nil && stat.IsDir() {
			frontendDist = d
			break
		}
	}

	if frontendDist != "" {
		log.Printf("Serving frontend from: %s", frontendDist)
		r.Static("/assets", filepath.Join(frontendDist, "assets"))
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// If not API request, serve index.html
			if len(path) >= 4 && path[:4] == "/api" {
				c.JSON(http.StatusNotFound, gin.H{"error": "接口不存在", "code": 404})
				return
			}
			c.File(filepath.Join(frontendDist, "index.html"))
		})
	}

	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	log.Printf("SmartPanel server listening on http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
