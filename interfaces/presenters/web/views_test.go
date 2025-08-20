package web

import (
    "testing"
    "quickr/models"
    "quickr/services"
)

func TestHomeView(t *testing.T) {
    m := HomeView([]models.Link{{Alias: "a"}}, "user@example.com", true)
    if m["title"] != "Home" || m["active"] != "home" { t.Fatalf("unexpected meta: %+v", m) }
    if m["userEmail"] != "user@example.com" || m["isAdmin"] != true { t.Fatalf("unexpected user fields") }
}

func TestStatsView(t *testing.T) {
    ov := services.StatsOverview{TotalLinks: 3, TotalClicks: 5, ActiveUsers: 1}
    m := StatsView(ov, "u", false)
    if m["totalLinks"] != int64(3) || m["totalClicks"] != int64(5) || m["activeUsers"] != int64(1) {
        t.Fatalf("unexpected stats fields: %+v", m)
    }
}

func TestHotView(t *testing.T) {
    hot := services.HotStats{}
    m := HotView(hot, "u", false)
    if m["active"] != "hot" { t.Fatalf("unexpected active: %+v", m) }
}


