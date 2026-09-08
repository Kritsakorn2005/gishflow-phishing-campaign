package models

import (
	"testing"
	"time"
)

// TestComputeTiming checks per-recipient deltas and the aggregate medians.
func TestComputeTiming(t *testing.T) {
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	results := []Result{
		{Status: EventClicked, SendDate: base, BaseRecipient: BaseRecipient{Email: "a@x.com", FirstName: "A"}},
		{Status: EventDataSubmit, SendDate: base, BaseRecipient: BaseRecipient{Email: "b@x.com"}},
		{Status: EventSent, SendDate: base, BaseRecipient: BaseRecipient{Email: "c@x.com"}}, // no interaction
	}
	events := []Event{
		{Email: "a@x.com", Message: EventOpened, Time: base.Add(60 * time.Second)},
		{Email: "a@x.com", Message: EventClicked, Time: base.Add(120 * time.Second)},
		{Email: "b@x.com", Message: EventOpened, Time: base.Add(30 * time.Second)},
		{Email: "b@x.com", Message: EventOpened, Time: base.Add(45 * time.Second)}, // reopen
		{Email: "b@x.com", Message: EventClicked, Time: base.Add(300 * time.Second)},
		{Email: "b@x.com", Message: EventDataSubmit, Time: base.Add(360 * time.Second)},
	}
	ct := computeTiming(results, events)

	if len(ct.Recipients) != 3 {
		t.Fatalf("expected 3 recipients, got %d", len(ct.Recipients))
	}
	// clicks: a=120, b=300 -> median 210, fastest 120
	if ct.FastestClick == nil || *ct.FastestClick != 120 {
		t.Errorf("fastest click: %v", ct.FastestClick)
	}
	if ct.MedianTimeToClick == nil || *ct.MedianTimeToClick != 210 {
		t.Errorf("median click: %v", ct.MedianTimeToClick)
	}
	// only b submitted -> median 360
	if ct.MedianTimeToSubmit == nil || *ct.MedianTimeToSubmit != 360 {
		t.Errorf("median submit: %v", ct.MedianTimeToSubmit)
	}
	// b opened twice (earliest 30s wins for time-to-open)
	for _, r := range ct.Recipients {
		if r.Email == "b@x.com" {
			if r.Opens != 2 {
				t.Errorf("b opens: %d", r.Opens)
			}
			if r.TimeToOpen == nil || *r.TimeToOpen != 30 {
				t.Errorf("b time-to-open: %v", r.TimeToOpen)
			}
		}
		if r.Email == "c@x.com" && r.TimeToClick != nil {
			t.Errorf("c should have no click time")
		}
	}
	// click histogram: 120s -> 1-5m bucket, 300s -> 5-30m bucket
	if ct.ClickBuckets[1] != 1 || ct.ClickBuckets[2] != 1 {
		t.Errorf("unexpected click buckets: %v", ct.ClickBuckets)
	}
}
