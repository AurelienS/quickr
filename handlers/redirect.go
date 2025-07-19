package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quickr/models"
)

func HandleRedirect(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		alias := c.Param("alias")
		log.Printf("[DEBUG] HandleRedirect called for alias: %s", alias)

		var link models.Link
		// Use exact match and ensure not deleted
		result := db.Debug().Where("alias = ? AND deleted_at IS NULL", alias).First(&link)
		if result.Error != nil {
			log.Printf("[ERROR] Error finding link: %v", result.Error)
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		log.Printf("[DEBUG] Found link: ID=%d, Alias=%s, URL=%s", link.ID, link.Alias, link.URL)

		// Increment clicks atomically
		updateResult := db.Model(&link).Update("clicks", gorm.Expr("clicks + ?", 1))
		if updateResult.Error != nil {
			log.Printf("[ERROR] Error updating clicks: %v", updateResult.Error)
		}

		log.Printf("[DEBUG] Redirecting to: %s", link.URL)
		// Use 302 Found instead of 301 Moved Permanently to avoid browser caching
		c.Redirect(http.StatusFound, link.URL)
	}
}