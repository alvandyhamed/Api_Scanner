// functions/discord.go
package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type discordPayload struct {
	Content string `json:"content"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func SendDiscordWebhook(ctx context.Context, webhookURL, content string) error {
	if webhookURL == "" {
		return errors.New("discord webhook is empty")
	}
	body, _ := json.Marshal(discordPayload{Content: content})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord http error: %w", err)
	}
	defer resp.Body.Close()

	// Discord معمولاً 204 برمی‌گردونه؛ هر 2xx رو موفق بدون.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord unexpected status: %d", resp.StatusCode)
	}
	return nil
}
