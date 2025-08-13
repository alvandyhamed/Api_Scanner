package functions

import "strings"

// خروجی: (URL نهایی, نوع منبع)
func normalizeSourceURL(pageURL, u string) (string, string) {
	if u == "" || u == "<anonymous>" {
		return pageURL + "#inline", "inline"
	}
	if strings.HasPrefix(u, "blob:") {
		return u, "blob"
	}
	if strings.HasPrefix(u, "data:") {
		return u, "data"
	}
	if strings.HasPrefix(u, "sc-") && !strings.Contains(u, "://") {
		return pageURL + "#" + u, "dynamic"
	}
	return u, "script"
}
