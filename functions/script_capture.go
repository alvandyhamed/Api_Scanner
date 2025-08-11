// functions/script_capture.go
package functions

import (
	"context"
	"sync"

	"github.com/chromedp/chromedp"

	"github.com/chromedp/cdproto/debugger"
	rt "github.com/chromedp/cdproto/runtime"
)

// CollectScripts همهٔ اسکریپت‌هایی که تب واقعاً اجرا می‌کند را جمع می‌کند
// map[key] = source   (key = URL یا inline:<id>)
func CollectScripts(ctx context.Context) (map[string]string, error) {
	var mu sync.Mutex
	scripts := make(map[string]string)

	// رویداد ScriptParsed را گوش بده
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if e, ok := ev.(*debugger.EventScriptParsed); ok {
			sid := e.ScriptID // در نسخه‌های جدید از نوع runtime.ScriptID است
			u := e.URL
			if u == "" {
				u = "inline:" + string(sid)
			}

			// سورس را بگیر (امضای جدید 3 خروجی می‌دهد)
			go func(uid string, id rt.ScriptID) {
				src, _, err := debugger.GetScriptSource(id).Do(ctx)
				if err != nil || src == "" {
					return
				}
				// سقف 2MB برای هر فایل
				if len(src) > 2<<20 {
					src = src[:2<<20]
				}
				mu.Lock()
				scripts[uid] = src
				mu.Unlock()
			}(u, sid)
		}
	})

	// Enable به‌صورت ActionFunc چون Do الان دو خروجی می‌دهد
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
		_, err := debugger.Enable().Do(c)
		return err
	})); err != nil {
		return nil, err
	}

	return scripts, nil
}
