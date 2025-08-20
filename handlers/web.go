package handlers

import (
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"quickr/models"
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
func (h *AppHandler) HandleStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		links, _ := h.LinkService.ListLinks()
		var totalLinks int64 = int64(len(links))
		var totalClicks int64
		creators := map[string]struct{}{}
		for _, l := range links {
			totalClicks += int64(l.Clicks)
			creators[l.CreatorName] = struct{}{}
		}
		activeUsers := int64(len(creators))
		topLinks := append([]models.Link(nil), links...)
		sort.Slice(topLinks, func(i, j int) bool { return topLinks[i].Clicks > topLinks[j].Clicks })
		if len(topLinks) > 5 { topLinks = topLinks[:5] }
		recentLinks := append([]models.Link(nil), links...)
		sort.Slice(recentLinks, func(i, j int) bool { return recentLinks[i].CreatedAt.After(recentLinks[j].CreatedAt) })
		if len(recentLinks) > 5 { recentLinks = recentLinks[:5] }
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
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
func (h *AppHandler) HandleHot() gin.HandlerFunc {
	return func(c *gin.Context) {
		links, _ := h.LinkService.ListLinks()
		var recent []models.Link
		var top7d []models.Link
		var top30d []models.Link
		var topAll []models.Link
		cut24 := time.Now().Add(-24 * time.Hour)
		cut7d := time.Now().Add(-7 * 24 * time.Hour)
		cut30d := time.Now().Add(-30 * 24 * time.Hour)
		for _, l := range links {
			if l.CreatedAt.After(cut24) { recent = append(recent, l) }
			if l.CreatedAt.After(cut7d) { top7d = append(top7d, l) }
			if l.CreatedAt.After(cut30d) { top30d = append(top30d, l) }
		}
		sort.Slice(top7d, func(i, j int) bool { return top7d[i].Clicks > top7d[j].Clicks })
		if len(top7d) > 20 { top7d = top7d[:20] }
		sort.Slice(top30d, func(i, j int) bool { return top30d[i].Clicks > top30d[j].Clicks })
		if len(top30d) > 20 { top30d = top30d[:20] }
		topAll = append([]models.Link(nil), links...)
		sort.Slice(topAll, func(i, j int) bool { return topAll[i].Clicks > topAll[j].Clicks })
		if len(topAll) > 20 { topAll = topAll[:20] }
		emailVal, _ := c.Get("userEmail")
		roleVal, _ := c.Get("userRole")
		isAdmin := roleVal == "admin"
		c.HTML(http.StatusOK, "hot.html", gin.H{
			"active":    "hot",
			"recent":    recent,
			"top7d":     top7d,
			"top30d":    top30d,
			"topAll":    topAll,
			"userEmail": emailVal,
			"isAdmin":   isAdmin,
		})
	}
}