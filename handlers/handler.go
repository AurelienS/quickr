package handlers

import (
    "quickr/services"
)

type AppHandler struct {
    LinkService *services.LinkService
    AuthService *services.AuthService
    RateLimiter *IPLimiter
    AppBaseURL  string
}

func NewAppHandler(linkSvc *services.LinkService, authSvc *services.AuthService, limiter *IPLimiter, appBaseURL string) *AppHandler {
    return &AppHandler{
        LinkService:  linkSvc,
        AuthService:  authSvc,
        RateLimiter:  limiter,
        AppBaseURL:   appBaseURL,
    }
}


