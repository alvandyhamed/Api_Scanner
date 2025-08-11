package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

func WithCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// dev-friendly: اگر UI جداست برداری
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func srvError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func qLimit(r *http.Request) int64 {
	lim, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if lim <= 0 {
		return defaultLimit
	}
	if lim > maxLimit {
		return maxLimit
	}
	return int64(lim)
}

func qSkip(r *http.Request) int64 {
	s, _ := strconv.Atoi(r.URL.Query().Get("skip"))
	if s < 0 {
		return 0
	}
	return int64(s)
}

func qSort(r *http.Request, defField string, defOrder int) bson.D {
	field := r.URL.Query().Get("sort")
	if field == "" {
		field = defField
	}
	orderStr := strings.ToLower(r.URL.Query().Get("order"))
	order := defOrder
	if orderStr == "asc" {
		order = 1
	} else if orderStr == "desc" {
		order = -1
	}
	return bson.D{{Key: field, Value: order}}
}

func qTime(r *http.Request, key string) (time.Time, bool) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func rxContains(s string) bson.M {
	if s == "" {
		return nil
	}
	return bson.M{"$regex": s, "$options": "i"}
}
