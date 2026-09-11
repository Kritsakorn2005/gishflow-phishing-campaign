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

	log "github.com/gophish/gophish/logger"
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
		"body_html is simple inline HTML. red_flags lists the phishing cues a trainee could notice. " +
		"Write the email in " + language + ". " +
		"Output ONLY a raw JSON object of the exact form " +
		"{\"variants\":[{\"subject\":\"...\",\"body_html\":\"...\",\"red_flags\":[\"...\"]}]} " +
		"- no markdown code fences, no explanation, and no text before or after the JSON."
	user := fmt.Sprintf("Scenario: %s\nTarget context: %s\nProduce exactly %d distinct variant(s).",
		strings.TrimSpace(scenario), strings.TrimSpace(targetContext), n)
	return system, user
}

// GeneratePretexts asks the configured AI provider for n pretext variants. When
// model is empty the configured default model is used.
func GeneratePretexts(scenario, targetContext, language, model string, n int) ([]PretextVariant, error) {
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
	if strings.TrimSpace(model) == "" {
		model = conf.AIConf.Model
	}
	system, user := buildPretextPrompt(scenario, targetContext, language, n)

	var content string
	var err error
	if strings.EqualFold(conf.AIConf.Provider, "anthropic") {
		content, err = aiCallAnthropic(system, user, model)
	} else {
		content, err = aiCallOpenAI(system, user, model)
	}
	if err != nil {
		return nil, err
	}
	variants, perr := parseVariants(content)
	if perr != nil {
		log.Errorf("AI pretext parse failed (model=%s). Raw response: %s", model, content)
	}
	return variants, perr
}

// quizDifficultyGuidance maps a difficulty level to instructions for the model.
func quizDifficultyGuidance(difficulty string) string {
	switch strings.ToLower(strings.TrimSpace(difficulty)) {
	case "easy":
		return "Easy: test basic recall of facts stated directly in the training content. Make the distractors clearly wrong."
	case "hard":
		return "Hard: scenario-based questions that require judgment and applying the concepts to new situations. Make the distractors subtle and tempting so the learner must think carefully."
	default:
		return "Medium: require the learner to understand and apply the concepts, not just recall them. Make the distractors plausible."
	}
}

// buildQuizPrompt returns the system and user messages for generating a
// multiple-choice quiz that tests a security-awareness training page.
func buildQuizPrompt(topic, content, language, difficulty string, n int) (string, string) {
	system := "You are creating a multiple-choice quiz for a security-awareness training exercise. " +
		"Generate quiz questions that test whether the learner understood the training material. " +
		"Each question must have exactly one correct answer and 3 to 4 answer choices. " +
		quizDifficultyGuidance(difficulty) + " " +
		"Write every question and choice in " + language + ". " +
		"Output ONLY a raw JSON array of the exact form " +
		"[{\"q\":\"...\",\"choices\":[\"...\",\"...\",\"...\"],\"answer\":0}] " +
		"where answer is the zero-based index of the correct choice in that question's choices array. " +
		"No markdown code fences, no explanation, and no text before or after the JSON."
	user := "Training content the quiz should test:\n" + strings.TrimSpace(content)
	if t := strings.TrimSpace(topic); t != "" {
		user += "\n\nFocus especially on this topic: " + t
	}
	user += fmt.Sprintf("\n\nProduce exactly %d question(s).", n)
	return system, user
}

// GenerateQuiz asks the configured AI provider for n multiple-choice quiz
// questions based on the given training content. When model is empty the
// configured default model is used.
func GenerateQuiz(topic, content, language, difficulty, model string, n int) ([]QuizQuestion, error) {
	if !AIConfigured() {
		return nil, ErrAINotConfigured
	}
	if n < 1 {
		n = 1
	}
	if n > 20 {
		n = 20
	}
	if strings.TrimSpace(language) == "" {
		language = "English"
	}
	if strings.TrimSpace(model) == "" {
		model = conf.AIConf.Model
	}
	system, user := buildQuizPrompt(topic, content, language, difficulty, n)

	var content2 string
	var err error
	if strings.EqualFold(conf.AIConf.Provider, "anthropic") {
		content2, err = aiCallAnthropic(system, user, model)
	} else {
		content2, err = aiCallOpenAI(system, user, model)
	}
	if err != nil {
		return nil, err
	}
	questions, perr := parseQuizQuestions(content2)
	if perr != nil {
		log.Errorf("AI quiz parse failed (model=%s). Raw response: %s", model, content2)
	}
	return questions, perr
}

// parseQuizQuestions extracts quiz questions from the model's text response,
// tolerating markdown fences and either a bare array or a {"questions":[...]}
// object. Malformed questions (blank text or fewer than two choices) are
// dropped, and an out-of-range answer index is clamped to the first choice.
func parseQuizQuestions(content string) ([]QuizQuestion, error) {
	js := extractJSON(content)
	var arr []QuizQuestion
	if err := json.Unmarshal([]byte(js), &arr); err != nil || len(arr) == 0 {
		var wrap struct {
			Questions []QuizQuestion `json:"questions"`
		}
		if err := json.Unmarshal([]byte(js), &wrap); err == nil {
			arr = wrap.Questions
		}
	}
	out := make([]QuizQuestion, 0, len(arr))
	for _, q := range arr {
		if strings.TrimSpace(q.Q) == "" || len(q.Choices) < 2 {
			continue
		}
		if q.Answer < 0 || q.Answer >= len(q.Choices) {
			q.Answer = 0
		}
		out = append(out, q)
	}
	if len(out) == 0 {
		return nil, errors.New("could not parse the AI response as quiz questions")
	}
	return out, nil
}

// aiCallOpenAI calls an OpenAI-compatible chat-completions endpoint.
func aiCallOpenAI(system, user, model string) (string, error) {
	payload := map[string]interface{}{
		"model": model,
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
func aiCallAnthropic(system, user, model string) (string, error) {
	payload := map[string]interface{}{
		"model":      model,
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
