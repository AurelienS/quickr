package main

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"quickr/handlers"
	"quickr/models"
)

// Ensure database directory exists
func ensureDBDir() error {
	dbDir := "data"
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}
	return nil
}

func main() {
	// Create data directory if it doesn't exist
	if err := ensureDBDir(); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Initialize SQLite database
	dbPath := filepath.Join("data", "quickr.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.Link{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Setup Gin router
	r := gin.New() // Don't use Default() as it already includes Logger
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Add our custom request logger
	r.Use(func(c *gin.Context) {
		log.Printf("[REQUEST] %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("[RESPONSE] %s %s -> %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	})

	// Load HTML templates from embedded FS
	templ := template.Must(template.New("").ParseFS(templateFS, "templates/*.html"))
	r.SetHTMLTemplate(templ)
	log.Println("Templates loaded from embedded FS")

	// Serve static files from embedded FS
	staticSubFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem for static files:", err)
	}
	r.StaticFS("/static", http.FS(staticSubFS))
	log.Println("Static files route added from embedded FS")

	// Web routes
	r.GET("/", handlers.HandleHome(db))
	r.GET("/stats", handlers.HandleStats(db))
	r.GET("/hot", handlers.HandleHot(db))

	// Redirect route with debug handler
	r.GET("/go/:alias", func(c *gin.Context) {
		log.Printf("[DEBUG] About to call redirect handler for alias: %s", c.Param("alias"))
		handlers.HandleRedirect(db)(c)
	})

	// API routes
	api := r.Group("/api")
	{
		api.GET("/links", handlers.ListLinks(db))
		api.POST("/links", handlers.CreateLink(db))
		api.GET("/links/:id/edit", handlers.GetLinkEditField(db))
		api.PUT("/links/:id", handlers.UpdateLink(db))
		api.DELETE("/links/:id", handlers.DeleteLink(db))
		api.GET("/search", handlers.SearchLinks(db))
	}

	log.Printf("Server starting on http://localhost:8080")
	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}