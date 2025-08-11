package functions

import (
	"context"
	"github.com/chromedp/chromedp"
)

func fetchAndExtractFromScripts(ctx context.Context, srcs []string, timeoutSec int) ([]string, []string) {
	var all []string
	var errs []string

	for _, u := range srcs {

		var code = `
(async function(){
  try {
    const ctrl = new AbortController();
    const t = setTimeout(() => ctrl.abort(), ` + itoa(timeoutSec*1000) + `);
    const resp = await fetch(` + jsonString(u) + `, { signal: ctrl.signal });
    clearTimeout(t);
    if(!resp.ok){ return { ok:false, err:"` + u + ` -> " + resp.status }; }
    const txt = await resp.text();

    // الگوی بدون lookbehind/forward: کوتیشن + مسیر نسبی + کوتیشن
    const re = /["'` + "`" + `]((?:\/|\.\.\/|\.\/)[a-zA-Z0-9_?&=\/\-\#\.]*)["'` + "`" + `]/g;
    const found = new Set();
    for (const m of txt.matchAll(re)) {
      if (m[1]) found.add(m[1]); // گروه ۱ = مسیر
    }
    return { ok:true, arr: Array.from(found) };
  } catch(e){
    return { ok:false, err:"` + u + ` -> " + (e && e.message ? e.message : String(e)) };
  }
})()
`
		type jsRes struct {
			Ok  bool     `json:"ok"`
			Arr []string `json:"arr"`
			Err string   `json:"err"`
		}
		var r jsRes
		err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(code, &r))
		if err != nil {
			errs = append(errs, u+" -> "+err.Error())
			continue
		}
		if !r.Ok {
			if r.Err != "" {
				errs = append(errs, r.Err)
			} else {
				errs = append(errs, u+" -> unknown error")
			}
			continue
		}
		for _, p := range r.Arr {
			if isRelativeLike(p) {
				all = append(all, p)
			}
		}
	}
	return uniqueStrings(all), errs
}
