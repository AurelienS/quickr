package httpx

import (
    "crypto/tls"
    "net/http"
    "testing"
)

func TestResolveBaseURL(t *testing.T) {
    r := &http.Request{Header: make(http.Header)}
    r.Host = "example.com"
    if got := ResolveBaseURL(r, ""); got != "http://example.com" {
        t.Fatalf("got %q", got)
    }

    r = &http.Request{Header: make(http.Header)}
    r.Host = "example.com"
    r.TLS = &tls.ConnectionState{}
    if got := ResolveBaseURL(r, ""); got != "https://example.com" {
        t.Fatalf("got %q", got)
    }

    r = &http.Request{Header: make(http.Header)}
    r.Header.Set("X-Forwarded-Proto", "https")
    r.Header.Set("X-Forwarded-Host", "cdn.example.com")
    if got := ResolveBaseURL(r, ""); got != "https://cdn.example.com" {
        t.Fatalf("got %q", got)
    }

    r = &http.Request{Header: make(http.Header)}
    r.Header.Set("X-Forwarded-Proto", "https, http")
    r.Header.Set("X-Forwarded-Host", "a.example.com, b.example.com")
    if got := ResolveBaseURL(r, ""); got != "https://a.example.com" {
        t.Fatalf("got %q", got)
    }

    r = &http.Request{Header: make(http.Header)}
    if got := ResolveBaseURL(r, " https://fallback.example.com/ "); got != "https://fallback.example.com" {
        t.Fatalf("got %q", got)
    }
}


