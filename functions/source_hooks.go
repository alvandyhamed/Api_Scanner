package functions

import (
	"context"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"SiteChecker/models"
)

// برچسب‌گذاری روی eval/new Function/timeout/... (اختیاری ولی توصیه می‌شود)
func InstallSourceURLHooks() chromedp.Action {
	const js = `(function(){
try{
  if (window.__SC_HOOKS__) return; window.__SC_HOOKS__=true;
  let __scId=0;
  function tag(code, label){
    if (typeof code!=='string') return code;
    __scId++;
    const mark = '\n//# sourceURL='+label+'-'+__scId+'.js';
    return code.endsWith(mark)? code : (code+mark);
  }
  const _eval = window.eval;
  window.eval = function(code){ return _eval(tag(code,'sc-eval')); };
  const _Fn = window.Function;
  window.Function = function(...args){
    const body = args.pop();
    args.push(tag(body,'sc-fn'));
    return _Fn.apply(this,args);
  };
  const _to = window.setTimeout;
  window.setTimeout = function(h,t,...rest){
    if (typeof h==='string') h=tag(h,'sc-timeout');
    return _to(h,t,...rest);
  };
  const _iv = window.setInterval;
  window.setInterval = function(h,t,...rest){
    if (typeof h==='string') h=tag(h,'sc-interval');
    return _iv(h,t,...rest);
  };
  const _Blob = window.Blob;
  window.Blob = function(parts,opts){
    try{
      if (opts && /javascript|ecmascript/i.test(String(opts.type||''))){
        parts=(parts||[]).map(p=> typeof p==='string'? (p+'\n//# sourceURL=sc-blob-'+(++__scId)+'.js'): p);
      }
    }catch(e){}
    return new _Blob(parts,opts);
  };
}catch(e){}
})();`
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(js).Do(ctx)
		return err
	})
}

// Capture postMessage send/recv با محل تعریف از stacktrace
func InstallRuntimePostMessageHook() chromedp.Action {
	const js = `(function(){
try{
  if (window.__SC_PM_HOOK__) return; window.__SC_PM_HOOK__=true;
  function where(){
    try{
      const st = (new Error()).stack || '';
      const lines = st.split('\n').slice(1);
      for (const ln of lines){
        let m = ln.match(/\(?((?:blob:|data:|https?:\/\/)[^():]+):(\d+):(\d+)\)?/);
        if (m) return {url:m[1], line:+m[2], col:+m[3]};
      }
    }catch(e){}
    return {url: location.href + '#inline', line:0, col:0};
  }
  window.__SC_RUNTIMESINKS = window.__SC_RUNTIMESINKS || [];

  const _add = EventTarget.prototype.addEventListener;
  EventTarget.prototype.addEventListener = function(type, listener, opts){
    if (type === 'message'){
      const loc = where();
      window.__SC_RUNTIMESINKS.push({kind:'postMessageListen', url:loc.url, line:loc.line, col:loc.col});
    }
    return _add.call(this, type, listener, opts);
  };

  Object.defineProperty(window, 'onmessage', {
    set(v){
      const loc = where();
      window.__SC_RUNTIMESINKS.push({kind:'postMessageListen', url:loc.url, line:loc.line, col:loc.col});
      return Reflect.set(this, '__onmessage', v);
    },
    get(){ return this.__onmessage; }
  });

  const _pm = window.postMessage;
  window.postMessage = function(){
    const loc = where();
    window.__SC_RUNTIMESINKS.push({kind:'postMessageSend', url:loc.url, line:loc.line, col:loc.col});
    return _pm.apply(this, arguments);
  };
}catch(e){}
})();`
	return chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(js).Do(ctx)
		return err
	})
}

// جمع‌آوری یافته‌های runtime و تبدیل به SinkDoc
func CollectRuntimeSinks(ctx context.Context, pageURL, siteID string) ([]models.SinkDoc, error) {
	type raw struct {
		Kind string `json:"kind"`
		URL  string `json:"url"`
		Line int    `json:"line"`
		Col  int    `json:"col"`
	}
	var arr []raw
	if err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`window.__SC_RUNTIMESINKS || []`, &arr)); err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]models.SinkDoc, 0, len(arr))
	for _, r := range arr {
		u, t := normalizeSourceURL(pageURL, r.URL) // همون نرمال‌ساز مرحله ۳
		out = append(out, models.SinkDoc{
			SiteID:     siteID,
			PageURL:    pageURL,
			SourceType: t,
			SourceURL:  u,
			Kind:       r.Kind,
			Line:       r.Line,
			Col:        r.Col,
			Snippet:    "",
			DetectedAt: now,
		})
	}
	return out, nil
}
