package models

import "testing"

func cueNames(r PhishScaleResult) map[string]bool {
	m := map[string]bool{}
	for _, c := range r.Cues {
		m[c.Name] = true
	}
	return m
}

// TestPhishScaleEasy: a clumsy phish (many cues) with low premise alignment is
// the easiest to detect.
func TestPhishScaleEasy(t *testing.T) {
	subject := "URGENT: your account will be suspended"
	html := `<p>Dear Customer,</p><p>Congratulations! Kindly verify your password immediately or your account will be deleted.</p><a href="http://192.168.10.5/login">http://real-bank.com</a>`
	r := ScorePhishScale(subject, html, "", "promo@gmail.com", 1)

	names := cueNames(r)
	for _, want := range []string{
		"Urgency or pressure",
		"Generic, impersonal greeting",
		"Requests credentials or sensitive info",
		"Reward / too good to be true",
		"Threat or consequence",
		"Suspicious phrasing",
		"Sender uses a free email provider",
		"Link to a raw IP address",
		"Link text doesn't match its destination",
	} {
		if !names[want] {
			t.Errorf("expected cue %q to be detected", want)
		}
	}
	if r.CueBucket != "Many" {
		t.Errorf("expected Many cues, got %s (%d)", r.CueBucket, r.CueCount)
	}
	if r.DifficultyKey != "least" {
		t.Errorf("expected least difficulty, got %s", r.Difficulty)
	}
}

// TestPhishScaleHard: a clean, personalized message (few cues) with high premise
// alignment is the hardest to detect.
func TestPhishScaleHard(t *testing.T) {
	subject := "Project Falcon - review notes for tomorrow's meeting"
	html := `<p>Hi {{.FirstName}},</p><p>Here are the notes from our sync. Let me know your thoughts before we meet.</p><a href="{{.URL}}">Open the shared doc</a>`
	r := ScorePhishScale(subject, html, "", "colleague@company.com", 5)

	if r.CueBucket != "Few" {
		t.Errorf("expected Few cues, got %s (%d): %+v", r.CueBucket, r.CueCount, r.Cues)
	}
	if r.PremiseLevel != "High" {
		t.Errorf("expected High premise, got %s", r.PremiseLevel)
	}
	if r.DifficultyKey != "very" {
		t.Errorf("expected very difficult, got %s", r.Difficulty)
	}
	// The {{.FirstName}} personalization must suppress the generic-greeting cue.
	if cueNames(r)["Generic, impersonal greeting"] {
		t.Error("personalized message should not flag a generic greeting")
	}
}
