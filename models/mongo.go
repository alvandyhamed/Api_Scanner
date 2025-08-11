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

// InitMongo: اتصال به Mongo و انتخاب دیتابیس
func InitMongo(ctx context.Context) error {
	uri := os.Getenv("MONGO_URI")

	candidates := []string{}
	if uri != "" {
		candidates = []string{uri}
	} else {
		candidates = []string{
			"mongodb://mongo:27017/sitechecker",     // داخل docker compose
			"mongodb://127.0.0.1:27018/sitechecker", // اجرای لوکال
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
			if err = c.Ping(pingCtx, nil); err == nil {
				Mongo = c

				dbName := os.Getenv("MONGO_DB")
				if dbName == "" {
					pu, _ := url.Parse(u)
					dbName = strings.TrimPrefix(pu.Path, "/")
					if dbName == "" {
						dbName = "sitechecker"
					}
				}
				DB = c.Database(dbName)
				log.Printf("[mongo] connected db=%s uri=%s", DB.Name(), u)
				return nil
			}
			cancel()
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
func SinksColl() *mongo.Collection     { return DB.Collection("sinks") }

// EnsureIndexes: ساخت ایندکس‌ها (و حذف ایندکس قدیمی sinks اگر بود)
func EnsureIndexes(ctx context.Context) error {
	// sites
	_, _ = SitesColl().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// pages
	_, _ = PagesColl().Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "url_norm", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_url_norm")},
		{Keys: bson.D{{Key: "site_id", Value: 1}, {Key: "host", Value: 1}, {Key: "path", Value: 1}}, Options: options.Index().SetName("q_site_host_path")},
	})

	// endpoints
	_, _ = EndpointsColl().Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "site_id", Value: 1}, {Key: "endpoint", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_site_endpoint"),
	})

	// sinks — حذف ایندکس قدیمی اگر وجود داشت
	// (اسم پیش‌فرض: site_id_1_page_url_1_source_url_1_kind_1_line_1_col_1)
	_, _ = SinksColl().Indexes().DropOne(ctx, "site_id_1_page_url_1_source_url_1_kind_1_line_1_col_1")

	iv := SinksColl().Indexes()

	// 1) ایندکس یکتای sig (کلید ددوپ)
	_, _ = iv.CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "sig", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_sig"),
	})

	// 2) ایندکس برای گزارش اخیر بر اساس نوع sink
	_, _ = iv.CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "site_id", Value: 1}, {Key: "kind", Value: 1}, {Key: "last_detected_at", Value: -1}},
		Options: options.Index().SetName("q_site_kind_recent"),
	})

	// 3) ایندکس برای گزارش صفحه/نوع
	_, _ = iv.CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "site_id", Value: 1}, {Key: "page_url", Value: 1}, {Key: "kind", Value: 1}},
		Options: options.Index().SetName("q_site_page_kind"),
	})

	return nil
}
