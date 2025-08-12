package functions

import (
	"SiteChecker/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func StartWatchScheduler(ctx context.Context) {
	go func() {
		t := time.NewTicker(1 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				_ = runDueWatches(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func runDueWatches(ctx context.Context) error {
	now := time.Now()
	cur, err := models.WatchesColl().Find(ctx,
		bson.M{"enabled": true, "next_run_at": bson.M{"$lte": now}},
		options.Find().SetLimit(50),
	)
	if err != nil {
		return err
	}

	for cur.Next(ctx) {
		var w models.WatchDoc
		if err := cur.Decode(&w); err != nil {
			continue
		}

		// 1) اسکن
		req := models.ScanRequest{URL: w.URL, WaitSec: 7, JSFetchTimeout: 8}
		resp, err := RunScan(req)
		if err != nil {
			log.Printf("[watch] scan error url=%s err=%v", w.URL, err)
		}

		if resp != nil {
			// ذخیره نتایج صفحه/اندپوینت‌ها
			_ = SaveScanResults(ctx, w.URL, resp.Resources, resp.UniquePaths, resp.AllScripts)
		}

		// 2) محاسبه تغییرات
		changed, summary := computeChangeSummary(ctx, w.SiteID, w.URLNorm, w.LastSummary)

		// 3) آپدیت زمان‌بندی
		upd := bson.M{
			"$set": bson.M{
				"last_run_at":  now,
				"next_run_at":  now.Add(time.Duration(max(5, w.FreqMin)) * time.Minute),
				"last_summary": summary,
				"updated_at":   time.Now(),
			},
		}
		if changed {
			upd["$set"].(bson.M)["last_change_at"] = time.Now()
			// 4) Notify Discord
			_ = notifyDiscord(ctx, w.SiteID, w.URL, summary)
		}
		_, _ = models.WatchesColl().UpdateOne(ctx, bson.M{"site_id": w.SiteID, "url_norm": w.URLNorm}, upd)
	}
	return nil
}

func computeChangeSummary(ctx context.Context, siteID, urlNorm string, prev models.WatchSummary) (bool, models.WatchSummary) {
	epCount, epLast := endpointsStatsForPage(ctx, siteID, urlNorm)
	skCount, skLast := sinksStatsForPage(ctx, siteID, urlNorm)
	sum := models.WatchSummary{Endpoints: epCount, Sinks: skCount, LastEP: epLast, LastSink: skLast}
	// دلخواه: Digest
	h := sha256.New()
	h.Write([]byte(siteID))
	h.Write([]byte(urlNorm))
	h.Write([]byte(epLast.Format(time.RFC3339)))
	h.Write([]byte(skLast.Format(time.RFC3339)))
	h.Write([]byte{byte(epCount), byte(skCount)})
	sum.Digest = hex.EncodeToString(h.Sum(nil))
	changed := (sum.Endpoints != prev.Endpoints) || (sum.Sinks != prev.Sinks) || epLast.After(prev.LastEP) || skLast.After(prev.LastSink)
	return changed, sum
}

func endpointsStatsForPage(ctx context.Context, siteID, urlNorm string) (int, time.Time) {
	// اندپوینت‌هایی که source_urls شامل این صفحه است
	epCount, _ := models.EndpointsColl().CountDocuments(ctx, bson.M{"site_id": siteID, "source_urls": urlNorm})
	var last struct {
		LastSeen time.Time `bson:"last"`
	}
	_ = models.EndpointsColl().FindOne(ctx, bson.M{"site_id": siteID, "source_urls": urlNorm},
		options.FindOne().SetSort(bson.D{{"last_seen", -1}}).SetProjection(bson.M{"last": "$last_seen"})).Decode(&last)
	return int(epCount), last.LastSeen
}

func sinksStatsForPage(ctx context.Context, siteID, urlNorm string) (int, time.Time) {
	skCount, _ := models.SinksColl().CountDocuments(ctx, bson.M{"site_id": siteID, "page_url": urlNorm})
	var last struct {
		Last time.Time `bson:"last"`
	}
	_ = models.SinksColl().FindOne(ctx, bson.M{"site_id": siteID, "page_url": urlNorm},
		options.FindOne().SetSort(bson.D{{"last_detected_at", -1}}).SetProjection(bson.M{"last": "$last_detected_at"})).Decode(&last)
	return int(skCount), last.Last
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// notifyDiscord: اگر دیسکورد فعال و وبهوک ست باشد، پیام تغییرات را ارسال می‌کند.
func notifyDiscord(ctx context.Context, siteID, pageURL string, sum models.WatchSummary) error {
	// خواندن تنظیمات
	doc, err := models.GetDiscordSettings(ctx)
	if err != nil {
		return err
	}
	if !doc.Enabled || strings.TrimSpace(doc.WebhookURL) == "" {
		// پیکربندی نشده یا غیرفعال → نوتیفای نکن
		return nil
	}

	// ساخت پیام خلاصه
	msg := fmt.Sprintf(
		"🔔 *SiteChecker*\nSite: `%s`\nPage: %s\nEndpoints: %d (last: %s)\nSinks: %d (last: %s)\nTime: %s",
		siteID,
		pageURL,
		sum.Endpoints, sum.LastEP.Format(time.RFC3339),
		sum.Sinks, sum.LastSink.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
	)

	// ارسال با وبهوک (همین پکیج: functions.SendDiscordWebhook)
	return SendDiscordWebhook(ctx, doc.WebhookURL, msg)
}
