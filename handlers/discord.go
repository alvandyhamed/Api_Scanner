package handlers

import (
	"SiteChecker/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GET /api/settings/discord
func DiscordGetHandler(w http.ResponseWriter, r *http.Request) {
	var cfg models.DiscordSetting
	err := models.SettingsColl().FindOne(r.Context(), bson.M{"_id": "discord"}).Decode(&cfg)
	if err == mongo.ErrNoDocuments {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "webhook_masked": ""})
		return
	}
	if err != nil {
		srvError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":        cfg.Enabled,
		"webhook_masked": maskWebhook(cfg.WebhookURL),
	})
}

// POST /api/settings/discord/set
// body: { "webhook_url": "...", "enabled": true }
func DiscordSetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		badRequest(w, "POST only")
		return
	}
	var req struct {
		WebhookURL string `json:"webhook_url"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badRequest(w, "invalid json")
		return
	}

	upd := bson.M{"updated_at": time.Now()}
	if strings.TrimSpace(req.WebhookURL) != "" {
		if !isDiscordWebhook(req.WebhookURL) {
			badRequest(w, "invalid webhook_url")
			return
		}
		upd["webhook_url"] = strings.TrimSpace(req.WebhookURL)
	}
	if req.Enabled != nil {
		upd["enabled"] = *req.Enabled
	}

	_, err := models.SettingsColl().UpdateByID(
		r.Context(),
		"discord",
		bson.M{"$set": upd, "$setOnInsert": bson.M{"_id": "discord"}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		srvError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/settings/discord/test
func DiscordTestHandler(w http.ResponseWriter, r *http.Request) {
	var cfg models.DiscordSetting
	if err := models.SettingsColl().FindOne(
		r.Context(),
		bson.M{"_id": "discord", "enabled": true},
	).Decode(&cfg); err != nil {
		badRequest(w, "discord not configured/enabled")
		return
	}
	if err := sendDiscord(r.Context(), cfg.WebhookURL, "✅ Test from SiteChecker"); err != nil {
		srvError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// helpers

func isDiscordWebhook(s string) bool {
	return strings.HasPrefix(s, "https://discord.com/api/webhooks/") ||
		strings.Contains(s, "discordapp.com/api/webhooks/")
}

func maskWebhook(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 12 {
		return "****"
	}
	return s[:10] + "…" + s[len(s)-4:]
}

func sendDiscord(ctx context.Context, webhook, content string) error {
	body := map[string]string{"content": content}
	b, _ := json.Marshal(body)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord http %d", resp.StatusCode)
	}
	return nil
}
