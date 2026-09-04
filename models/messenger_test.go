package models

import "testing"

// TestMessengerProfileValidate verifies a Messenger sending profile is accepted
// only when it carries a page access token (stored in Password).
func TestMessengerProfileValidate(t *testing.T) {
	s := &SMTP{Interface: InterfaceMessenger}
	if err := s.Validate(); err != ErrMessengerTokenNotSpecified {
		t.Errorf("expected ErrMessengerTokenNotSpecified, got %v", err)
	}
	s.Password = "page-access-token"
	if err := s.Validate(); err != nil {
		t.Errorf("expected a token-only Messenger profile to validate, got %v", err)
	}
}
