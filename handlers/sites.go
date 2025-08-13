package handlers

import (
	"SiteChecker/models"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

func SitesListHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	filter := bson.M{}
	if q != "" {
		// روی _id (site_id) و hosts
		filter["$or"] = bson.A{
			bson.M{"_id": rxContains(q)},
			bson.M{"hosts": rxContains(q)},
		}
	}

	opts := mopts.Find().
		SetSort(bson.D{{Key: "last_scan_at", Value: -1}}).
		SetLimit(qLimit(r)).
		SetSkip(qSkip(r))

	cur, err := models.SitesColl().Find(ctx, filter, opts)
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

	total, _ := models.SitesColl().CountDocuments(ctx, filter)
	writeJSON(w, http.StatusOK, bson.M{
		"items": items, "total": total, "limit": qLimit(r), "skip": qSkip(r),
	})
}

// POST /api/sites/delete  { site_id }
func SiteDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		badRequest(w, "POST/DELETE only")
		http.Error(w, "POST/DELETE only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SiteID string `json:"site_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	if req.SiteID == "" {
		badRequest(w, "site_id required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// حذف از تمام collection ها
	siteID := req.SiteID

	// 1. حذف Pages
	pagesResult, _ := models.PagesColl().DeleteMany(ctx, bson.M{"site_id": siteID})

	// 2. حذف Endpoints
	endpointsResult, _ := models.EndpointsColl().DeleteMany(ctx, bson.M{"site_id": siteID})

	// 3. حذف Sinks
	sinksResult, _ := models.SinksColl().DeleteMany(ctx, bson.M{"site_id": siteID})

	// 4. حذف Watches
	watchesResult, _ := models.WatchesColl().DeleteMany(ctx, bson.M{"site_id": siteID})

	// 5. حذف Site اصلی
	siteResult, err := models.SitesColl().DeleteOne(ctx, bson.M{"_id": siteID})
	if err != nil {
		srvError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, bson.M{
		"ok":      true,
		"site_id": siteID,
		"deleted": bson.M{
			"site":      siteResult.DeletedCount,
			"pages":     pagesResult.DeletedCount,
			"endpoints": endpointsResult.DeletedCount,
			"sinks":     sinksResult.DeletedCount,
			"watches":   watchesResult.DeletedCount,
		},
	})
}
