package models

import (
	"strings"
	"testing"
)

// TestParseVariants covers the three shapes the model might return: a wrapped
// object, a fenced code block, and a bare array.
func TestParseVariants(t *testing.T) {
	object := `{"variants":[{"subject":"Reset your password","body_html":"<p>Hi {{.FirstName}}</p>","red_flags":["urgency","generic link"]}]}`
	v, err := parseVariants(object)
	if err != nil || len(v) != 1 {
		t.Fatalf("object: err=%v n=%d", err, len(v))
	}
	if v[0].Subject != "Reset your password" || len(v[0].RedFlags) != 2 {
		t.Errorf("object parsed wrong: %+v", v[0])
	}

	fenced := "Here you go:\n```json\n{\"variants\":[{\"subject\":\"A\",\"body_html\":\"<p>x</p>\",\"red_flags\":[]},{\"subject\":\"B\",\"body_html\":\"<p>y</p>\",\"red_flags\":[]}]}\n```\nHope that helps!"
	v, err = parseVariants(fenced)
	if err != nil || len(v) != 2 {
		t.Fatalf("fenced: err=%v n=%d", err, len(v))
	}

	bare := `[{"subject":"Only one","body_html":"<p>z</p>","red_flags":["threat"]}]`
	v, err = parseVariants(bare)
	if err != nil || len(v) != 1 || v[0].Subject != "Only one" {
		t.Fatalf("bare: err=%v n=%d", err, len(v))
	}

	if _, err := parseVariants("sorry, I cannot help with that"); err == nil {
		t.Error("expected an error for a non-JSON response")
	}
}

// TestBuildPretextPrompt checks the prompt carries the scenario, language, and
// count, and instructs the model to use Gophish template variables.
func TestBuildPretextPrompt(t *testing.T) {
	system, user := buildPretextPrompt("IT password reset", "university staff", "Thai", 3)
	if !strings.Contains(system, "{{.URL}}") || !strings.Contains(system, "{{.FirstName}}") {
		t.Error("system prompt should reference Gophish template variables")
	}
	if !strings.Contains(system, "Thai") {
		t.Error("system prompt should carry the language")
	}
	if !strings.Contains(user, "IT password reset") || !strings.Contains(user, "3") {
		t.Errorf("user prompt missing scenario/count: %q", user)
	}
}
