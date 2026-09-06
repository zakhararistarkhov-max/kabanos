// Package notify contains outbound notification channels. Telegram is the
// first: the worker delivers the morning digest through it. The client is a
// thin wrapper over the Bot API so it stays dependency-free and testable.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Telegram delivers messages via the Bot API. When Enabled is false (no token
// configured) Send is a no-op, so the rest of the system runs unchanged in
// environments without a bot.
type Telegram struct {
	token   string
	client  *http.Client
	Enabled bool
}

func NewTelegram(token string) *Telegram {
	return &Telegram{
		token:   token,
		Enabled: token != "",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// DigestPayload is the body of an outbox `telegram.daily_digest` message.
type DigestPayload struct {
	ChatID  int64  `json:"chatId"`
	Message string `json:"message"`
}

// Send posts a Markdown message to a chat.
func (t *Telegram) Send(ctx context.Context, chatID int64, text string) error {
	if !t.Enabled {
		return nil
	}
	body, _ := json.Marshal(map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage: status %d", resp.StatusCode)
	}
	return nil
}
