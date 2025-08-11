package handlers

import (
	"SiteChecker/models"
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

func SinksListHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	filter := bson.M{"site_id": siteID}

	if kinds := strings.TrimSpace(r.URL.Query().Get("kind")); kinds != "" {
		arr := strings.Split(kinds, ",")
		for i := range arr {
			arr[i] = strings.TrimSpace(arr[i])
		}
		filter["kind"] = bson.M{"$in": arr}
	}
	if pageURL := strings.TrimSpace(r.URL.Query().Get("page_url")); pageURL != "" {
		filter["page_url"] = pageURL
	}
	if src := strings.TrimSpace(r.URL.Query().Get("source_url")); src != "" {
		filter["source_url"] = rxContains(src)
	}
	if fn := strings.TrimSpace(r.URL.Query().Get("func")); fn != "" {
		filter["func"] = rxContains(fn)
	}
	if from, ok := qTime(r, "from"); ok {
		filter["last_detected_at"] = bson.M{"$gte": from}
	}
	if to, ok := qTime(r, "to"); ok {
		if m, ok := filter["last_detected_at"].(bson.M); ok {
			m["$lte"] = to
		} else {
			filter["last_detected_at"] = bson.M{"$lte": to}
		}
	}

	opts := mopts.Find().
		SetSort(qSort(r, "last_detected_at", -1)).
		SetLimit(qLimit(r)).
		SetSkip(qSkip(r)).
		SetProjection(bson.M{
			"site_id":           1,
			"page_url":          1,
			"source_type":       1,
			"source_url":        1,
			"kind":              1,
			"func":              1,
			"line":              1,
			"col":               1,
			"snippet":           1,
			"hits":              1,
			"first_detected_at": 1,
			"last_detected_at":  1,
		})

	cur, err := models.SinksColl().Find(ctx, filter, opts)
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
	total, _ := models.SinksColl().CountDocuments(ctx, filter)

	writeJSON(w, http.StatusOK, bson.M{
		"items": items, "total": total, "limit": qLimit(r), "skip": qSkip(r),
	})
}

func SinksStatsHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	match := bson.M{"site_id": siteID}
	if pageURL := strings.TrimSpace(r.URL.Query().Get("page_url")); pageURL != "" {
		match["page_url"] = pageURL
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: match}},
		bson.D{{Key: "$facet", Value: bson.M{
			"by_kind": mongo.Pipeline{
				bson.D{{Key: "$group", Value: bson.M{"_id": "$kind", "count": bson.M{"$sum": 1}}}},
				bson.D{{Key: "$project", Value: bson.M{"kind": "$_id", "count": 1, "_id": 0}}},
				bson.D{{Key: "$sort", Value: bson.M{"count": -1}}},
			},
			"recent": mongo.Pipeline{
				bson.D{{Key: "$sort", Value: bson.M{"last_detected_at": -1}}},
				bson.D{{Key: "$limit", Value: 20}},
				bson.D{{Key: "$project", Value: bson.M{"kind": 1, "last_detected_at": 1, "source_url": 1, "_id": 0}}},
			},
		}}},
	}

	cur, err := models.SinksColl().Aggregate(ctx, pipeline)
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
