package services

import (
    "testing"
    "time"

    "quickr/models"
)

func TestStatsService_ComputeOverview(t *testing.T) {
    now := time.Now()
    repo := &fakeRepo{
        ListAllFunc: func() ([]models.Link, error) {
            return []models.Link{
                {Alias: "a", Clicks: 10, CreatorName: "u1", CreatedAt: now.Add(-time.Hour)},
                {Alias: "b", Clicks: 5, CreatorName: "u2", CreatedAt: now.Add(-2 * time.Hour)},
                {Alias: "c", Clicks: 20, CreatorName: "u1", CreatedAt: now.Add(-30 * time.Minute)},
            }, nil
        },
    }
    ls := NewLinkService(repo)
    ss := NewStatsService(ls)

    ov, err := ss.ComputeOverview()
    if err != nil { t.Fatalf("unexpected err: %v", err) }
    if ov.TotalLinks != 3 { t.Fatalf("expected 3 links, got %d", ov.TotalLinks) }
    if ov.TotalClicks != 35 { t.Fatalf("expected 35 clicks, got %d", ov.TotalClicks) }
    if ov.ActiveUsers != 2 { t.Fatalf("expected 2 active users, got %d", ov.ActiveUsers) }
    if len(ov.TopLinks) != 3 || ov.TopLinks[0].Alias != "c" || ov.TopLinks[0].Clicks != 20 {
        t.Fatalf("unexpected top links order: %+v", ov.TopLinks)
    }
    if len(ov.RecentLinks) != 3 || ov.RecentLinks[0].Alias != "c" {
        t.Fatalf("unexpected recent links order: %+v", ov.RecentLinks)
    }
}

func TestStatsService_ComputeHot(t *testing.T) {
    now := time.Now()
    repo := &fakeRepo{
        ListAllFunc: func() ([]models.Link, error) {
            return []models.Link{
                {Alias: "old", Clicks: 100, CreatedAt: now.Add(-40 * 24 * time.Hour)},
                {Alias: "recent", Clicks: 1, CreatedAt: now.Add(-2 * time.Hour)},
                {Alias: "d7a", Clicks: 10, CreatedAt: now.Add(-2 * 24 * time.Hour)},
                {Alias: "d30a", Clicks: 5, CreatedAt: now.Add(-20 * 24 * time.Hour)},
                {Alias: "topall", Clicks: 200, CreatedAt: now.Add(-100 * time.Hour)},
            }, nil
        },
    }
    ls := NewLinkService(repo)
    ss := NewStatsService(ls)

    hot, err := ss.ComputeHot()
    if err != nil { t.Fatalf("unexpected err: %v", err) }

    if len(hot.Recent) == 0 || hot.Recent[0].Alias != "recent" {
        t.Fatalf("unexpected recent: %+v", hot.Recent)
    }
    if len(hot.Top7d) == 0 || hot.Top7d[0].Alias != "topall" {
        t.Fatalf("unexpected top7d: %+v", hot.Top7d)
    }
    if len(hot.Top30d) == 0 || hot.Top30d[0].Alias != "topall" {
        t.Fatalf("unexpected top30d order: %+v", hot.Top30d)
    }
    if len(hot.TopAll) == 0 || hot.TopAll[0].Alias != "topall" {
        t.Fatalf("unexpected topall: %+v", hot.TopAll)
    }
}


