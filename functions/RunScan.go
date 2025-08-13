package functions

import (
	"SiteChecker/models"
	"context"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func RunScan(req models.ScanRequest) (*models.ScanResponse, error) {
	browserCtx, cancelBrowser := newBrowserCtx(context.Background())
	defer cancelBrowser()

	if req.WaitSec <= 0 {
		req.WaitSec = 6
	}
	if req.JSFetchTimeout <= 0 {
		req.JSFetchTimeout = 8
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, time.Duration(45+req.WaitSec)*time.Second)
	defer cancelTimeout()

	var (
		resourcesJS []string
		pageHTML    string
		scriptSrcs  []string
	)

	scriptsMap, err := CollectScripts(timeoutCtx)
	if err != nil {
		return nil, err
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

	err = chromedp.Run(timeoutCtx,
		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Accept-Language":           "en-US,en;q=0.9",
			"Upgrade-Insecure-Requests": "1",
		}),

		chromedp.ActionFunc(func(c context.Context) error {
			return emulation.SetUserAgentOverride(ua).
				WithPlatform("Windows").
				WithUserAgentMetadata(&emulation.UserAgentMetadata{
					Platform:        "Windows",
					PlatformVersion: "10.0",
					Architecture:    "x86",
					Model:           "",
					Mobile:          false,
				}).Do(c)
		}),

		chromedp.Evaluate(`Object.defineProperty(navigator,'webdriver',{get:()=>undefined})`, nil),

		InstallSourceURLHooks(),
		InstallRuntimePostMessageHook(),
		chromedp.Navigate(req.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(time.Duration(req.WaitSec)*time.Second),

		chromedp.EvaluateAsDevTools(`performance.getEntriesByType('resource').map(r => r.name)`, &resourcesJS),
		chromedp.OuterHTML("html", &pageHTML, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(`Array.from(document.querySelectorAll('script[src]')).map(s => new URL(s.src, location.href).href)`, &scriptSrcs),
	)
	if err != nil {
		return nil, err
	}

	paths := extractPathsFromHTML(pageHTML)

	for _, code := range scriptsMap {
		if code != "" {
			paths = append(paths, extractPathsFromHTML(code)...)
		}
	}

	extraPaths, errorsList := fetchAndExtractFromScripts(timeoutCtx, scriptSrcs, req.JSFetchTimeout)
	paths = append(paths, extraPaths...)

	// Dedup
	paths = uniqueStrings(paths)
	resourcesJS = uniqueStrings(resourcesJS)
	scriptSrcs = uniqueStrings(scriptSrcs)

	return &models.ScanResponse{
		URL:         req.URL,
		Resources:   resourcesJS,
		UniquePaths: paths,
		AllScripts:  scriptSrcs,
		Errors:      errorsList,
	}, nil
}
