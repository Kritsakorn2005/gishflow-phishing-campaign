package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// PretextVariant is one AI-generated phishing pretext for a training
// simulation. RedFlags lists the cues a trainee could use to spot it (these
// double as teachable-moment talking points and feed the NIST Phish Scale).
type PretextVariant struct {
	Subject  string   `json:"subject"`
	BodyHTML string   `json:"body_html"`
	RedFlags []string `json:"red_flags"`
}

// ErrAINotConfigured is returned when the AI settings are missing.
var ErrAINotConfigured = errors.New("AI is not configured - set the \"ai\" block (base_url, api_key, model) in config.json")

var aiHTTPClient = &http.Client{Timeout: 60 * time.Second}

// AIConfigured reports whether the AI generator has the settings it needs.
func AIConfigured() bool {
	return conf != nil && conf.AIConf.BaseURL != "" && conf.AIConf.APIKey != "" && conf.AIConf.Model != ""
}

// buildPretextPrompt returns the system and user messages for generating
// simulated phishing pretexts for an authorized, consented training exercise.
func buildPretextPrompt(scenario, targetContext, language string, n int) (string, string) {
	system := "You are assisting an authorized security-awareness team that runs consented phishing " +
		"simulations for internal employee training. Generate realistic simulated phishing email pretexts " +
		"for the given scenario. This is a controlled exercise with participant consent and no real payloads. " +
		"Use the Gophish template variables {{.FirstName}} for personalization and {{.URL}} for the link. " +
		"Respond with ONLY a JSON object of the form " +
		"{\"variants\":[{\"subject\":\"...\",\"body_html\":\"...\",\"red_flags\":[\"...\"]}]}. " +
		"body_html is simple inline HTML. red_flags lists the phishing cues a trainee could notice. " +
		"Write the email in " + language + "."
	user := fmt.Sprintf("Scenario: %s\nTarget context: %s\nProduce exactly %d distinct variant(s).",
		strings.TrimSpace(scenario), strings.TrimSpace(targetContext), n)
	return system, user
}

// GeneratePretexts asks the configured AI provider for n pretext variants.
func GeneratePretexts(scenario, targetContext, language string, n int) ([]PretextVariant, error) {
	if !AIConfigured() {
		return nil, ErrAINotConfigured
	}
	if n < 1 {
		n = 1
	}
	if n > 5 {
		n = 5
	}
	if strings.TrimSpace(language) == "" {
		language = "English"
	}
	system, user := buildPretextPrompt(scenario, targetContext, language, n)

	var content string
	var err error
	if strings.EqualFold(conf.AIConf.Provider, "anthropic") {
		content, err = aiCallAnthropic(system, user)
	} else {
		content, err = aiCallOpenAI(system, user)
	}
	if err != nil {
		return nil, err
	}
	return parseVariants(content)
}

// aiCallOpenAI calls an OpenAI-compatible chat-completions endpoint.
func aiCallOpenAI(system, user string) (string, error) {
	payload := map[string]interface{}{
		"model": conf.AIConf.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.8,
	}
	rb, err := aiPost(conf.AIConf.BaseURL, payload, map[string]string{
		"Authorization": "Bearer " + conf.AIConf.APIKey,
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rb, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("AI returned no choices")
	}
	return out.Choices[0].Message.Content, nil
}

// aiCallAnthropic calls the Anthropic Messages API.
func aiCallAnthropic(system, user string) (string, error) {
	payload := map[string]interface{}{
		"model":      conf.AIConf.Model,
		"max_tokens": 2000,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": user}},
	}
	rb, err := aiPost(conf.AIConf.BaseURL, payload, map[string]string{
		"x-api-key":         conf.AIConf.APIKey,
		"anthropic-version": "2023-06-01",
	})
	if err != nil {
		return "", err
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rb, &out); err != nil {
		return "", err
	}
	if len(out.Content) == 0 {
		return "", errors.New("AI returned no content")
	}
	return out.Content[0].Text, nil
}

// aiPost sends a JSON POST and returns the raw response body, erroring on a
// non-2xx status.
func aiPost(url string, payload interface{}, headers map[string]string) ([]byte, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := aiHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI API returned %d: %s", resp.StatusCode, string(rb))
	}
	return rb, nil
}

// parseVariants extracts pretext variants from the model's text response,
// tolerating markdown fences and either a {"variants":[...]} object or a bare
// array.
func parseVariants(content string) ([]PretextVariant, error) {
	js := extractJSON(content)
	var wrap struct {
		Variants []PretextVariant `json:"variants"`
	}
	if err := json.Unmarshal([]byte(js), &wrap); err == nil && len(wrap.Variants) > 0 {
		return wrap.Variants, nil
	}
	var arr []PretextVariant
	if err := json.Unmarshal([]byte(js), &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	return nil, errors.New("could not parse the AI response as pretext variants")
}

// extractJSON pulls the JSON payload out of a model response, stripping code
// fences and any prose around it.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[i+3:]
		s = strings.TrimPrefix(s, "json")
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	s = strings.TrimSpace(s)
	start := strings.IndexAny(s, "{[")
	end := strings.LastIndexAny(s, "}]")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
