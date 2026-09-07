package models

import (
	"math"
	"testing"
	"time"
)

// TestRecencyWeight verifies the risk decay halves every half-life and clamps
// future timestamps to full weight.
func TestRecencyWeight(t *testing.T) {
	now := time.Now().UTC()
	halfLife := time.Duration(riskHalfLifeDays*24) * time.Hour
	cases := []struct {
		name string
		age  time.Duration
		want float64
	}{
		{"now", 0, 1.0},
		{"one half-life", halfLife, 0.5},
		{"two half-lives", 2 * halfLife, 0.25},
	}
	for _, c := range cases {
		got := recencyWeight(now.Add(-c.age), now)
		if math.Abs(got-c.want) > 0.001 {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if got := recencyWeight(now.Add(24*time.Hour), now); math.Abs(got-1.0) > 0.001 {
		t.Errorf("future timestamp: got %v, want 1.0", got)
	}
}

// TestResultContribution verifies the per-result risk weighting: worst action,
// mitigated by reporting/training, floored at zero.
func TestResultContribution(t *testing.T) {
	if got := resultContribution(Result{Status: EventDataSubmit}); got != riskSubmit {
		t.Errorf("submit: got %d, want %d", got, riskSubmit)
	}
	mitigated := Result{Status: EventDataSubmit, Reported: true, Trained: true}
	if got := resultContribution(mitigated); got != riskSubmit+riskReport+riskTrained {
		t.Errorf("mitigated submit: got %d, want %d", got, riskSubmit+riskReport+riskTrained)
	}
	floored := Result{Status: EventOpened, Reported: true, Trained: true}
	if got := resultContribution(floored); got != 0 {
		t.Errorf("floored contribution: got %d, want 0", got)
	}
}
