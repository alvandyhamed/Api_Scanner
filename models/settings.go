package models

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// یک رکورد ثابت با _id = "discord"
type DiscordSetting struct {
	ID         string    `bson:"_id"         json:"id"`
	WebhookURL string    `bson:"webhook_url" json:"webhook_url"`
	Enabled    bool      `bson:"enabled"     json:"enabled"`
	UpdatedAt  time.Time `bson:"updated_at"  json:"updated_at"`
}

func SettingsColl() *mongo.Collection { return DB.Collection("settings") }

// خواندن تنظیمات؛ اگر هنوز ذخیره نشده بود، آبجکت خالی با id=discord برمی‌گرده
func GetDiscordSettings(ctx context.Context) (DiscordSetting, error) {
	var out DiscordSetting
	err := SettingsColl().FindOne(ctx, bson.M{"_id": "discord"}).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return DiscordSetting{ID: "discord"}, nil
	}
	return out, err
}

// ذخیره/آپدیت (Upsert) تنظیمات
func SetDiscordSettings(ctx context.Context, webhook string, enabled bool) error {
	now := time.Now()
	_, err := SettingsColl().UpdateByID(ctx, "discord", bson.M{
		"$set": bson.M{
			"webhook_url": webhook,
			"enabled":     enabled,
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"_id": "discord",
		},
	}, options.Update().SetUpsert(true))
	return err
}
