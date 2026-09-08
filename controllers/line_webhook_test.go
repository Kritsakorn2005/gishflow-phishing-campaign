package controllers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
)

// TestLineWebhookParse verifies the LINE webhook payload is parsed into events
// with their source userIds, which is what the handler stores as contacts.
func TestLineWebhookParse(t *testing.T) {
	payload := `{
		"destination": "Ubotid",
		"events": [
			{"type": "follow",  "source": {"type": "user", "userId": "Uaaa"}},
			{"type": "message", "source": {"type": "user", "userId": "Ubbb"}},
			{"type": "message", "source": {"type": "group", "groupId": "Cxyz"}}
		]
	}`
	var wb lineWebhookBody
	if err := json.Unmarshal([]byte(payload), &wb); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(wb.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(wb.Events))
	}
	var ids []string
	for _, e := range wb.Events {
		if e.Source.UserID != "" {
			ids = append(ids, e.Source.UserID)
		}
	}
	// Two user events carry a userId; the group event has none.
	if len(ids) != 2 || ids[0] != "Uaaa" || ids[1] != "Ubbb" {
		t.Errorf("unexpected userIds: %v", ids)
	}
}

// TestVerifyLineSignature checks the HMAC-SHA256 signature verification: a
// signature made with the right secret passes; wrong secret, wrong signature,
// and empty values fail.
func TestVerifyLineSignature(t *testing.T) {
	secret := "channel-secret"
	body := []byte(`{"events":[]}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	valid := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if !verifyLineSignature(secret, body, valid) {
		t.Error("valid signature should verify")
	}
	if verifyLineSignature("wrong-secret", body, valid) {
		t.Error("wrong secret must not verify")
	}
	if verifyLineSignature(secret, body, "bad-signature") {
		t.Error("wrong signature must not verify")
	}
	if verifyLineSignature("", body, valid) || verifyLineSignature(secret, body, "") {
		t.Error("empty secret or signature must not verify")
	}
}
