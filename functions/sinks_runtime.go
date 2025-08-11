package functions

import (
	"SiteChecker/models"
	"context"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type sinkEntry struct {
	Kind   string      `json:"kind"`
	Info   interface{} `json:"info"`
	WhenMs int64       `json:"when"`
	Src    string      `json:"src"`
}

func ScanSinksRuntime(ctx context.Context, pageURL string) ([]models.SinkDoc, error) {

	bctx, cancel := newBrowserCtx(ctx)
	defer cancel()

	instrument := `
(function(){
  try {
    if (window.__sinkLog) return;
    window.__sinkLog = [];

    function log(kind, info) {
      try {
        window.__sinkLog.push({
          kind: String(kind),
          info: info === undefined ? null : info,
          when: Date.now(),
          src: (location && location.href) ? location.href : ""
        });
      } catch (e) {}
    }

    // eval
    try {
      const _eval = window.eval;
      window.eval = function(s){ log("eval",{snippet: String(s).slice(0,200)}); return _eval.apply(this, arguments); };
    } catch(e){}

    // new Function
    try {
      const _F = window.Function;
      window.Function = function(){ try{ log("newFunction",{args: Array.prototype.map.call(arguments, x=>String(x)).join(",").slice(0,200)});}catch(e){}; return _F.apply(this, arguments); };
      window.Function.prototype = _F.prototype;
    } catch(e){}

    // setTimeout / setInterval با رشته
    try {
      const _st = window.setTimeout;
      window.setTimeout = function(cb, t){ if (typeof cb === "string") log("setTimeoutStr",{snippet: cb.slice(0,200)}); return _st.apply(this, arguments); };
      const _si = window.setInterval;
      window.setInterval = function(cb, t){ if (typeof cb === "string") log("setIntervalStr",{snippet: cb.slice(0,200)}); return _si.apply(this, arguments); };
    } catch(e){}

    // document.write
    try {
      const _dw = document.write;
      document.write = function(){ try{ log("documentWrite",{argsLen: arguments.length}); }catch(e){}; return _dw.apply(this, arguments); };
    } catch(e){}

    // innerHTML setter
    try {
      const desc = Object.getOwnPropertyDescriptor(Element.prototype, "innerHTML");
      if (desc && desc.set) {
        Object.defineProperty(Element.prototype, "innerHTML", {
          configurable: true,
          get: desc.get,
          set: function(v){ try{ log("innerHTML",{snippet: String(v).slice(0,200)});}catch(e){}; return desc.set.call(this, v); }
        });
      }
    } catch(e){}

    // fetch
    try {
      const _fetch = window.fetch;
      window.fetch = function(input, init){ try{ log("fetch",{to: (typeof input==="string"?input:(input&&input.url)||"").slice(0,300)});}catch(e){}; return _fetch.apply(this, arguments); };
    } catch(e){}

    // XHR (sync/open)
    try {
      const _open = XMLHttpRequest.prototype.open;
      XMLHttpRequest.prototype.open = function(method, url, async){
        try{
          log("XMLHttpRequest",{to: String(url).slice(0,300), async: (async!==false)});
          if (async === false) log("syncXHR",{to:String(url).slice(0,300)});
        }catch(e){}
        return _open.apply(this, arguments);
      };
    } catch(e){}

    // postMessage (send)
    try {
      const _pm = window.postMessage;
      window.postMessage = function(message, targetOrigin, transfer){
        try{ log("postMessageSend",{targetOrigin: String(targetOrigin||"")}); }catch(e){}
        return _pm.apply(this, arguments);
      };
    } catch(e){}

    // postMessage (receive)
    try {
      window.addEventListener("message", function(ev){
        try{ log("postMessageRecv",{origin: String(ev.origin||"")}); }catch(e){}
      }, true);
      if ("onmessage" in window) {
        const d = Object.getOwnPropertyDescriptor(window, "onmessage");
        if (!d || d.configurable) {
          let _h = null;
          Object.defineProperty(window, "onmessage", {
            configurable: true,
            get(){ return _h; },
            set(fn){ _h = fn; try{ log("postMessageRecv","onmessage-set"); }catch(e){} }
          });
        }
      }
    } catch(e){}

    // Inline event handlers (on*)
    try {
      const _setAttr = Element.prototype.setAttribute;
      Element.prototype.setAttribute = function(name, value){
        if (name && /^on[a-z]+$/i.test(name)) {
          try{ log("inlineEventHandler",{name:String(name), snippet: String(value).slice(0,200)}); }catch(e){}
        }
        return _setAttr.apply(this, arguments);
      };
    } catch(e){}

    // direct DOM API
    try {
      const api = ["getElementById","getElementsByClassName","querySelector","querySelectorAll"];
      for (const k of api) {
        const _fn = Document.prototype[k];
        if (typeof _fn === "function") {
          Document.prototype[k] = function(){ try{ log("directDOM",k);}catch(e){}; return _fn.apply(this, arguments); };
        }
      }
    } catch(e){}
  } catch(e){}
})();`

	if err := chromedp.Run(bctx, chromedp.ActionFunc(func(c context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(instrument).Do(c)
		return err
	})); err != nil {
		return nil, err
	}

	var _ string
	if err := chromedp.Run(bctx,
		chromedp.Navigate(pageURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(6*time.Second),
		//chromedp.Title(&_), // یه اکشن سبک برای sync
	); err != nil {
		return nil, err
	}

	// لاگ‌ها را از صفحه بخوان
	var entries []sinkEntry
	if err := chromedp.Run(bctx, chromedp.EvaluateAsDevTools(`window.__sinkLog || []`, &entries)); err != nil {
		return nil, err
	}

	// تبدیل به SinkDoc
	out := make([]models.SinkDoc, 0, len(entries))
	now := time.Now()
	for _, e := range entries {
		out = append(out, models.SinkDoc{
			SiteID:     "",        // در هندلر پر می‌کنی
			PageURL:    pageURL,   // نرمالایزش در هندلر
			SourceType: "runtime", // تفکیک از html/script
			SourceURL:  e.Src,
			Kind:       e.Kind,
			Line:       0,
			Col:        0,
			Snippet:    "", // اگر خواستی می‌تونی از e.Info برداری
			DetectedAt: now,
		})
	}
	return out, nil
}
