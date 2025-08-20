package session

import (
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
)

func TestSession_SignIn_Parse_Clear(t *testing.T) {
    gin.SetMode(gin.TestMode)
    m := NewManager([]byte("secret"), "session", time.Hour)

    // SignIn sets cookie
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    if err := m.SignIn(c, "user@example.com", "admin"); err != nil { t.Fatalf("signin error: %v", err) }
    cookies := w.Result().Cookies()
    if len(cookies) == 0 { t.Fatalf("expected a cookie to be set") }

    // Parse reads cookie
    r := httptest.NewRequest("GET", "/", nil)
    for _, ck := range cookies { r.AddCookie(ck) }
    w2 := httptest.NewRecorder()
    c2, _ := gin.CreateTestContext(w2)
    c2.Request = r
    email, role, err := m.Parse(c2)
    if err != nil || email != "user@example.com" || role != "admin" { t.Fatalf("unexpected parse: %q %q %v", email, role, err) }

    // Clear removes cookie
    w3 := httptest.NewRecorder()
    c3, _ := gin.CreateTestContext(w3)
    m.Clear(c3)
    if cleared := w3.Result().Cookies(); len(cleared) == 0 {
        t.Fatalf("expected clear to set expired cookie")
    }
}


