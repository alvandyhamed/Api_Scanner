package models

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Mongo *mongo.Client
var DB *mongo.Database

// models/mongo.go
func InitMongo(ctx context.Context) error {
	uri := os.Getenv("MONGO_URI")

	candidates := []string{}
	if uri != "" {
		candidates = []string{uri}
	} else {
		candidates = []string{
			"mongodb://mongo:27017/sitechecker",     // برای داخل داکر
			"mongodb://127.0.0.1:27018/sitechecker", // برای go run بیرون
		}
	}

	var lastErr error
	for _, u := range candidates {
		c, err := mongo.Connect(ctx, options.Client().
			ApplyURI(u).
			SetServerSelectionTimeout(5*time.Second),
		)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err = c.Ping(pingCtx, nil); err == nil {
				Mongo = c
				// استخراج نام دیتابیس
				dbName := os.Getenv("MONGO_DB")
				if dbName == "" {
					pu, _ := url.Parse(u)
					dbName = strings.TrimPrefix(pu.Path, "/")
					if dbName == "" {
						dbName = "sitechecker"
					}
				}
				DB = c.Database(dbName)
				return nil
			}
			_ = c.Disconnect(context.Background())
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no mongo candidates succeeded")
	}
	return fmt.Errorf("mongo init failed: %w", lastErr)
}

func SitesColl() *mongo.Collection     { return DB.Collection("sites") }
func PagesColl() *mongo.Collection     { return DB.Collection("pages") }
func EndpointsColl() *mongo.Collection { return DB.Collection("endpoints") }

func EnsureIndexes(ctx context.Context) error {
	// sites: _id یکتا (site_id = hamed0x.ir)
	_, _ = SitesColl().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// pages: url_norm یکتا + جستجوهای متداول
	_, _ = PagesColl().Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "url_norm", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "site_id", Value: 1}, {Key: "host", Value: 1}, {Key: "path", Value: 1}}},
	})

	// endpoints: (site_id, endpoint) یکتا
	_, _ = EndpointsColl().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "site_id", Value: 1}, {Key: "endpoint", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	log.Printf("[mongo] connected db=%s uri=%s", DB.Name(), os.Getenv("MONGO_URI"))
	return nil
}
