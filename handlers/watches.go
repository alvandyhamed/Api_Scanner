// handlers/watches.go
package handlers

import (
	"SiteChecker/functions"
	"SiteChecker/models"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func deriveSiteAndNorm(rawURL string) (siteID, urlNorm string) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	u, _ := url.Parse(rawURL)
	if u == nil {
		return "", ""
	}
	host := strings.ToLower(u.Hostname())
	base := host // اگر publicsuffix داری می‌تونی اینجا دقیق‌ترش کنی

	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	urlNorm = u.Scheme + "://" + u.Host + path
	if u.RawQuery != "" {
		urlNorm += "?" + u.RawQuery
	}
	return base, urlNorm
}

// GET /api/watches?site_id=&url_norm=
func WatchesListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		badRequest(w, "GET only")
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	q := bson.M{}
	if site := r.URL.Query().Get("site_id"); site != "" {
		q["site_id"] = site
	}
	if un := r.URL.Query().Get("url_norm"); un != "" {
		q["url_norm"] = un
	}
	cur, err := models.WatchesColl().Find(r.Context(), q, options.Find().SetSort(bson.D{{Key: "next_run_at", Value: 1}}))
	if err != nil {
		srvError(w, err)
		return
	}
	var items []models.WatchDoc
	_ = cur.All(r.Context(), &items)
	writeJSON(w, http.StatusOK, bson.M{"items": items})
}

type watchCreateReq struct {
	URL     string `json:"url"`
	SiteID  string `json:"site_id"`
	FreqMin int    `json:"freq_min"`
	Enabled bool   `json:"enabled"`
}

// POST /api/watches/create
func WatchCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		badRequest(w, "POST only")
		return
	}

	var req watchCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		badRequest(w, "url is required")
		return
	}
	if req.FreqMin <= 0 {
		req.FreqMin = 1440 // پیش‌فرض: روزانه
	}

	siteID, urlNorm := deriveSiteAndNorm(req.URL)
	if req.SiteID != "" {
		siteID = req.SiteID
	}
	now := time.Now()
	next := now.Add(time.Duration(req.FreqMin) * time.Minute)

	filter := bson.M{"site_id": siteID, "url_norm": urlNorm}

	update := bson.M{
		"$set": bson.M{
			"site_id":     siteID, // در هر حالتی ست کنیم تا همواره درست بماند
			"url":         req.URL,
			"url_norm":    urlNorm,
			"enabled":     req.Enabled,
			"freq_min":    req.FreqMin,
			"next_run_at": next,
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"created_at": now, // فقط فیلدهای مخصوص insert
		},
	}

	_, err := models.WatchesColl().UpdateOne(r.Context(), filter, update, options.Update().SetUpsert(true))
	if err != nil {
		srvError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, bson.M{
		"ok":       true,
		"site_id":  siteID,
		"url_norm": urlNorm,
	})
}

type watchKeyReq struct {
	URL     string `json:"url"`      // اختیاری
	URLNorm string `json:"url_norm"` // ترجیحاً این
	SiteID  string `json:"site_id"`  // اگر URLNorm بدون scheme/host دقیق بود
}

// POST /api/watches/scan-now  { url_norm | url }
func WatchScanNowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		badRequest(w, "POST only")
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req watchKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}
	siteID, urlNorm := req.SiteID, req.URLNorm
	if urlNorm == "" && req.URL != "" {
		siteID, urlNorm = deriveSiteAndNorm(req.URL)
	}
	if urlNorm == "" {
		badRequest(w, "url or url_norm required")
		return
	}
	if siteID == "" {
		// از خود urlNorm استخراج کن
		u, _ := url.Parse(urlNorm)
		if u != nil {
			siteID = strings.ToLower(u.Hostname())
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// پیدا کردن Watch
	var wdoc models.WatchDoc
	err := models.WatchesColl().FindOne(ctx, bson.M{"site_id": siteID, "url_norm": urlNorm}).Decode(&wdoc)
	if err == mongo.ErrNoDocuments {
		badRequest(w, "watch not found")
		return
	}
	if err != nil {
		srvError(w, err)
		return
	}

	// اسکن فوری
	resp, err := functions.RunScan(models.ScanRequest{
		URL:            wdoc.URL,
		WaitSec:        7,
		JSFetchTimeout: 8,
	})
	if err != nil {
		srvError(w, err)
		return
	}
	if resp != nil {
		_ = functions.SaveScanResults(ctx, wdoc.URL, resp.Resources, resp.UniquePaths, resp.AllScripts)
	}

	now := time.Now()
	next := now.Add(time.Duration(maxInt(5, wdoc.FreqMin)) * time.Minute)
	_, _ = models.WatchesColl().UpdateOne(ctx,
		bson.M{"site_id": siteID, "url_norm": urlNorm},
		bson.M{"$set": bson.M{
			"last_run_at": now,
			"next_run_at": next,
			"updated_at":  time.Now(),
		}},
	)

	writeJSON(w, http.StatusOK, bson.M{"ok": true, "site_id": siteID, "url_norm": urlNorm})
}

// POST /api/watches/delete  { url_norm | url }
func WatchDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		badRequest(w, "POST/DELETE only")
		http.Error(w, "POST/DELETE only", http.StatusMethodNotAllowed)
		return
	}
	var req watchKeyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}
	siteID, urlNorm := req.SiteID, req.URLNorm
	if urlNorm == "" && req.URL != "" {
		siteID, urlNorm = deriveSiteAndNorm(req.URL)
	}
	if urlNorm == "" {
		badRequest(w, "url or url_norm required")
		return
	}
	if siteID == "" {
		u, _ := url.Parse(urlNorm)
		if u != nil {
			siteID = strings.ToLower(u.Hostname())
		}
	}

	res, err := models.WatchesColl().DeleteOne(r.Context(), bson.M{"site_id": siteID, "url_norm": urlNorm})
	if err != nil {
		srvError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bson.M{"ok": true, "deleted": res.DeletedCount, "site_id": siteID, "url_norm": urlNorm})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
