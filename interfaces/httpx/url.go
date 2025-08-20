package httpx

import (
    "fmt"
    "net/http"
    "strings"
)

// ResolveBaseURL determines the base URL for links using the incoming request
// headers when available, falling back to the configured APP_BASE_URL.
// Priority: X-Forwarded-Proto/Host -> Request Host/TLS -> fallback.
func ResolveBaseURL(r *http.Request, fallback string) string {
    proto := r.Header.Get("X-Forwarded-Proto")
    if proto == "" {
        proto = r.Header.Get("X-Forwarded-Scheme")
    }
    if proto != "" {
        if idx := strings.IndexByte(proto, ','); idx >= 0 {
            proto = strings.TrimSpace(proto[:idx])
        }
    }
    host := r.Header.Get("X-Forwarded-Host")
    if host != "" {
        if idx := strings.IndexByte(host, ','); idx >= 0 {
            host = strings.TrimSpace(host[:idx])
        }
    }

    if host == "" {
        host = r.Host
    }
    if proto == "" {
        if r.TLS != nil {
            proto = "https"
        } else {
            proto = "http"
        }
    }

    if host != "" {
        return fmt.Sprintf("%s://%s", proto, strings.TrimRight(host, "/"))
    }
    return strings.TrimRight(strings.TrimSpace(fallback), "/")
}


