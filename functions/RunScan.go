package functions

import (
	"SiteChecker/models"
	"context"
	"github.com/chromedp/chromedp"
	"time"
)

func RunScan(req models.ScanRequest) (*models.ScanResponse, error) {
	browserCtx, cancelBrowser := newBrowserCtx(context.Background())
	defer cancelBrowser()

	if req.WaitSec <= 0 {
		req.WaitSec = 6
	}
	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, time.Duration(45+req.WaitSec)*time.Second)
	defer cancelTimeout()

	var (
		resourcesJS []string
		pageHTML    string
		scriptSrcs  []string
	)

	err := chromedp.Run(timeoutCtx,
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
	extraPaths, errorsList := fetchAndExtractFromScripts(timeoutCtx, scriptSrcs, req.JSFetchTimeout)
	paths = append(paths, extraPaths...)

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
