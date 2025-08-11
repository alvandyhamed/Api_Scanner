package functions

import (
	"context"
	"github.com/chromedp/chromedp"
)

func newBrowserCtx(parent context.Context) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],

		chromedp.ExecPath("/usr/bin/chromium"),

		chromedp.Flag("headless", "new"),

		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("window-size", "1366,768"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	return ctx, func() {
		cancelCtx()
		cancelAlloc()
	}
}
