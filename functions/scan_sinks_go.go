package functions

import (
	"SiteChecker/models"
	"regexp"
	"strings"
	"time"
)

var (
	reInnerHTML      = regexp.MustCompile(`\.innerHTML\s*=`)
	reDangerousSet   = regexp.MustCompile(`dangerouslySetInnerHTML\s*:`)
	reEval           = regexp.MustCompile(`\beval\s*\(`)
	reNewFunction    = regexp.MustCompile(`\bnew\s+Function\s*\(`)
	reStrArg         = `(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')` // رشتهٔ JS
	reSetTimeoutStr  = regexp.MustCompile(`\bsetTimeout\s*\(\s*` + reStrArg)
	reSetIntervalStr = regexp.MustCompile(`\bsetInterval\s*\(\s*` + reStrArg)
	reDocWrite       = regexp.MustCompile(`\bdocument\.write\s*\(`)
	rePrompt         = regexp.MustCompile(`\bprompt\s*\(`)
	reAlert          = regexp.MustCompile(`\balert\s*\(`)
	reConfirm        = regexp.MustCompile(`\bconfirm\s*\(`)
	reFetch          = regexp.MustCompile(`\bfetch\s*\(`)
	reXHR            = regexp.MustCompile(`\bXMLHttpRequest\b`)
	reSyncXHR        = regexp.MustCompile(`\.open\s*\([^,]+,[^,]+,\s*false\s*\)`)
	reLocalStorage   = regexp.MustCompile(`\blocalStorage\b`)
	reSessionStorage = regexp.MustCompile(`\bsessionStorage\b`)
	reJSONParse      = regexp.MustCompile(`\bJSON\.parse\s*\(`)
	reJSONStr        = regexp.MustCompile(`\bJSON\.stringify\s*\(`)
	reDirectDOM      = regexp.MustCompile(`document\.(getElementById|getElementsByClassName|querySelector(All)?)\s*\(`)
	reHeavyLoop      = regexp.MustCompile(`for\s*\([^;]*;[^;]*<\s*\d{6,}\s*;|while\s*\(\s*true\s*\)`)
	// postMessage
	rePostMessageSend = regexp.MustCompile(`\.postMessage\s*\(`)
	rePostMessageRecv = regexp.MustCompile(`addEventListener\s*\(\s*['"]message['"]|\.onmessage\s*=`)
	// HTML inline handlers
	reInlineHandler = regexp.MustCompile(`on[a-zA-Z]+\s*=\s*` + reStrArg)
)

func lineCol(s string, idx int) (int, int) {
	line, col := 1, 1
	for i := 0; i < idx && i < len(s); i++ {
		if s[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}
func snippetAround(s string, idx int) string {
	start := idx - 120
	if start < 0 {
		start = 0
	}
	end := idx + 120
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

func scanOne(kind, sourceType, sourceURL, text string, re *regexp.Regexp, out *[]models.SinkDoc, siteID, pageURL string) {
	for _, loc := range re.FindAllStringIndex(text, -1) {
		if len(loc) != 2 {
			continue
		}
		l, c := lineCol(text, loc[0])
		*out = append(*out, models.SinkDoc{
			SiteID:     siteID,
			PageURL:    pageURL,
			SourceType: sourceType,
			SourceURL:  sourceURL,
			Kind:       kind,
			Line:       l,
			Col:        c,
			Snippet:    snippetAround(text, loc[0]),
			DetectedAt: time.Now(),
		})
	}
}

func ScanSinksGo(html string, scripts map[string]string, pageURL, siteID string) []models.SinkDoc {
	var out []models.SinkDoc

	// HTML
	srcURL := pageURL
	srcType := "html"
	htmlL := strings.ToLower(html)
	_ = htmlL

	scanOne("inlineEventHandler", srcType, srcURL, html, reInlineHandler, &out, siteID, pageURL)

	scanOne("postMessageRecv", srcType, srcURL, html, rePostMessageRecv, &out, siteID, pageURL)

	scanOne("directDOM", srcType, srcURL, html, reDirectDOM, &out, siteID, pageURL)

	for u, code := range scripts {
		srcType = "script"
		srcURL = u
		if normU, normT := normalizeSourceURL(pageURL, srcURL); normU != "" {
			srcURL = normU
			if srcType == "" || srcType == "script" { // فقط اگر لازم بود
				srcType = normT
			}
		}
		scanOne("innerHTML", srcType, srcURL, code, reInnerHTML, &out, siteID, pageURL)
		scanOne("dangerouslySetInnerHTML", srcType, srcURL, code, reDangerousSet, &out, siteID, pageURL)
		scanOne("eval", srcType, srcURL, code, reEval, &out, siteID, pageURL)
		scanOne("newFunction", srcType, srcURL, code, reNewFunction, &out, siteID, pageURL)
		scanOne("setTimeoutStr", srcType, srcURL, code, reSetTimeoutStr, &out, siteID, pageURL)
		scanOne("setIntervalStr", srcType, srcURL, code, reSetIntervalStr, &out, siteID, pageURL)
		scanOne("documentWrite", srcType, srcURL, code, reDocWrite, &out, siteID, pageURL)
		scanOne("prompt", srcType, srcURL, code, rePrompt, &out, siteID, pageURL)
		scanOne("alert", srcType, srcURL, code, reAlert, &out, siteID, pageURL)
		scanOne("confirm", srcType, srcURL, code, reConfirm, &out, siteID, pageURL)
		scanOne("fetch", srcType, srcURL, code, reFetch, &out, siteID, pageURL)
		scanOne("XMLHttpRequest", srcType, srcURL, code, reXHR, &out, siteID, pageURL)
		scanOne("syncXHR", srcType, srcURL, code, reSyncXHR, &out, siteID, pageURL)
		scanOne("localStorage", srcType, srcURL, code, reLocalStorage, &out, siteID, pageURL)
		scanOne("sessionStorage", srcType, srcURL, code, reSessionStorage, &out, siteID, pageURL)
		scanOne("JSON.parse", srcType, srcURL, code, reJSONParse, &out, siteID, pageURL)
		scanOne("JSON.stringify", srcType, srcURL, code, reJSONStr, &out, siteID, pageURL)
		scanOne("postMessageSend", srcType, srcURL, code, rePostMessageSend, &out, siteID, pageURL)
		scanOne("postMessageRecv", srcType, srcURL, code, rePostMessageRecv, &out, siteID, pageURL)
		scanOne("directDOM", srcType, srcURL, code, reDirectDOM, &out, siteID, pageURL)
		scanOne("heavyLoop", srcType, srcURL, code, reHeavyLoop, &out, siteID, pageURL)
	}
	return out
}
