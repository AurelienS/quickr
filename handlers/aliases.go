package handlers

import "quickr/domain/reserved"

// IsReservedAliasPublic is an exported helper for router guards
func IsReservedAliasPublic(alias string) bool { return reserved.IsReservedAlias(alias) }


