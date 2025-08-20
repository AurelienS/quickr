package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// JWT cookie settings
const (
	cookieName   = "session"
	cookieMaxAge = 180 * 24 * time.Hour // ~6 months
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
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

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RequireAuth middleware validates the JWT session cookie and ensures the user is not disabled
func (h *AppHandler) RequireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cookieName)
		if err != nil || strings.TrimSpace(cookie) == "" {
			accept := c.GetHeader("Accept")
			if strings.Contains(accept, "text/html") || c.Request.Method == http.MethodGet {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		secret, err := getJWTSecret()
		if err != nil {
			log.Println("Auth error: JWT secret not configured")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server configuration error"})
			return
		}
		token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(token *jwt.Token) (interface{}, error) { return secret, nil })
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		if disabled, _ := h.AuthService.IsUserDisabled(claims.Subject); disabled {
			c.SetCookie(cookieName, "", -1, "/", "", true, true)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account revoked"})
			return
		}
		c.Set("userEmail", claims.Subject)
		c.Set("userRole", claims.Role)
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
func (h *AppHandler) RequestMagicLink(db *gorm.DB) gin.HandlerFunc {
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
			user, _ := h.AuthService.GetUserByEmail(email)
			secret, err := getJWTSecret()
			if err != nil {
				log.Println("Auth error: JWT secret not configured")
				c.String(http.StatusInternalServerError, "server configuration error")
				return
			}
			claims := &Claims{Role: "admin", RegisteredClaims: jwt.RegisteredClaims{Subject: user.Email, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(cookieMaxAge))}}
			jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, err := jwtToken.SignedString(secret)
			if err != nil {
				c.String(http.StatusInternalServerError, "failed to sign token")
				return
			}
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie(cookieName, signed, int(cookieMaxAge.Seconds()), "/", "", true, true)
			c.Redirect(http.StatusFound, "/admin")
			return
		}

		base := resolveBaseURL(c.Request, h.AppBaseURL)
		if err := h.AuthService.RequireAndSendMagicLink(email, base); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "email not invited"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Magic link sent if email is invited"})
	}
}

// GET /magic redeems token, issues JWT cookie, invalidates invite
func (h *AppHandler) RedeemMagicLink(db *gorm.DB) gin.HandlerFunc {
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
		secret, err := getJWTSecret()
		if err != nil {
			log.Println("Auth error: JWT secret not configured")
			c.String(http.StatusInternalServerError, "server configuration error")
			return
		}
		claims := &Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: email, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(cookieMaxAge))}}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := jwtToken.SignedString(secret)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to sign token")
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(cookieName, signed, int(cookieMaxAge.Seconds()), "/", "", true, true)
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