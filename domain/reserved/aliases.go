package reserved

import "strings"

var aliases = map[string]struct{}{
    "":            {},
    "login":       {},
    "magic":       {},
    "logout":      {},
    "admin":       {},
    "stats":       {},
    "hot":         {},
    "api":         {},
    "static":      {},
    "favicon.ico": {},
    "robots.txt":  {},
    "go":          {},
}

func IsReservedAlias(alias string) bool {
    _, ok := aliases[strings.ToLower(strings.Trim(alias, "/"))]
    return ok
}


