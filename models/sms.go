package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// thaiBulkSMSURL is the ThaiBulkSMS v2 send endpoint.
const thaiBulkSMSURL = "https://api-v2.thaibulksms.com/sms"

var smsHTTPClient = &http.Client{Timeout: 20 * time.Second}

// SendThaiBulkSMS sends a single SMS via the ThaiBulkSMS v2 API. apiKey and
// apiSecret are the account credentials (sent as HTTP Basic auth), sender is
// the registered sender name, phone is the recipient's number and message is
// the text body.
func SendThaiBulkSMS(apiKey, apiSecret, sender, phone, message string) error {
	form := url.Values{}
	form.Set("msisdn", phone)
	form.Set("message", message)
	if sender != "" {
		form.Set("sender", sender)
	}
	req, err := http.NewRequest(http.MethodPost, thaiBulkSMSURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(apiKey, apiSecret)

	resp, err := smsHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ThaiBulkSMS returned %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
