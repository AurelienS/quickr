package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"quickr/models"
)

// JWT cookie settings
const (
	cookieName   = "session"
	cookieMaxAge = 180 * 24 * time.Hour // ~6 months
)

// Claims defines our JWT payload
// subject: user email; role: user/admin
// token id (jti) for traceability, issued at/expiry for validation
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

// generateToken creates a secure random token for invitations
func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// RequireAuth middleware validates the JWT session cookie and ensures the user is not disabled
func RequireAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(cookieName)
		if err != nil || strings.TrimSpace(cookie) == "" {
			// For browsers requesting HTML, redirect to login instead of JSON
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
		token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}
		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		// Check if the user has been disabled (revoked at the account level)
		var user models.User
		if err := db.Where("email = ?", claims.Subject).First(&user).Error; err == nil {
			if user.Disabled {
				// Invalidate cookie and reject
				c.SetCookie(cookieName, "", -1, "/", "", true, true)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account revoked"})
				return
			}
		}
		// set context
		c.Set("userEmail", claims.Subject)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

// RequireAdmin ensures the user is admin
func RequireAdmin() gin.HandlerFunc {
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
func ShowLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{})
	}
}

// POST /login requests a new magic link if email was invited
func RequestMagicLink(db *gorm.DB, rateLimiter *IPLimiter, mailer *SendinblueClient, appBaseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rateLimiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
		if email == "" {
			// Some password managers submit `username` for the email field
			email = strings.TrimSpace(strings.ToLower(c.PostForm("username")))
		}
		if email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email required"})
			return
		}

		// If admin email, auto-login without sending email
		if strings.EqualFold(email, getAdminEmail()) {
			var user models.User
			if err := db.Where("email = ?", email).First(&user).Error; err != nil {
				user = models.User{Email: email, Role: "admin"}
			}
			user.Role = "admin"
			user.LastLogin = time.Now()
			if err := db.Save(&user).Error; err != nil {
				c.String(http.StatusInternalServerError, "failed to save user")
				return
			}
			secret, err := getJWTSecret()
			if err != nil {
				log.Println("Auth error: JWT secret not configured")
				c.String(http.StatusInternalServerError, "server configuration error")
				return
			}
			claims := &Claims{
				Role: "admin",
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   user.Email,
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(cookieMaxAge)),
					ID:        fmt.Sprintf("%d", user.ID),
				},
			}
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

		// must be previously invited, except allow bootstrap for admin email
		var inv models.Invitation
		err := db.Where("email = ? AND status <> ?", email, "revoked").Order("created_at desc").First(&inv).Error
		if err != nil {
			if strings.EqualFold(email, getAdminEmail()) {
				inv = models.Invitation{Email: email, Status: "pending", ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
				_ = db.Create(&inv).Error
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "email not invited"})
				return
			}
		}

		// create new token and mark previous pending/sent as revoked
		db.Model(&models.Invitation{}).Where("email = ? AND status IN ?", email, []string{"pending", "sent"}).Update("status", "revoked")

		token, err := generateToken(32)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		newInv := models.Invitation{
			Email:     email,
			Token:     token,
			Status:    "sent",
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		if err := db.Create(&newInv).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save token"})
			return
		}

		link := fmt.Sprintf("%s/magic?token=%s", strings.TrimRight(appBaseURL, "/"), token)
		if err := mailer.SendMagicLink(email, link); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send email"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Magic link sent if email is invited"})
	}
}

// GET /magic redeems token, issues JWT cookie, invalidates invite
func RedeemMagicLink(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenParam := c.Query("token")
		if tokenParam == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}
		var inv models.Invitation
		if err := db.Where("token = ?", tokenParam).First(&inv).Error; err != nil {
			c.String(http.StatusUnauthorized, "invalid token")
			return
		}
		if inv.Status == "used" || inv.Status == "revoked" || inv.ExpiresAt.Before(time.Now()) {
			c.String(http.StatusUnauthorized, "token expired or used")
			return
		}

		// upsert user
		var user models.User
		if err := db.Where("email = ?", inv.Email).First(&user).Error; err != nil {
			user = models.User{Email: inv.Email, Role: "user"}
		}
		// admin auto-assign
		if strings.EqualFold(user.Email, getAdminEmail()) {
			user.Role = "admin"
		}
		user.LastLogin = time.Now()
		if err := db.Save(&user).Error; err != nil {
			c.String(http.StatusInternalServerError, "failed to save user")
			return
		}

		// mark invite used
		now := time.Now()
		db.Model(&inv).Updates(map[string]interface{}{"status": "used", "used_at": &now})

		// issue JWT
		secret, err := getJWTSecret()
		if err != nil {
			log.Println("Auth error: JWT secret not configured")
			c.String(http.StatusInternalServerError, "server configuration error")
			return
		}
		claims := &Claims{
			Role: user.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   user.Email,
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(cookieMaxAge)),
				ID:        fmt.Sprintf("%d", user.ID),
			},
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := jwtToken.SignedString(secret)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to sign token")
			return
		}

		secure := true
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(cookieName, signed, int(cookieMaxAge.Seconds()), "/", "", secure, true)

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