package core

import "strings"

// CleanString trims all leading and trailing whitespace in `s` and optionally lowers it.
func CleanString(s string, lower ...bool) string {
	s = strings.TrimSpace(s)
	if len(lower) > 0 && lower[0] {
		return strings.ToLower(s)
	}
	return s
}
