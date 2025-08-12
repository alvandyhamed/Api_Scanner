package functions

import (
	"SiteChecker/models"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type discordMsg struct {
	Content string `json:"content"`
}

func notifyDiscord(ctx context.Context, siteID, url string, s models.WatchSummary) error {
	// گلوبال
	var cfg models.DiscordSetting
	_ = models.SettingsColl().FindOne(ctx, bson.M{"_id": "discord", "enabled": true}).Decode(&cfg)
	if cfg.WebhookURL == "" {
		return nil
	}

	content := "🔔 **Change detected** on `" + siteID + "`\n" +
		"URL: " + url + "\n" +
		"Endpoints: " + itoa(s.Endpoints) + " | Sinks: " + itoa(s.Sinks) + "\n" +
		"Time: " + time.Now().Format(time.RFC3339)

	b, _ := json.Marshal(discordMsg{Content: content})
	rq, _ := http.NewRequestWithContext(ctx, "POST", cfg.WebhookURL, bytes.NewReader(b))
	rq.Header.Set("Content-Type", "application/json")
	_, _ = http.DefaultClient.Do(rq)
	return nil
}
