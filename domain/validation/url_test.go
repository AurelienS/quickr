package validation

import "testing"

func TestIsValidHTTPURL(t *testing.T) {
    valid := []string{
        "https://example.com",
        "http://example.com",
        "https://example.com/path?q=1",
    }
    invalid := []string{
        "",
        "notaurl",
        "ftp://example.com",
        "http://",
        "https:///path",
    }

    for _, u := range valid {
        if !IsValidHTTPURL(u) {
            t.Errorf("expected valid: %q", u)
        }
    }
    for _, u := range invalid {
        if IsValidHTTPURL(u) {
            t.Errorf("expected invalid: %q", u)
        }
    }
}


