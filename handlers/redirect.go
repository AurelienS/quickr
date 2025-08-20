package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *AppHandler) HandleRedirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		alias := c.Param("alias")
		log.Printf("[DEBUG] HandleRedirect called for alias: %s", alias)

		link, err := h.LinkService.FindByAlias(alias)
		if err != nil {
			log.Printf("[ERROR] Error finding link: %v", err)
			c.String(http.StatusNotFound, "Link not found")
			return
		}

		if err := h.LinkService.IncrementClicks(link.ID); err != nil {
			log.Printf("[ERROR] Error updating clicks: %v", err)
		}

		log.Printf("[DEBUG] Redirecting to: %s", link.URL)
		// Use 302 Found instead of 301 Moved Permanently to avoid browser caching
		c.Redirect(http.StatusFound, link.URL)
	}
}