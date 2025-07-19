package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quickr/models"
)

// HandleHome renders the homepage with all links
func HandleHome(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var links []models.Link
		result := db.Order("created_at desc").Find(&links)
		if result.Error != nil {
			log.Printf("Database error: %v", result.Error)
			c.String(http.StatusInternalServerError, "Database error")
			return
		}

		log.Printf("Found %d links", len(links))
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":  "Home",
			"active": "home",
			"links":  links,
		})
	}
}

// HandleStats renders the stats page
func HandleStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get total links
		var totalLinks int64
		db.Model(&models.Link{}).Count(&totalLinks)

		// Get total clicks
		var totalClicks int64
		db.Model(&models.Link{}).Select("COALESCE(SUM(clicks), 0)").Row().Scan(&totalClicks)

		// Get unique creators count
		var activeUsers int64
		db.Model(&models.Link{}).Distinct("creator_name").Count(&activeUsers)

		// Get top 5 links by clicks
		var topLinks []models.Link
		db.Order("clicks desc").Limit(5).Find(&topLinks)

		// Get 5 most recent links
		var recentLinks []models.Link
		db.Order("created_at desc").Limit(5).Find(&recentLinks)

		log.Printf("Stats: %d links, %d clicks, %d users", totalLinks, totalClicks, activeUsers)
		c.HTML(http.StatusOK, "stats.html", gin.H{
			"title":       "Statistics",
			"active":      "stats",
			"totalLinks":  totalLinks,
			"totalClicks": totalClicks,
			"activeUsers": activeUsers,
			"topLinks":    topLinks,
			"recentLinks": recentLinks,
		})
	}
}