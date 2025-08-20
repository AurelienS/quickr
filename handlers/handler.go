package handlers

import (
    "quickr/services"
)

type RateLimiter interface { Allow(key string) bool }

type AppHandler struct {
    LinkService *services.LinkService
    AuthService *services.AuthService
    StatsService *services.StatsService
    RateLimiter RateLimiter
    AppBaseURL  string
}

func NewAppHandler(linkSvc *services.LinkService, authSvc *services.AuthService, statsSvc *services.StatsService, limiter RateLimiter, appBaseURL string) *AppHandler {
    return &AppHandler{
        LinkService:  linkSvc,
        AuthService:  authSvc,
        StatsService: statsSvc,
        RateLimiter:  limiter,
        AppBaseURL:   appBaseURL,
    }
}


