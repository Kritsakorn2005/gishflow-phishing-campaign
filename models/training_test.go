package models

import "testing"

// TestQuizQuestions verifies parsing of the stored quiz JSON, including the
// empty and malformed cases (which must yield no questions, not a panic).
func TestQuizQuestions(t *testing.T) {
	tp := TrainingPage{Quiz: `[{"q":"Is this phishing?","choices":["Yes","No"],"answer":1}]`}
	qs := tp.Questions()
	if len(qs) != 1 {
		t.Fatalf("expected 1 question, got %d", len(qs))
	}
	if qs[0].Q != "Is this phishing?" || qs[0].Answer != 1 || len(qs[0].Choices) != 2 {
		t.Errorf("question parsed incorrectly: %+v", qs[0])
	}
	if (&TrainingPage{Quiz: ""}).Questions() != nil {
		t.Error("empty quiz should yield no questions")
	}
	if (&TrainingPage{Quiz: "not-json"}).Questions() != nil {
		t.Error("malformed quiz should yield no questions")
	}
}

// TestQuizGrading verifies scoring against the correct answers.
func TestQuizGrading(t *testing.T) {
	tp := TrainingPage{Quiz: `[{"q":"1","choices":["a","b"],"answer":0},{"q":"2","choices":["a","b"],"answer":1}]`}

	cases := []struct {
		name    string
		answers map[string]int
		score   int
	}{
		{"all correct", map[string]int{"0": 0, "1": 1}, 2},
		{"one correct", map[string]int{"0": 1, "1": 1}, 1},
		{"none answered", map[string]int{}, 0},
	}
	for _, c := range cases {
		score, total := tp.GradeQuiz(c.answers)
		if total != 2 {
			t.Errorf("%s: expected total 2, got %d", c.name, total)
		}
		if score != c.score {
			t.Errorf("%s: expected score %d, got %d", c.name, c.score, score)
		}
	}
}
