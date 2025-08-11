package functions

import "regexp"

var htmlPathRe = regexp.MustCompile("(?m)[\"'\\x60]((?:/|\\.\\./|\\./)[a-zA-Z0-9_?&=\\./\\-#]*)[\"'\\x60]")

func extractPathsFromHTML(html string) []string {
	matches := htmlPathRe.FindAllStringSubmatch(html, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			p := m[1]
			if isRelativeLike(p) {
				out = append(out, p)
			}
		}
	}
	return uniqueStrings(out)
}
