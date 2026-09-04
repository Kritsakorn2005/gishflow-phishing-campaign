package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// messengerSendURL is the Facebook Messenger Send API endpoint. The page access
// token is supplied as a query parameter.
const messengerSendURL = "https://graph.facebook.com/v19.0/me/messages"

var messengerHTTPClient = &http.Client{Timeout: 15 * time.Second}

// SendMessengerMessage sends a single text message to a Messenger user via the
// Facebook Send API. token is the page access token of the sending profile, and
// psid is the recipient's page-scoped ID (obtained when they message the page).
// messaging_type RESPONSE requires the recipient to have messaged the page
// within the last 24 hours.
func SendMessengerMessage(token, psid, text string) error {
	payload := map[string]interface{}{
		"recipient":     map[string]string{"id": psid},
		"messaging_type": "RESPONSE",
		"message":       map[string]string{"text": text},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, messengerSendURL+"?access_token="+token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := messengerHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Messenger API returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
