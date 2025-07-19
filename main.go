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
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("templates/*.html")
	log.Println("Templates loaded from templates/*.html")

	// Web routes
	r.GET("/", handlers.HandleHome(db))
	r.GET("/stats", handlers.HandleStats(db))

	// Redirect route
	r.GET("/go/:alias", handlers.HandleRedirect(db))

	// API routes
	api := r.Group("/api")
	{
		api.GET("/links", handlers.ListLinks(db))
		api.POST("/links", handlers.CreateLink(db))
		api.PUT("/links/:id", handlers.UpdateLink(db))
		api.DELETE("/links/:id", handlers.DeleteLink(db))
	}

	log.Printf("Server starting on http://localhost:8080")
	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}