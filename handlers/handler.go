package handlers

import (
    "quickr/services"
)

type AppHandler struct {
    LinkService *services.LinkService
    AuthService *services.AuthService

    Mailer      *SendinblueClient
    RateLimiter *IPLimiter
    AppBaseURL  string
}

func NewAppHandler(linkSvc *services.LinkService, authSvc *services.AuthService, mailer *SendinblueClient, limiter *IPLimiter, appBaseURL string) *AppHandler {
    return &AppHandler{
        LinkService:  linkSvc,
        AuthService:  authSvc,
        Mailer:       mailer,
        RateLimiter:  limiter,
        AppBaseURL:   appBaseURL,
    }
}


