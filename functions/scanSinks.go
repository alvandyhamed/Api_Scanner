package functions

import (
	"SiteChecker/models"
	"context"
	"time"

	"github.com/chromedp/chromedp"
)

type sinkFinding struct {
	Kind       string `json:"kind"`
	SourceType string `json:"source_type"`
	SourceURL  string `json:"source_url"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
	Snippet    string `json:"snippet"`
}

func ScanSinks(ctx context.Context, pageURL, siteID string) ([]models.SinkDoc, error) {
	js := `
(async function(){
  function lineColFromIndex(text, idx){
    let line=1, col=1;
    for (let i=0;i<idx;i++){
      if (text[i] === '\n'){ line++; col=1; } else { col++; }
    }
    return {line, col};
  }
  function snippet(text, idx){
    const start = Math.max(0, idx - 120);
    const end   = Math.min(text.length, idx + 120);
    return text.slice(start, end);
  }
  const patterns = [
    {kind:"innerHTML",               re:/\.innerHTML\s*=/g},
    {kind:"dangerouslySetInnerHTML", re:/dangerouslySetInnerHTML\s*:/g},
    {kind:"eval",                    re:/\beval\s*\(/g},
    {kind:"newFunction",             re:/\bnew\s+Function\s*\(/g},
    {kind:"setTimeoutStr",           re:/\bsetTimeout\s*\(\s*(['"]).*?\1\s*(?:,|\))/g},
    {kind:"setIntervalStr",          re:/\bsetInterval\s*\(\s*(['"]).*?\1\s*(?:,|\))/g},
    {kind:"documentWrite",           re:/\bdocument\.write\s*\(/g},
    {kind:"prompt",                  re:/\bprompt\s*\(/g},
    {kind:"alert",                   re:/\balert\s*\(/g},
    {kind:"confirm",                 re:/\bconfirm\s*\(/g},
    {kind:"fetch",                   re:/\bfetch\s*\(/g},
    {kind:"XMLHttpRequest",          re:/\bXMLHttpRequest\b/g},
    {kind:"syncXHR",                 re:/\.open\s*\([^,]+,[^,]+,\s*false\s*\)/g},
    {kind:"localStorage",            re:/\blocalStorage\b/g},
    {kind:"sessionStorage",          re:/\bsessionStorage\b/g},
    {kind:"JSON.parse",              re:/\bJSON\.parse\s*\(/g},
    {kind:"JSON.stringify",          re:/\bJSON\.stringify\s*\(/g},
    {kind:"postMessageSend",         re:/\bpostMessage\s*\(/g},
    {kind:"postMessageRecv",         re:/addEventListener\s*\(\s*['"]message['"]/g},
    {kind:"onmessageHandler",        re:/\bonmessage\s*=/g},
    {kind:"inlineEventHandler",      re:/on[a-z]+\s*=\s*(['"]).*?\1/ig},
    {kind:"directDOM",               re:/document\.(getElementById|getElementsByClassName|querySelector(All)?)\s*\(/g},
    {kind:"heavyLoop",               re:/for\s*\(\s*var\s+i\s*=\s*0\s*;\s*i\s*<\s*\d{6,}\s*;|while\s*\(\s*true\s*\)/g}
  ];

  const results = [];

  // HTML
  const html = document.documentElement.outerHTML;
  for (const p of patterns) {
    for (const mm of html.matchAll(p.re)) {
      const idx = mm.index || 0;
      const lc = lineColFromIndex(html, idx);
      results.push({
        kind:p.kind, source_type:"html", source_url:location.href,
        line:lc.line, col:lc.col, snippet:snippet(html, idx)
      });
    }
  }

  // Scripts  ← این قسمت عوض شد
  const list = Array.from(document.scripts).map((s, i) => {
    const abs   = s.src ? new URL(s.src, location.href).href : null;
    const label = abs || (location.href + '#inline-' + (i+1));
    return { src: abs, label, text: s.src ? null : (s.text || "") };
  });

  for (const it of list) {
    if (it.src) {
      try {
        const resp = await fetch(it.src, { cache: "force-cache" });
        it.text = await resp.text();
      } catch (e) { it.text = ""; }
    }
  }

  for (const it of list) {
    if (!it.text) continue;
    for (const p of patterns) {
      const re = new RegExp(p.re.source, p.re.flags);
      for (const mm of it.text.matchAll(re)) {
        const idx = mm.index || 0;
        const lc = lineColFromIndex(it.text, idx);
        results.push({
          kind:p.kind,
          source_type: it.src ? "script" : "inline",
          source_url: it.label,   // ✅ همیشه label داریم
          line:lc.line, col:lc.col, snippet:snippet(it.text, idx)
        });
      }
    }
  }
  return results;
})()
`
	var found []sinkFinding
	if err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(js, &found)); err != nil {
		return nil, err
	}

	out := make([]models.SinkDoc, 0, len(found))
	now := time.Now()
	for _, f := range found {
		srcURL, srcType := normalizeSourceURL(pageURL, f.SourceURL)
		if f.SourceType == "" {
			f.SourceType = srcType
		}
		if srcURL != "" {
			f.SourceURL = srcURL
		}
		f.SourceURL = srcURL
		if f.SourceType == "" {
			f.SourceType = srcType
		}
		out = append(out, models.SinkDoc{
			SiteID:     siteID,
			PageURL:    pageURL,
			SourceType: f.SourceType,
			SourceURL:  f.SourceURL,
			Kind:       f.Kind,
			Line:       f.Line,
			Col:        f.Col,
			Snippet:    f.Snippet,
			DetectedAt: now,
		})
	}
	return out, nil
}
