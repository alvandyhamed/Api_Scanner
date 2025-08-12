package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WatchSummary struct {
	Endpoints int       `bson:"endpoints,omitempty" json:"endpoints,omitempty"`
	Sinks     int       `bson:"sinks,omitempty"     json:"sinks,omitempty"`
	LastEP    time.Time `bson:"last_ep,omitempty"   json:"last_ep,omitempty"`
	LastSink  time.Time `bson:"last_sink,omitempty" json:"last_sink,omitempty"`
	Digest    string    `bson:"digest,omitempty"    json:"digest,omitempty"`
}

type WatchDoc struct {
	ID          any          `bson:"_id,omitempty"   json:"_id"`
	SiteID      string       `bson:"site_id"         json:"site_id"`
	URL         string       `bson:"url"             json:"url"`
	URLNorm     string       `bson:"url_norm"        json:"url_norm"`
	Enabled     bool         `bson:"enabled"         json:"enabled"`
	FreqMin     int          `bson:"freq_min"        json:"freq_min"`
	NextRunAt   time.Time    `bson:"next_run_at"     json:"next_run_at"`
	LastRunAt   time.Time    `bson:"last_run_at"     json:"last_run_at"`
	LastChange  time.Time    `bson:"last_change_at,omitempty" json:"last_change_at,omitempty"`
	LastSummary WatchSummary `bson:"last_summary,omitempty"   json:"last_summary,omitempty"`
	CreatedAt   time.Time    `bson:"created_at"      json:"created_at"`
	UpdatedAt   time.Time    `bson:"updated_at"      json:"updated_at"`
}

// ⬅️ اینجا هم از DB.Collection استفاده کن
func WatchesColl() *mongo.Collection { return DB.Collection("watches") }

func EnsureWatchIndexes(ctx context.Context) error {
	_, err := WatchesColl().Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "site_id", Value: 1}, {Key: "url_norm", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}, {Key: "next_run_at", Value: 1}},
		},
	})
	return err
}
