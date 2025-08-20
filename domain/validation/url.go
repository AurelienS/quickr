package validation

import "net/url"

func IsValidHTTPURL(urlStr string) bool {
    u, err := url.Parse(urlStr)
    return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}


