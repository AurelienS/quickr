package handlers

import "strings"

// reservedAliases contains paths that must not be used as link aliases
var reservedAliases = map[string]struct{}{
    "":             {},
    "login":        {},
    "magic":        {},
    "logout":       {},
    "admin":        {},
    "stats":        {},
    "hot":          {},
    "api":          {},
    "static":       {},
    "favicon.ico":  {},
    "robots.txt":   {},
    "go":           {},
}

func isReservedAlias(alias string) bool {
    _, ok := reservedAliases[strings.ToLower(strings.Trim(alias, "/"))]
    return ok
}

// IsReservedAliasPublic is an exported helper for router guards
func IsReservedAliasPublic(alias string) bool { return isReservedAlias(alias) }


