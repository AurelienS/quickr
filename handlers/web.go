package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	webview "quickr/interfaces/presenters/web"
)

// HandleHome renders the homepage with all links
func (h *AppHandler) HandleHome() gin.HandlerFunc {
	return func(c *gin.Context) {
		links, err := h.LinkService.ListLinks()
		if err != nil {
			log.Printf("Service error: %v", err)
			c.String(http.StatusInternalServerError, "Service error")
			return
		}
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		log.Printf("Found %d links", len(links))
		c.HTML(http.StatusOK, "index.html", webview.HomeView(links, emailVal.(string), isAdmin))
	}
}

// HandleStats renders the stats page
func (h *AppHandler) HandleStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		overview, err := h.StatsService.ComputeOverview()
		if err != nil {
			c.String(http.StatusInternalServerError, "Service error")
			return
		}
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		c.HTML(http.StatusOK, "stats.html", webview.StatsView(overview, emailVal.(string), isAdmin))
	}
}

// GET /hot
func (h *AppHandler) HandleHot() gin.HandlerFunc {
	return func(c *gin.Context) {
		hot, err := h.StatsService.ComputeHot()
		if err != nil {
			c.String(http.StatusInternalServerError, "Service error")
			return
		}
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		c.HTML(http.StatusOK, "hot.html", webview.HotView(hot, emailVal.(string), isAdmin))
	}
}