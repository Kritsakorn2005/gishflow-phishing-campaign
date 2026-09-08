package models

import (
	"sort"
	"time"
)

// RecipientTiming holds how long a single recipient took to reach each stage of
// the funnel, measured in seconds from when their message was sent. A nil
// pointer means that stage was never reached.
type RecipientTiming struct {
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	TimeToOpen   *int64 `json:"time_to_open"`
	TimeToClick  *int64 `json:"time_to_click"`
	TimeToSubmit *int64 `json:"time_to_submit"`
	Opens        int    `json:"opens"`
}

// CampaignTiming aggregates engagement timing across a campaign's recipients.
type CampaignTiming struct {
	Recipients         []RecipientTiming `json:"recipients"`
	MedianTimeToOpen   *int64            `json:"median_time_to_open"`
	MedianTimeToClick  *int64            `json:"median_time_to_click"`
	MedianTimeToSubmit *int64            `json:"median_time_to_submit"`
	FastestClick       *int64            `json:"fastest_click"`
	// ClickBuckets is a histogram of time-to-click over BucketLabels.
	ClickBuckets []int    `json:"click_buckets"`
	BucketLabels []string `json:"bucket_labels"`
}

// median returns the median of the values, or nil when there are none.
func median(v []int64) *int64 {
	if len(v) == 0 {
		return nil
	}
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	n := len(s)
	var m int64
	if n%2 == 1 {
		m = s[n/2]
	} else {
		m = (s[n/2-1] + s[n/2]) / 2
	}
	return &m
}

// minVal returns the smallest value, or nil when there are none.
func minVal(v []int64) *int64 {
	if len(v) == 0 {
		return nil
	}
	m := v[0]
	for _, x := range v {
		if x < m {
			m = x
		}
	}
	return &m
}

// bucketize buckets time-to-click seconds into <1m / 1-5m / 5-30m / 30-60m / >60m.
func bucketize(v []int64) []int {
	b := make([]int, 5)
	for _, s := range v {
		switch {
		case s < 60:
			b[0]++
		case s < 300:
			b[1]++
		case s < 1800:
			b[2]++
		case s < 3600:
			b[3]++
		default:
			b[4]++
		}
	}
	return b
}

// computeTiming derives per-recipient and aggregate engagement timing from a
// campaign's results (for the send-time baseline and identity) and events (for
// when each stage happened). It is a pure function so it can be unit-tested
// without a database.
func computeTiming(results []Result, events []Event) CampaignTiming {
	sendDate := map[string]time.Time{}
	for _, r := range results {
		if !r.SendDate.IsZero() {
			sendDate[r.Email] = r.SendDate
		}
	}

	type stamps struct {
		open, click, submit *time.Time
		opens               int
	}
	byEmail := map[string]*stamps{}
	get := func(email string) *stamps {
		s, ok := byEmail[email]
		if !ok {
			s = &stamps{}
			byEmail[email] = s
		}
		return s
	}
	earliest := func(cur **time.Time, t time.Time) {
		if *cur == nil || t.Before(**cur) {
			tt := t
			*cur = &tt
		}
	}
	for _, e := range events {
		s := get(e.Email)
		switch e.Message {
		case EventOpened:
			s.opens++
			earliest(&s.open, e.Time)
		case EventClicked:
			earliest(&s.click, e.Time)
		case EventDataSubmit:
			earliest(&s.submit, e.Time)
		}
	}

	delta := func(base time.Time, ev *time.Time) *int64 {
		if ev == nil {
			return nil
		}
		d := int64(ev.Sub(base).Seconds())
		if d < 0 {
			d = 0
		}
		return &d
	}

	ct := CampaignTiming{
		BucketLabels: []string{"<1m", "1-5m", "5-30m", "30-60m", ">60m"},
	}
	var opens, clicks, submits []int64
	for _, r := range results {
		base, ok := sendDate[r.Email]
		if !ok {
			continue
		}
		s := get(r.Email)
		rt := RecipientTiming{
			Email:        r.Email,
			FirstName:    r.FirstName,
			LastName:     r.LastName,
			Opens:        s.opens,
			TimeToOpen:   delta(base, s.open),
			TimeToClick:  delta(base, s.click),
			TimeToSubmit: delta(base, s.submit),
		}
		ct.Recipients = append(ct.Recipients, rt)
		if rt.TimeToOpen != nil {
			opens = append(opens, *rt.TimeToOpen)
		}
		if rt.TimeToClick != nil {
			clicks = append(clicks, *rt.TimeToClick)
		}
		if rt.TimeToSubmit != nil {
			submits = append(submits, *rt.TimeToSubmit)
		}
	}
	ct.MedianTimeToOpen = median(opens)
	ct.MedianTimeToClick = median(clicks)
	ct.MedianTimeToSubmit = median(submits)
	ct.FastestClick = minVal(clicks)
	ct.ClickBuckets = bucketize(clicks)
	return ct
}

// GetCampaignTiming loads a campaign's results and events and returns its
// engagement timing.
func GetCampaignTiming(cid int64, uid int64) (CampaignTiming, error) {
	results := []Result{}
	if err := db.Where("campaign_id=? and user_id=?", cid, uid).Find(&results).Error; err != nil {
		return CampaignTiming{}, err
	}
	events := []Event{}
	if err := db.Where("campaign_id=?", cid).Find(&events).Error; err != nil {
		return CampaignTiming{}, err
	}
	return computeTiming(results, events), nil
}
