package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"quickr/interfaces/httpx"
)

// JWT cookie settings
const (
	cookieName   = "session"
	cookieMaxAge = 180 * 24 * time.Hour // ~6 months
)

type Claims struct { // deprecated; replaced by interfaces/session
	Role string `json:"role"`
}

func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("Config error: missing env JWT_SECRET")
		return nil, errors.New("JWT_SECRET is not set")
	}
	return []byte(secret), nil
}

func getAdminEmail() string {
	a := os.Getenv("ADMIN_EMAIL")
	if a == "" {
		log.Println("Config warning: missing env ADMIN_EMAIL")
	}
	return a
}

func getAdminName() string {
	n := os.Getenv("ADMIN_NAME")
	if n == "" {
		log.Println("Config warning: missing env ADMIN_NAME; defaulting to 'Admin'")
		return "Admin"
	}
	return n
}

// RequireAuth middleware validates the JWT session cookie and ensures the user is not disabled
func (h *AppHandler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		email, role, err := h.Session.Parse(c)
		if err != nil || strings.TrimSpace(email) == "" {
			accept := c.GetHeader("Accept")
			if strings.Contains(accept, "text/html") || c.Request.Method == http.MethodGet {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		if disabled, _ := h.AuthService.IsUserDisabled(email); disabled {
			h.Session.Clear(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account revoked"})
			return
		}
		c.Set("userEmail", email)
		c.Set("userRole", role)
		c.Next()
	}
}

func (h *AppHandler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin only"})
			return
		}
		c.Next()
	}
}

// GET /login renders a simple email input page
func (h *AppHandler) ShowLogin() gin.HandlerFunc {
	return func(c *gin.Context) { c.HTML(http.StatusOK, "login.html", gin.H{}) }
}

// POST /login requests a new magic link if email was invited
func (h *AppHandler) RequestMagicLink() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !h.RateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			email = strings.TrimSpace(strings.ToLower(c.PostForm("username")))
		}
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}

		// If admin email, ensure admin user and login immediately
		if strings.EqualFold(email, getAdminEmail()) {
			_ = h.AuthService.EnsureAdmin(email)
			if err := h.Session.SignIn(c, email, "admin"); err != nil {
				c.String(http.StatusInternalServerError, err.Error())
				return
			}
			c.Redirect(http.StatusFound, "/admin")
			return
		}

		base := httpx.ResolveBaseURL(c.Request, h.AppBaseURL)
		if err := h.AuthService.RequireAndSendMagicLink(email, base); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "email not invited"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Magic link sent if email is invited"})
	}
}

// GET /magic redeems token, issues JWT cookie, invalidates invite
func (h *AppHandler) RedeemMagicLink() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenParam := c.Query("token")
		if tokenParam == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}
		email, role, err := h.AuthService.RedeemMagicToken(tokenParam, func(e string) bool { return strings.EqualFold(e, getAdminEmail()) })
		if err != nil {
			c.String(http.StatusUnauthorized, err.Error())
			return
		}
		if err := h.Session.SignIn(c, email, role); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.Redirect(http.StatusFound, "/")
	}
}

// POST /logout clears the cookie
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.SetCookie(cookieName, "", -1, "/", "", true, true)
		c.Redirect(http.StatusFound, "/login")
	}
}