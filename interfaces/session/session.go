package session

import (
    "errors"
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    Role string `json:"role"`
    jwt.RegisteredClaims
}

type Manager struct {
    secret     []byte
    cookieName string
    maxAge     time.Duration
}

func NewManager(secret []byte, cookieName string, maxAge time.Duration) *Manager {
    return &Manager{secret: secret, cookieName: cookieName, maxAge: maxAge}
}

// Interface for handlers to depend on
type Service interface {
    Parse(c *gin.Context) (email string, role string, err error)
    SignIn(c *gin.Context, email string, role string) error
    Clear(c *gin.Context)
}

func (m *Manager) Parse(c *gin.Context) (string, string, error) {
    cookie, err := c.Cookie(m.cookieName)
    if err != nil {
        return "", "", err
    }
    token, err := jwt.ParseWithClaims(cookie, &Claims{}, func(token *jwt.Token) (interface{}, error) { return m.secret, nil })
    if err != nil || !token.Valid {
        return "", "", errors.New("invalid session")
    }
    claims, ok := token.Claims.(*Claims)
    if !ok {
        return "", "", errors.New("invalid session")
    }
    return claims.Subject, claims.Role, nil
}

func (m *Manager) SignIn(c *gin.Context, email string, role string) error {
    claims := &Claims{Role: role, RegisteredClaims: jwt.RegisteredClaims{Subject: email, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.maxAge))}}
    jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := jwtToken.SignedString(m.secret)
    if err != nil {
        log.Println("failed to sign token:", err)
        return errors.New("failed to sign token")
    }
    c.SetSameSite(http.SameSiteLaxMode)
    c.SetCookie(m.cookieName, signed, int(m.maxAge.Seconds()), "/", "", true, true)
    return nil
}

func (m *Manager) Clear(c *gin.Context) {
    c.SetCookie(m.cookieName, "", -1, "/", "", true, true)
}


