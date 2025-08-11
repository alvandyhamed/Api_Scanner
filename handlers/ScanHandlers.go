// handlers/ScanHandlers.go
package handlers

import (
	"SiteChecker/functions"
	"SiteChecker/models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

func ScanHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req models.ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		req.URL = "https://" + req.URL
	}
	if req.WaitSec <= 0 {
		req.WaitSec = 6
	}
	if req.JSFetchTimeout <= 0 {
		req.JSFetchTimeout = 8
	}

	start := time.Now()
	resp, err := functions.RunScan(req)
	if err != nil {
		http.Error(w, "scan error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp.ProcessedAt = time.Now().Format(time.RFC3339)
	resp.PageDuration = time.Since(start).String()

	// ذخیره نتایج صفحه/اندپوینت‌ها
	saveCtx, cancelSave := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancelSave()
	if err := functions.SaveScanResults(saveCtx, req.URL, resp.Resources, resp.UniquePaths, resp.AllScripts); err != nil {
		log.Printf("[mongo save] url=%s err=%v", req.URL, err)
	}

	sinksCtx, cancelSinks := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancelSinks()
	sinks, err := functions.ScanSinksRuntime(sinksCtx, req.URL) // ← نسخه‌ی runtime که نوشتیم
	if err == nil && len(sinks) > 0 {
		// siteID و urlNorm
		u, _ := url.Parse(req.URL)
		host := strings.ToLower(u.Hostname())
		base, _ := publicsuffix.EffectiveTLDPlusOne(host)
		if base == "" {
			base = host
		}
		urlNorm := u.Scheme + "://" + u.Host + u.EscapedPath()
		if u.RawQuery != "" {
			urlNorm += "?" + u.RawQuery
		}

		for i := range sinks {
			if sinks[i].SiteID == "" {
				sinks[i].SiteID = base
			}
			if sinks[i].PageURL == "" {
				sinks[i].PageURL = urlNorm
			}
		}

		bwRes, bwErr := functions.PersistSinks(sinksCtx, sinks)
		if bwErr != nil {
			log.Printf("[sinks] persist error: %v", bwErr)
		} else {
			log.Printf("[sinks] bulk write matched=%d modified=%d upserted=%d",
				bwRes.MatchedCount, bwRes.ModifiedCount, bwRes.UpsertedCount)
		}
	} else if err != nil {
		log.Printf("[sinks] scan error: %v", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
