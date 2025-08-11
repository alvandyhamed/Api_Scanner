package handlers

import (
	"SiteChecker/models"
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func PagesListHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	filter := bson.M{"site_id": siteID}

	if host := strings.TrimSpace(r.URL.Query().Get("host")); host != "" {
		filter["host"] = host
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		filter["$or"] = bson.A{
			bson.M{"url": rxContains(q)},
			bson.M{"url_norm": rxContains(q)},
			bson.M{"path": rxContains(q)},
		}
	}
	if from, ok := qTime(r, "from"); ok {
		filter["scanned_at"] = bson.M{"$gte": from}
	}
	if to, ok := qTime(r, "to"); ok {
		if m, ok := filter["scanned_at"].(bson.M); ok {
			m["$lte"] = to
		} else {
			filter["scanned_at"] = bson.M{"$lte": to}
		}
	}

	opts := mopts.Find().
		SetSort(qSort(r, "scanned_at", -1)).
		SetLimit(qLimit(r)).
		SetSkip(qSkip(r))

	// فقط فیلدهای لازم برگردون
	proj := bson.M{
		"url":             1,
		"url_norm":        1,
		"site_id":         1,
		"host":            1,
		"path":            1,
		"scanned_at":      1,
		"groups":          1,
		"resource_groups": 1,
		"externals":       1,
	}
	opts.SetProjection(proj)
	cur, err := models.PagesColl().Find(ctx, filter, opts)
	if err != nil {
		srvError(w, err)
		return
	}

	defer cur.Close(ctx)

	var items []bson.M
	if err := cur.All(ctx, &items); err != nil {
		srvError(w, err)
		return
	}
	total, _ := models.PagesColl().CountDocuments(ctx, filter)

	writeJSON(w, http.StatusOK, bson.M{
		"items": items, "total": total, "limit": qLimit(r), "skip": qSkip(r),
	})
}

func PageByURLHandler(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("url"))
	if raw == "" {
		badRequest(w, "url is required")
		return
	}
	// نرمال‌سازی ساده مثل SaveScanResults
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		badRequest(w, "invalid url")
		return
	}
	urlNorm := u.Scheme + "://" + u.Host + u.EscapedPath()
	if u.RawQuery != "" {
		urlNorm += "?" + u.RawQuery
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var out bson.M
	err = models.PagesColl().FindOne(ctx, bson.M{"url_norm": urlNorm}).Decode(&out)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, http.StatusOK, bson.M{"item": nil})
		return
	}
	if err != nil {
		srvError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bson.M{"item": out})

}
