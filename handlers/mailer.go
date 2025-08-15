package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// SendinblueClient sends transactional emails via Brevo/Sendinblue API
// Required env: SENDINBLUE_API_KEY, SENDER_EMAIL, SENDER_NAME
// Optional: BREVO_API_BASE (default https://api.brevo.com)

type SendinblueClient struct {
	apiKey      string
	baseURL     string
	senderEmail string
	senderName  string
}

func NewSendinblueClient() *SendinblueClient {
	base := os.Getenv("BREVO_API_BASE")
	if base == "" {
		base = "https://api.brevo.com"
	}
	return &SendinblueClient{
		apiKey:      os.Getenv("SENDINBLUE_API_KEY"),
		baseURL:     base,
		senderEmail: os.Getenv("SENDER_EMAIL"),
		senderName:  os.Getenv("SENDER_NAME"),
	}
}

func (s *SendinblueClient) SendMagicLink(to string, link string) error {
	if s.apiKey == "" {
		err := errors.New("SENDINBLUE_API_KEY missing")
		log.Println("Email send error:", err)
		return err
	}
	if s.senderEmail == "" {
		err := errors.New("SENDER_EMAIL missing")
		log.Println("Email send error:", err)
		return err
	}
	if s.senderName == "" {
		s.senderName = "Quickr"
	}
	payload := map[string]interface{}{
		"sender":   map[string]string{"name": s.senderName, "email": s.senderEmail},
		"to":       []map[string]string{{"email": to}},
		"subject":  "Your one-time sign-in link",
		"htmlContent": fmt.Sprintf("<p>Click to sign in. This link works once and expires in 7 days.</p><p><a href=\"%s\">Sign in</a></p>", link),
	}
	buf, _ := json.Marshal(payload)
	endpoint := s.baseURL + "/v3/smtp/email"
	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", s.apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Email send error:", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("brevo smtp error: %d - %s", resp.StatusCode, string(body))
		log.Println("Email send error:", err)
		return err
	}
	log.Println("Email sent via Brevo to", to)
	return nil
}