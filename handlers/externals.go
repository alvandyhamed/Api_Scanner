package handlers

import (
	"SiteChecker/models"
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func ExternalsListHandler(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSpace(r.URL.Query().Get("site_id"))
	if siteID == "" {
		badRequest(w, "site_id is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"site_id": siteID}}},
		bson.D{{Key: "$project", Value: bson.M{"externals": 1}}},
		bson.D{{Key: "$replaceWith", Value: "$externals"}},
		bson.D{{Key: "$project", Value: bson.M{"k": bson.M{"$objectToArray": "$$ROOT"}}}},
		bson.D{{Key: "$unwind", Value: "$k"}},
		bson.D{{Key: "$replaceWith", Value: bson.M{"$mergeObjects": bson.A{"$k.v", bson.M{"ext_site_id": "$k.k"}}}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":       "$ext_site_id",
			"hosts":     bson.M{"$addToSet": "$hosts"},
			"endpoints": bson.M{"$sum": bson.M{"$size": bson.M{"$ifNull": bson.A{"$endpoints", bson.A{}}}}},
			"resources": bson.M{"$sum": bson.M{"$size": bson.M{"$ifNull": bson.A{"$resources", bson.A{}}}}},
			"scripts":   bson.M{"$sum": bson.M{"$size": bson.M{"$ifNull": bson.A{"$scripts", bson.A{}}}}},
		}}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":         0,
			"ext_site_id": "$_id",
			"hosts":       bson.M{"$reduce": bson.M{"input": "$hosts", "initialValue": bson.A{}, "in": bson.M{"$setUnion": bson.A{"$$value", "$$this"}}}},
			"endpoints":   1, "resources": 1, "scripts": 1,
		}}},
		bson.D{{Key: "$sort", Value: bson.M{"scripts": -1, "resources": -1}}},
		bson.D{{Key: "$skip", Value: qSkip(r)}},
		bson.D{{Key: "$limit", Value: qLimit(r)}},
	}

	cur, err := models.PagesColl().Aggregate(ctx, pipeline)
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

	writeJSON(w, http.StatusOK, bson.M{
		"items": items, "total": len(items), "limit": qLimit(r), "skip": qSkip(r),
	})
}
