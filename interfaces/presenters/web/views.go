package web

import (
	"quickr/models"
	"quickr/services"
)

func HomeView(links []models.Link, email string, isAdmin bool) map[string]any {
	return map[string]any{
		"title":     "Home",
		"active":    "home",
		"links":     links,
		"userEmail": email,
		"isAdmin":   isAdmin,
	}
}

func StatsView(overview services.StatsOverview, email string, isAdmin bool) map[string]any {
	return map[string]any{
		"title":       "Statistics",
		"active":      "stats",
		"totalLinks":  overview.TotalLinks,
		"totalClicks": overview.TotalClicks,
		"activeUsers": overview.ActiveUsers,
		"topLinks":    overview.TopLinks,
		"recentLinks": overview.RecentLinks,
		"userEmail":   email,
		"isAdmin":     isAdmin,
	}
}

func HotView(hot services.HotStats, email string, isAdmin bool) map[string]any {
	return map[string]any{
		"active":    "hot",
		"recent":    hot.Recent,
		"top7d":     hot.Top7d,
		"top30d":    hot.Top30d,
		"topAll":    hot.TopAll,
		"userEmail": email,
		"isAdmin":   isAdmin,
	}
}


