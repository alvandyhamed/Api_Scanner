package functions

import (
	"SiteChecker/models"
	"context"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type sinkEntry struct {
	Kind    string `json:"kind"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Func    string `json:"func"`
	Snippet string `json:"snippet"`
	Extra   any    `json:"extra"`
	WhenMs  int64  `json:"when"`
}

// ScanSinksRuntime: با اینسترومنتیشن قبل از لود، سینک‌ها را با فایل/خط/ستون ثبت می‌کند
func ScanSinksRuntime(ctx context.Context, pageURL string) ([]models.SinkDoc, error) {
	bctx, cancel := newBrowserCtx(ctx)
	defer cancel()

	// اسکریپتِ اینسترومنتیشن
	instrument := `
(function(){
  try {
    if (window.__sinkLog) return;
    window.__sinkLog = [];

    function push(e){ try{ window.__sinkLog.push(e); }catch(_){} }

    // استخراج callsite از stack
    function callsite(skip){
      try{
        const err = new Error();
        const st = (err.stack||"").split("\n").slice(1+(skip||0));
        for (const ln of st){
          // شکل‌های رایج: "at func (https://.../app.js:123:45)" یا "at https://.../app.js:123:45"
          const m = ln.match(/at\s+(?:(.*?)\s+\()?(.*?):(\d+):(\d+)\)?/);
          if (m){
            const func = (m[1]||"").trim();
            const file = (m[2]||"").trim();
            const line = parseInt(m[3]||"0",10);
            const col  = parseInt(m[4]||"0",10);
            // فریم‌های خودمان را رد کن
            if (file.includes("extensions::") || file.startsWith("chrome-extension:")) continue;
            return {func, file, line, col};
          }
        }
      }catch(e){}
      return {func:"", file:(location&&location.href)||"", line:0, col:0};
    }
    function takeSnippet(x){
      try{
        const s = String(x==null?"":x);
        return s.length>200 ? s.slice(0,200) : s;
      }catch(_){ return ""; }
    }
    function log(kind, info, skip){
      const cs = callsite((skip||0)+1);
      push({
        kind: kind,
        file: cs.file,
        line: cs.line,
        col:  cs.col,
        func: cs.func,
        snippet: info && info.snippet ? takeSnippet(info.snippet) : (info==null?"":takeSnippet(info)),
        extra: info && info.extra!==undefined ? info.extra : info,
        when: Date.now()
      });
    }

    // --- سینک‌ها ---

    // eval
    try{
      const _eval = window.eval;
      window.eval = function(s){ log("eval",{snippet:s},1); return _eval.apply(this, arguments); };
    }catch(_){}

    // new Function
    try{
      const _F = window.Function;
      window.Function = function(){ log("newFunction",{snippet:Array.prototype.join.call(arguments,",")},1); return _F.apply(this, arguments); };
      window.Function.prototype = _F.prototype;
    }catch(_){}

    // setTimeout / setInterval (رشته)
    try{
      const _st = window.setTimeout;
      window.setTimeout = function(cb, t){ if (typeof cb==="string") log("setTimeoutStr",{snippet:cb},1); return _st.apply(this, arguments); };
      const _si = window.setInterval;
      window.setInterval = function(cb, t){ if (typeof cb==="string") log("setIntervalStr",{snippet:cb},1); return _si.apply(this, arguments); };
    }catch(_){}

    // document.write
    try{
      const _dw = document.write;
      document.write = function(){ log("documentWrite", {extra:{argsLen:arguments.length}},1); return _dw.apply(this, arguments); };
    }catch(_){}

    // innerHTML setter
    try{
      const desc = Object.getOwnPropertyDescriptor(Element.prototype, "innerHTML");
      if (desc && desc.set){
        Object.defineProperty(Element.prototype, "innerHTML", {
          configurable:true,
          get: desc.get,
          set: function(v){ log("innerHTML", {snippet:v},1); return desc.set.call(this, v); }
        });
      }
    }catch(_){}

    // fetch
    try{
      const _fetch = window.fetch;
      window.fetch = function(input, init){
        const to = typeof input==="string" ? input : (input && input.url) || "";
        log("fetch", {extra:{to: String(to)}}, 1);
        return _fetch.apply(this, arguments);
      };
    }catch(_){}

    // XHR
    try{
      const _open = XMLHttpRequest.prototype.open;
      XMLHttpRequest.prototype.open = function(method, url, async){
        const a = (async!==false);
        log("XMLHttpRequest", {extra:{to:String(url), async:a}}, 1);
        if (!a) log("syncXHR", {extra:{to:String(url)}}, 1);
        return _open.apply(this, arguments);
      };
    }catch(_){}

    // Inline handlers
    try{
      const _setAttr = Element.prototype.setAttribute;
      Element.prototype.setAttribute = function(name, value){
        if (name && /^on[a-z]+$/i.test(name)){
          log("inlineEventHandler",{snippet:value},1);
        }
        return _setAttr.apply(this, arguments);
      };
    }catch(_){}

    // direct DOM
    try{
      const api = ["getElementById","getElementsByClassName","querySelector","querySelectorAll"];
      for (const k of api){
        const _fn = Document.prototype[k];
        if (typeof _fn === "function"){
          Document.prototype[k] = function(){ log("directDOM",{extra:{api:k}},1); return _fn.apply(this, arguments); };
        }
      }
    }catch(_){}

    // --- postMessage: send + listen + recv ---

    // send
    try{
      const _pm = window.postMessage;
      window.postMessage = function(message, targetOrigin, transfer){
        log("postMessageSend",{extra:{targetOrigin:String(targetOrigin||"")}},1);
        return _pm.apply(this, arguments);
      };
    }catch(_){}

    // listen: addEventListener("message", handler)
    try{
      const regMap = new WeakMap();
      const _add = window.addEventListener;
      window.addEventListener = function(type, handler, options){
        if (String(type).toLowerCase()==="message" && typeof handler==="function"){
          const cs = (function(){ const c=callsite(1); return c; })();
          regMap.set(handler, cs);
          // wrap برای لاگ گرفتن هنگام دریافت
          const wrapped = function(ev){
            const meta = regMap.get(handler) || {};
            push({
              kind: "postMessageRecv",
              file: meta.file || "",
              line: meta.line || 0,
              col:  meta.col  || 0,
              func: (handler.name||meta.func||""),
              snippet: "",
              extra: {origin: String(ev.origin||"")},
              when: Date.now()
            });
            return handler.apply(this, arguments);
          };
          return _add.call(this, type, wrapped, options);
        }
        return _add.apply(this, arguments);
      };
    }catch(_){}

    // listen: window.onmessage = fn
    try{
      let _h = null;
      const cs = {};
      Object.defineProperty(window, "onmessage", {
        configurable:true,
        get(){ return _h; },
        set(fn){
          _h = fn;
          const c = callsite(1);
          cs.file = c.file; cs.line=c.line; cs.col=c.col; cs.func=c.func;
          if (typeof fn==="function"){
            const wrap = function(ev){
              push({
                kind:"postMessageRecv",
                file: cs.file||"",
                line: cs.line||0,
                col:  cs.col||0,
                func: (fn.name||cs.func||""),
                snippet:"",
                extra:{origin:String(ev.origin||"")},
                when: Date.now()
              });
              return fn.apply(this, arguments);
            };
            return Object.defineProperty(this, "onmessage", {value:wrap, writable:true, configurable:true});
          }
        }
      });
      // ثبت محل لیسنر
      (function(){
        const c = callsite(1);
        push({ kind:"postMessageListen", file:c.file, line:c.line, col:c.col, func:c.func, snippet:"", extra:null, when:Date.now() });
      })();
    }catch(_){}
  } catch(_){}
})();`

	// تزریق قبل از هر document
	if err := chromedp.Run(bctx, chromedp.ActionFunc(func(c context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(instrument).Do(c)
		return err
	})); err != nil {
		return nil, err
	}

	// ناوبری و کمی صبر
	if err := chromedp.Run(bctx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(8*time.Second),
	); err != nil {
		return nil, err
	}

	// خواندن لاگ‌ها
	var entries []sinkEntry
	if err := chromedp.Run(bctx, chromedp.EvaluateAsDevTools(`window.__sinkLog || []`, &entries)); err != nil {
		return nil, err
	}

	out := make([]models.SinkDoc, 0, len(entries))
	now := time.Now()
	for _, e := range entries {
		out = append(out, models.SinkDoc{
			SiteID:     "",
			PageURL:    pageURL,
			SourceType: "runtime",
			SourceURL:  e.File,
			Kind:       e.Kind,
			Line:       e.Line,
			Col:        e.Col,
			Func:       e.Func,
			Snippet:    e.Snippet,
			DetectedAt: now,
		})
	}
	return out, nil
}
