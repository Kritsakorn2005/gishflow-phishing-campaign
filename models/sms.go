package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var smsHTTPClient = &http.Client{Timeout: 20 * time.Second}

// SendSMSViaGateway sends a single SMS through a self-hosted Android SMS
// gateway (e.g. the "SMS Gateway for Android" app in local mode) instead of a
// paid SMS provider, so a real SIM sends the message. baseURL is the phone's
// HTTP endpoint (http://<ip>:<port>), username/password are the app's HTTP
// Basic-auth credentials (may be empty if the app has none), phone is the
// recipient's number and message is the text body.
func SendSMSViaGateway(baseURL, username, password, phone, message string) error {
	endpoint := strings.TrimRight(baseURL, "/") + "/message"
	payload, err := json.Marshal(map[string]interface{}{
		"message":      message,
		"phoneNumbers": []string{phone},
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := smsHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS gateway returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
