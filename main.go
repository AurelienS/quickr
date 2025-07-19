package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"quickr/handlers"
	"quickr/models"
)

func main() {
	// Initialize SQLite database
	db, err := gorm.Open(sqlite.Open("quickr.db"), &gorm.Config{})
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

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")
	log.Println("Templates loaded")

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