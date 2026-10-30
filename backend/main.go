package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"smartpanel/config"
	"smartpanel/database"
	"smartpanel/handlers"
	"smartpanel/middleware"
	"smartpanel/services"
)

func main() {
	cfg := config.InitConfig()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	if err := database.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	services.InitDDNSService()

	// Background bookmark health sweep so card indicators reflect reality.
	handlers.StartHealthScheduler(2 * time.Minute)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)
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
	memoH := &handlers.MemoHandler{}

	api := r.Group("/api")
	{
		// Health probe ping
		api.GET("/health/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "panel": "smartpanel"})
		})

		// 1x1 transparent PNG probe endpoint for client network connectivity detection
		api.GET("/health/probe.png", func(c *gin.Context) {
			pngBytes := []byte{
				0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
				0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
				0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
				0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
				0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
				0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
			}
			c.Header("Content-Type", "image/png")
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Private-Network", "true")
			c.Data(http.StatusOK, "image/png", pngBytes)
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
		api.GET("/health/check/:bookmarkId", middleware.OptionalAuthMiddleware(), healthH.CheckBookmarkHealth)
		// OptionalAuth so the handler can decide whether guest access is allowed;
		// it is no longer an unauthenticated "fetch any URL for me" endpoint.
		api.GET("/bookmarks/fetch-favicon", middleware.OptionalAuthMiddleware(), bookmarkH.FetchFavicon)
		api.GET("/memo", memoH.GetMemo)

		// Protected Admin Routes
		admin := api.Group("")
		admin.Use(middleware.AuthMiddleware())
		{
			// Health checks that fan out across every bookmark stay admin-only.
			admin.GET("/health/all", healthH.CheckAllHealth)

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

			// System & Docker monitoring & container control
			admin.GET("/system/status", systemH.GetSystemStatus)
			admin.GET("/system/docker", dockerH.GetDockerContainers)
			admin.POST("/system/docker/:id/start", dockerH.StartContainer)
			admin.POST("/system/docker/:id/stop", dockerH.StopContainer)
			admin.POST("/system/docker/:id/restart", dockerH.RestartContainer)
			admin.GET("/system/docker/:id/logs", dockerH.GetContainerLogs)
			admin.POST("/system/network/check", networkH.CheckNetwork)

			// Backup & Restore
			admin.POST("/backup/export", backupH.ExportBackup)
			admin.POST("/backup/import", backupH.ImportBackup)

			// Quick memo write is owner-only (guests keep read access)
			admin.PUT("/memo", memoH.UpdateMemo)
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
		log.Printf("================================================================================")
		log.Printf("[ERROR] SmartPanel 启动失败: 端口 %s 无法绑定 (%v)", cfg.Port, err)
		log.Printf("[排查指南]")
		log.Printf(" 1. 如果使用了 network_mode: host，说明 NAS 宿主机该端口已有其他服务运行（或旧容器未停止）。")
		log.Printf(" 2. 如果 docker-compose.yml 中同时开启了 network_mode: host 和 ports 映射，会导致 Docker 端口冲突。")
		log.Printf(" 3. 您可以通过环境变量 PANEL_PORT=5050 (或其他可用端口) 更换服务端口。")
		log.Printf("================================================================================")
		time.Sleep(10 * time.Second)
		log.Fatalf("Server failed to start: %v", err)
	}
}
