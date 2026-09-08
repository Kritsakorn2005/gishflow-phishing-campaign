package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// linePushURL is the LINE Messaging API push endpoint.
const linePushURL = "https://api.line.me/v2/bot/message/push"

// lineProfileURL is the LINE Messaging API profile endpoint (per userId).
const lineProfileURL = "https://api.line.me/v2/bot/profile/"

var lineHTTPClient = &http.Client{Timeout: 15 * time.Second}

// SendLineMessage pushes a single text message to a LINE user via the LINE
// Messaging API. The token is the channel access token of the sending profile,
// and `to` is the recipient's LINE userId.
func SendLineMessage(token, to, text string) error {
	payload := map[string]interface{}{
		"to": to,
		"messages": []map[string]string{
			{"type": "text", "text": text},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, linePushURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := lineHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("LINE API returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// GetLineDisplayName fetches a LINE user's display name via the Messaging API
// profile endpoint, using the OA's channel access token. It's best-effort:
// any error (bad token, user not a friend, etc.) yields an empty string.
func GetLineDisplayName(token, userID string) string {
	if token == "" || userID == "" {
		return ""
	}
	req, err := http.NewRequest(http.MethodGet, lineProfileURL+userID, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := lineHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var p struct {
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return ""
	}
	return p.DisplayName
}
