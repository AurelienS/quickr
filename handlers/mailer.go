package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// SendinblueClient sends transactional emails via API
// Expects SENDINBLUE_API_KEY env var
// Uses a very simple template

type SendinblueClient struct {
	apiKey string
}

func NewSendinblueClient() *SendinblueClient {
	return &SendinblueClient{apiKey: os.Getenv("SENDINBLUE_API_KEY")}
}

func (s *SendinblueClient) SendMagicLink(to string, link string) error {
	if s.apiKey == "" {
		return nil // no-op in dev
	}
	payload := map[string]interface{}{
		"sender": map[string]string{"name": "Quickr", "email": "no-reply@quickr.local"},
		"to":     []map[string]string{{"email": to}},
		"subject": "Your one-time sign-in link",
		"htmlContent": fmt.Sprintf("<p>Click to sign in. This link works once and expires in 7 days.</p><p><a href=\"%s\">Sign in</a></p>", link),
	}
	buf, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "https://api.sendinblue.com/v3/smtp/email", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sendinblue error: %d", resp.StatusCode)
	}
	return nil
}