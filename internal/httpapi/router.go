package httpapi

import (
	"context"
	"database/sql"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Icingworld/doubao-space/internal/entertainment"
	"github.com/Icingworld/doubao-space/internal/recipes"
	"github.com/Icingworld/doubao-space/internal/schedules"
	"github.com/Icingworld/doubao-space/internal/tasks"
	"github.com/Icingworld/doubao-space/web"
	"github.com/gin-gonic/gin"
)

// NewRouter creates the API and embedded frontend routes.
func NewRouter(db *sql.DB, version string, dev bool) *gin.Engine {
	return NewRouterWithDataDir(db, version, dev, "")
}

// NewRouterWithDataDir also enables local task attachment storage.
func NewRouterWithDataDir(db *sql.DB, version string, dev bool, dataDir string) *gin.Engine {
	if !dev {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "doubao-space",
			"version": version,
		})
	})

	router.GET("/api/meta", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":    "豆包空间",
			"version": version,
		})
	})

	taskHandler := tasks.NewHandler(tasks.NewService(db, dataDir))
	taskHandler.Register(router.Group("/api"))
	recipeHandler := recipes.NewHandler(recipes.NewService(db))
	recipeHandler.Register(router.Group("/api"))
	scheduleHandler := schedules.NewHandler(schedules.NewService(db))
	scheduleHandler.Register(router.Group("/api"))
	planHandler := schedules.NewPlanHandler(schedules.NewPersonalService(db))
	planHandler.Register(router.Group("/api"))
	entertainmentHandler := entertainment.NewHandler(entertainment.NewService(db))
	entertainmentHandler.Register(router.Group("/api"))

	staticServer := http.FileServer(http.FS(web.DistFS))
	router.NoRoute(func(c *gin.Context) {
		requestPath := path.Clean(strings.TrimPrefix(c.Request.URL.Path, "/"))
		if strings.HasPrefix(requestPath, "api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		if requestPath == "." {
			c.Request.URL.Path = "/"
			staticServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		if info, err := fs.Stat(web.DistFS, requestPath); err == nil && !info.IsDir() {
			staticServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// Vue Router history mode needs unknown extensionless paths to return index.html.
		if filepath.Ext(requestPath) == "" {
			c.Request.URL.Path = "/"
			staticServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Status(http.StatusNotFound)
	})

	return router
}
