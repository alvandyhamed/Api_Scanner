package handlers

import (
	"SiteChecker/functions"
	"SiteChecker/models"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func ScanHandler(w http.ResponseWriter, r *http.Request) {
	// اگر panic شد 500 بده، نه connection reset
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

	// ذخیره در Mongo با تایم‌اوت
	saveCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := functions.SaveScanResults(saveCtx, req.URL, resp.Resources, resp.UniquePaths, resp.AllScripts); err != nil {
		log.Printf("[mongo save] url=%s err=%v", req.URL, err)
	} else {
		log.Printf("[mongo save OK] url=%s res=%d paths=%d scripts=%d",
			req.URL, len(resp.Resources), len(resp.UniquePaths), len(resp.AllScripts))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}
