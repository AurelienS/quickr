package handlers

import (
    "time"
    "quickr/services"
    "quickr/interfaces/session"
)

type RateLimiter interface { Allow(key string) bool }

type AppHandler struct {
    LinkService *services.LinkService
    AuthService *services.AuthService
    StatsService *services.StatsService
    RateLimiter RateLimiter
    AppBaseURL  string
    Session     session.Service
}

func NewAppHandler(linkSvc *services.LinkService, authSvc *services.AuthService, statsSvc *services.StatsService, limiter RateLimiter, appBaseURL string, sess session.Service) *AppHandler {
    return &AppHandler{
        LinkService:  linkSvc,
        AuthService:  authSvc,
        StatsService: statsSvc,
        RateLimiter:  limiter,
        AppBaseURL:   appBaseURL,
        Session:      sess,
    }
}

// Helper to construct default session when wiring without DI container
func NewDefaultSession(jwtSecret []byte) session.Service {
    return session.NewManager(jwtSecret, "session", 180*24*time.Hour)
}


