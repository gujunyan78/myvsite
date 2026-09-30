package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"myvsite/internal/config"
	"myvsite/internal/handler"
	"myvsite/internal/middleware"
)

// New 注册所有路由
func New(c *config.Config) *gin.Engine {
	r := gin.Default()

	r.Use(func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, Authorization")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if ctx.Request.Method == http.MethodOptions {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})

	// 静态资源
	staticDir := filepath.Join(c.WebDir, "static")
	r.Static("/static", staticDir)

	// 页面
	r.GET("/", func(ctx *gin.Context) {
		ctx.File(filepath.Join(c.WebDir, "index.html"))
	})
	r.GET("/admin", func(ctx *gin.Context) {
		ctx.File(filepath.Join(c.WebDir, "admin.html"))
	})
	r.GET("/admin.html", func(ctx *gin.Context) {
		ctx.File(filepath.Join(c.WebDir, "admin.html"))
	})

	api := r.Group("/api")
	{
		api.GET("/videos", handler.ListVideos)
		api.GET("/videos/:id", handler.GetVideo)
		api.POST("/videos/:id/view", handler.IncrViews)
		api.POST("/videos/:id/like", handler.ToggleLike)
		api.GET("/videos/:id/comments", handler.ListComments)
		api.POST("/videos/:id/comments", handler.CreateComment)
		api.GET("/bloggers", handler.ListBloggers)
		api.GET("/bloggers/:id", handler.GetBlogger)

		api.POST("/admin/login", handler.Login)
		api.POST("/admin/logout", handler.Logout)

		adm := api.Group("/admin")
		adm.Use(middleware.AdminAuth())
		{
			adm.GET("/profile", handler.Profile)
			adm.POST("/password", handler.ChangePassword)
			adm.POST("/change-password", handler.ChangePassword)
			adm.GET("/stats", handler.Stats)
			adm.GET("/bloggers", handler.AdminListBloggers)
			adm.POST("/bloggers", handler.AdminCreateBlogger)
			adm.PUT("/bloggers/:id", handler.AdminUpdateBlogger)
			adm.DELETE("/bloggers/:id", handler.AdminDeleteBlogger)
			adm.GET("/videos", handler.AdminListVideos)
			adm.POST("/videos", handler.AdminCreateVideo)
			adm.PUT("/videos/:id", handler.AdminUpdateVideo)
			adm.DELETE("/videos/:id", handler.AdminDeleteVideo)
			adm.DELETE("/comments/:id", handler.AdminDeleteComment)
			adm.POST("/upload", handler.Upload)
		}
	}

	r.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api/") {
			ctx.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "API not found"})
			return
		}
		target := filepath.Join(c.WebDir, "index.html")
		if _, err := os.Stat(target); err == nil {
			ctx.File(target)
			return
		}
		ctx.String(http.StatusNotFound, "404 page not found")
	})
	return r
}
