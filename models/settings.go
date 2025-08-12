package models

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type DiscordSetting struct {
	ID         string    `bson:"_id"         json:"id"`
	WebhookURL string    `bson:"webhook_url" json:"webhook_url"`
	Enabled    bool      `bson:"enabled"     json:"enabled"`
	UpdatedAt  time.Time `bson:"updated_at"  json:"updated_at"`
}

func SettingsColl() *mongo.Collection { return DB.Collection("settings") }
