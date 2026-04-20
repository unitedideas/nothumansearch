// Package notify fires best-effort Discord alerts. Mirrors the ADB
// notify package exactly — DISCORD_BOT_TOKEN + DISCORD_CHANNEL_ID from env,
// 5s timeout, never blocks the request path.
package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

func Discord(content string) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	channel := os.Getenv("DISCORD_CHANNEL_ID")
	if token == "" || channel == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{"content": content})
	req, err := http.NewRequest("POST",
		"https://discord.com/api/v10/channels/"+channel+"/messages",
		bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func DiscordAsync(content string) {
	go Discord(content)
}
