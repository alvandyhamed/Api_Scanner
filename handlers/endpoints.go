package handlers

import (
	"SiteChecker/models"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

func EndpointsListHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	filter := bson.M{"site_id": siteID}

	if cat := strings.TrimSpace(r.URL.Query().Get("category")); cat != "" {
		filter["category"] = cat
	}
	if q := strings.TrimSpace(r.URL.Query().Get("q")); q != "" {
		filter["endpoint"] = rxContains(q)
	}
	if from, ok := qTime(r, "from"); ok {
		filter["last_seen"] = bson.M{"$gte": from}
	}
	if to, ok := qTime(r, "to"); ok {
		if m, ok := filter["last_seen"].(bson.M); ok {
			m["$lte"] = to
		} else {
			filter["last_seen"] = bson.M{"$lte": to}
		}
	}
	if minSeen, _ := strconv.Atoi(r.URL.Query().Get("min_seen")); minSeen > 0 {
		filter["seen_count"] = bson.M{"$gte": minSeen}
	}
	if maxSeen, _ := strconv.Atoi(r.URL.Query().Get("max_seen")); maxSeen > 0 {
		if m, ok := filter["seen_count"].(bson.M); ok {
			m["$lte"] = maxSeen
		} else {
			filter["seen_count"] = bson.M{"$lte": maxSeen}
		}
	}

	opts := mopts.Find().
		SetSort(qSort(r, "last_seen", -1)).
		SetLimit(qLimit(r)).
		SetSkip(qSkip(r)).
		SetProjection(bson.M{
			"endpoint":    1,
			"site_id":     1,
			"category":    1,
			"seen_count":  1,
			"last_seen":   1,
			"hosts":       1,
			"source_urls": 1,
		})

	cur, err := models.EndpointsColl().Find(ctx, filter, opts)
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
	total, _ := models.EndpointsColl().CountDocuments(ctx, filter)

	writeJSON(w, http.StatusOK, bson.M{
		"items": items, "total": total, "limit": qLimit(r), "skip": qSkip(r),
	})
}

func EndpointsStatsHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"site_id": siteID}}},
		bson.D{{Key: "$facet", Value: bson.M{
			"by_category": mongo.Pipeline{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$category", "count": bson.M{"$sum": 1}}}},
				bson.D{{Key: "$project", Value: bson.M{"category": "$_id", "count": 1, "_id": 0}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
			},
			"top_endpoints": mongo.Pipeline{
				bson.D{{Key: "$sort", Value: bson.M{"seen_count": -1}}},
				bson.D{{Key: "$limit", Value: 20}},
				bson.D{{Key: "$project", Value: bson.M{"endpoint": 1, "seen_count": 1, "_id": 0}}},
			},
		}}},
	}

	cur, err := models.EndpointsColl().Aggregate(ctx, pipeline)
	if err != nil {
		srvError(w, err)
		return
	}
	defer cur.Close(ctx)

	var out []bson.M
	if err := cur.All(ctx, &out); err != nil || len(out) == 0 {
		srvError(w, errors.New("aggregation failed"))
		return
	}
	writeJSON(w, http.StatusOK, out[0])
}
