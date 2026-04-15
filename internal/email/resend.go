// Package email wraps the Resend HTTP API for transactional sends.
//
// Auth: reads RESEND_API_KEY from env. The nothumansearch.ai domain is
// already verified in Resend (SPF + DKIM) for both sending and receiving.
package email

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const resendURL = "https://api.resend.com/emails"

type Client struct {
	apiKey string
	from   string
	http   *http.Client
}

// NewClientFromEnv pulls the key from RESEND_API_KEY and uses a default
// from-address on the verified nothumansearch.ai domain.
func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("RESEND_API_KEY")
	if key == "" {
		return nil, errors.New("RESEND_API_KEY not set")
	}
	from := os.Getenv("MONITOR_FROM_EMAIL")
	if from == "" {
		from = "Not Human Search <alerts@nothumansearch.ai>"
	}
	return &Client{
		apiKey: key,
		from:   from,
		http:   &http.Client{Timeout: 15 * time.Second},
	}, nil
}

type sendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

// Send delivers one email. Either html or text (or both) must be non-empty.
// Returns the Resend message id on success.
func (c *Client) Send(to, subject, htmlBody, textBody string) (string, error) {
	if to == "" || subject == "" || (htmlBody == "" && textBody == "") {
		return "", errors.New("to, subject, and body required")
	}
	body, _ := json.Marshal(sendRequest{
		From:    c.from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	})
	req, err := http.NewRequest("POST", resendURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("resend %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &out)
	return out.ID, nil
}
