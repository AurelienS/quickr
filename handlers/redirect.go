package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"quickr/models"
)

func HandleRedirect(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		alias := c.Param("alias")

		var link models.Link
		result := db.First(&link, "alias = ?", alias)
		if result.Error != nil {
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		// Increment clicks atomically
		db.Model(&link).Update("clicks", gorm.Expr("clicks + ?", 1))

		c.Redirect(http.StatusMovedPermanently, link.URL)
	}
}