package controllers

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gophish/gophish/models"
)

// TestBuildQuizForm checks that the rendered quiz form carries the marker, the
// recipient id, and a radio input per question.
func TestBuildQuizForm(t *testing.T) {
	qs := []models.QuizQuestion{
		{Q: "Is this phishing?", Choices: []string{"Yes", "No"}, Answer: 0},
	}
	form := buildQuizForm(qs, "abc123")
	for _, want := range []string{`name="__gfquiz"`, `value="abc123"`, "Is this phishing?", `name="gfq_0"`, "Yes", "No"} {
		if !strings.Contains(form, want) {
			t.Errorf("quiz form missing %q", want)
		}
	}
}

// TestParseQuizAnswers checks that gfq_-prefixed inputs are collected and
// non-quiz fields (like rid) are ignored.
func TestParseQuizAnswers(t *testing.T) {
	form := url.Values{}
	form.Set("gfq_0", "1")
	form.Set("gfq_1", "0")
	form.Set("rid", "x")
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ParseForm()

	answers := parseQuizAnswers(r)
	if answers["0"] != 1 || answers["1"] != 0 {
		t.Errorf("unexpected answers: %v", answers)
	}
	if _, ok := answers["rid"]; ok {
		t.Error("non-quiz field rid should be ignored")
	}
}
