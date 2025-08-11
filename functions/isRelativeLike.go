package functions

import "strings"

func isRelativeLike(s string) bool {
	if len(s) <= 1 || len(s) >= 200 {
		return false
	}
	if !(strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../")) {
		return false
	}
	// ASCII printable only
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7E {
			return false
		}
	}
	return true
}
