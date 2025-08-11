package handlers

import (
	"SiteChecker/functions"
	"SiteChecker/models"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func ScanHandler(w http.ResponseWriter, r *http.Request) {

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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)

}
