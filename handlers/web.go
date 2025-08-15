package handlers

import (
	"log"
	"net/http"
	"time"

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

		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"

		log.Printf("Found %d links", len(links))
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":     "Home",
			"active":    "home",
			"links":     links,
			"userEmail": emailVal,
			"isAdmin":   isAdmin,
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

		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"

		log.Printf("Stats: %d links, %d clicks, %d users", totalLinks, totalClicks, activeUsers)
		c.HTML(http.StatusOK, "stats.html", gin.H{
			"title":       "Statistics",
			"active":      "stats",
			"totalLinks":  totalLinks,
			"totalClicks": totalClicks,
			"activeUsers": activeUsers,
			"topLinks":    topLinks,
			"recentLinks": recentLinks,
			"userEmail":   emailVal,
			"isAdmin":     isAdmin,
		})
	}
}

// GET /hot
func HandleHot(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get recently added links (last 24h)
        var recent []models.Link
        db.Where("created_at > ?", time.Now().Add(-24*time.Hour)).
            Order("created_at desc").
            Find(&recent)

        // Get top links for last 7 days
        var top7d []models.Link
        db.Where("created_at > ?", time.Now().Add(-7*24*time.Hour)).
            Order("clicks desc").
            Limit(20).
            Find(&top7d)

        // Get top links for last 30 days
        var top30d []models.Link
        db.Where("created_at > ?", time.Now().Add(-30*24*time.Hour)).
            Order("clicks desc").
            Limit(20).
            Find(&top30d)

        // Get top links all time
        var topAll []models.Link
        db.Order("clicks desc").
            Limit(20).
            Find(&topAll)

        emailVal, _ := c.Get("userEmail")
        roleVal, _ := c.Get("userRole")
        isAdmin := roleVal == "admin"

        c.HTML(http.StatusOK, "hot.html", gin.H{
            "active":   "hot",
            "recent":   recent,
            "top7d":    top7d,
            "top30d":   top30d,
            "topAll":   topAll,
            "userEmail": emailVal,
            "isAdmin":   isAdmin,
        })
    }
}