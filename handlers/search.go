package handlers

import (
	"SiteChecker/models"
	"context"
	"go.mongodb.org/mongo-driver/bson"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"strings"
	"time"
)

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {

		badRequest(w, "site_id is required")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		badRequest(w, "q is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	lim := qLimit(r)

	// pages
	pagesCur, _ := models.PagesColl().Find(ctx, bson.M{
		"site_id": siteID,
		"$or": bson.A{
			bson.M{"url": rxContains(q)},
			bson.M{"url_norm": rxContains(q)},
		},
	}, mopts.Find().SetLimit(lim).SetProjection(bson.M{"url_norm": 1, "scanned_at": 1}))
	var pages []bson.M
	if pagesCur != nil {
		_ = pagesCur.All(ctx, &pages)
	}

	// endpoints
	epCur, _ := models.EndpointsColl().Find(ctx, bson.M{
		"site_id":  siteID,
		"endpoint": rxContains(q),
	}, mopts.Find().SetLimit(lim).SetProjection(bson.M{"endpoint": 1, "category": 1, "last_seen": 1}))
	var endpoints []bson.M
	if epCur != nil {
		_ = epCur.All(ctx, &endpoints)
	}

	// sinks
	skCur, _ := models.SinksColl().Find(ctx, bson.M{
		"site_id": siteID,
		"$or": bson.A{
			bson.M{"source_url": rxContains(q)},
			bson.M{"page_url": rxContains(q)},
			bson.M{"func": rxContains(q)},
		},
	}, mopts.Find().SetLimit(lim).SetProjection(bson.M{"kind": 1, "source_url": 1, "line": 1, "col": 1, "last_detected_at": 1}))
	var sinks []bson.M
	if skCur != nil {
		_ = skCur.All(ctx, &sinks)
	}

	writeJSON(w, http.StatusOK, bson.M{
		"pages": pages, "endpoints": endpoints, "sinks": sinks,
	})
}
