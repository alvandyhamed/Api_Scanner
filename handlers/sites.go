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
