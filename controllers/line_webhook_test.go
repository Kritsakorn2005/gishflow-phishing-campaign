package controllers

import (
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
