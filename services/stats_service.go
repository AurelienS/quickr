package services

import (
    "sort"
    "time"

    "quickr/models"
)

type StatsService struct {
    links *LinkService
}

func NewStatsService(linkService *LinkService) *StatsService {
    return &StatsService{links: linkService}
}

type StatsOverview struct {
    TotalLinks  int64
    TotalClicks int64
    ActiveUsers int64
    TopLinks    []models.Link
    RecentLinks []models.Link
}

func (s *StatsService) ComputeOverview() (StatsOverview, error) {
    links, err := s.links.ListLinks()
    if err != nil {
        return StatsOverview{}, err
    }
    var totalClicks int64
    creators := map[string]struct{}{}
    for _, l := range links {
        totalClicks += int64(l.Clicks)
        creators[l.CreatorName] = struct{}{}
    }

    topLinks := append([]models.Link(nil), links...)
    sort.Slice(topLinks, func(i, j int) bool { return topLinks[i].Clicks > topLinks[j].Clicks })
    if len(topLinks) > 5 { topLinks = topLinks[:5] }

    recentLinks := append([]models.Link(nil), links...)
    sort.Slice(recentLinks, func(i, j int) bool { return recentLinks[i].CreatedAt.After(recentLinks[j].CreatedAt) })
    if len(recentLinks) > 5 { recentLinks = recentLinks[:5] }

    return StatsOverview{
        TotalLinks:  int64(len(links)),
        TotalClicks: totalClicks,
        ActiveUsers: int64(len(creators)),
        TopLinks:    topLinks,
        RecentLinks: recentLinks,
    }, nil
}

type HotStats struct {
    Recent  []models.Link
    Top7d   []models.Link
    Top30d  []models.Link
    TopAll  []models.Link
}

func (s *StatsService) ComputeHot() (HotStats, error) {
    links, err := s.links.ListLinks()
    if err != nil {
        return HotStats{}, err
    }
    var recent []models.Link
    var top7d []models.Link
    var top30d []models.Link
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
    topAll := append([]models.Link(nil), links...)
    sort.Slice(topAll, func(i, j int) bool { return topAll[i].Clicks > topAll[j].Clicks })
    if len(topAll) > 20 { topAll = topAll[:20] }
    return HotStats{Recent: recent, Top7d: top7d, Top30d: top30d, TopAll: topAll}, nil
}


