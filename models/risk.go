package models

import (
	"math"
	"sort"
	"time"
)

// RiskScore represents a single recipient's aggregated susceptibility risk
// across every campaign they have participated in.
type RiskScore struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
	Score     int    `json:"score"`
	Level     string `json:"level"`
	Campaigns int    `json:"campaigns"`
	Submitted int    `json:"submitted"`
	Clicked   int    `json:"clicked"`
	Opened    int    `json:"opened"`
	Reported  int    `json:"reported"`
	Trained   int    `json:"trained"`
}

// GroupRisk represents the rolled-up risk for a group ("department").
type GroupRisk struct {
	Name         string `json:"name"`
	Score        int    `json:"score"`
	Level        string `json:"level"`
	Members      int    `json:"members"`
	Participants int    `json:"participants"`
}

// Risk scoring weights, applied per campaign a recipient took part in.
const (
	riskSubmit         = 40
	riskClick          = 25
	riskOpen           = 10
	riskReport         = -15
	riskTrained        = -10
	riskMaxPerCampaign = 40
)

// riskHalfLifeDays is how long it takes an isolated risky action to lose half of
// its weight in the current risk score. Recent behaviour therefore dominates,
// and — absent any new failures — a recipient's risk decays over time.
const riskHalfLifeDays = 30.0

// recencyWeight returns a 0..1 multiplier that halves every riskHalfLifeDays of
// age, so older results contribute less to the current risk score.
func recencyWeight(t, now time.Time) float64 {
	ageDays := now.Sub(t).Hours() / 24.0
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Pow(0.5, ageDays/riskHalfLifeDays)
}

// resultContribution returns a single result's risk contribution: the weight of
// the worst action the recipient took, mitigated by reporting and training, and
// floored at zero. Shared by the aggregate and trend calculations.
func resultContribution(r Result) int {
	base := 0
	switch r.Status {
	case EventDataSubmit:
		base = riskSubmit
	case EventClicked:
		base = riskClick
	case EventOpened:
		base = riskOpen
	}
	c := base
	if r.Reported {
		c += riskReport
	}
	if r.Trained {
		c += riskTrained
	}
	if c < 0 {
		c = 0
	}
	return c
}

// riskLevel maps a 0-100 score to a Low / Medium / High band.
func riskLevel(score int) string {
	switch {
	case score >= 61:
		return "High"
	case score >= 31:
		return "Medium"
	default:
		return "Low"
	}
}

type riskAgg struct {
	rs  RiskScore
	sum float64
	n   int
}

// GetUserRiskScores computes a 0-100 risk score for every recipient that has
// participated in one of the user's campaigns. Each campaign's contribution
// (worst action, mitigated by reporting/training) is weighted by recency so
// older behaviour fades, then averaged and normalized against the maximum
// possible per-campaign contribution.
func GetUserRiskScores(uid int64) ([]RiskScore, error) {
	results := []Result{}
	err := db.Where("user_id=?", uid).Find(&results).Error
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	byEmail := map[string]*riskAgg{}
	for _, r := range results {
		a, ok := byEmail[r.Email]
		if !ok {
			a = &riskAgg{rs: RiskScore{Email: r.Email}}
			byEmail[r.Email] = a
		}
		// Keep the most recent identifying details.
		a.rs.FirstName = r.FirstName
		a.rs.LastName = r.LastName
		a.rs.Position = r.Position

		// Raw (undecayed) tallies for the informational columns.
		switch r.Status {
		case EventDataSubmit:
			a.rs.Submitted++
		case EventClicked:
			a.rs.Clicked++
		case EventOpened:
			a.rs.Opened++
		}
		if r.Reported {
			a.rs.Reported++
		}
		if r.Trained {
			a.rs.Trained++
		}
		// The score itself decays: an old action counts for less than a recent
		// one, so a recipient's risk falls over time if they stop failing.
		a.sum += float64(resultContribution(r)) * recencyWeight(r.ModifiedDate, now)
		a.n++
		a.rs.Campaigns = a.n
	}
	scores := make([]RiskScore, 0, len(byEmail))
	for _, a := range byEmail {
		avg := 0.0
		if a.n > 0 {
			avg = a.sum / float64(a.n)
		}
		score := int(math.Round(avg / float64(riskMaxPerCampaign) * 100))
		if score > 100 {
			score = 100
		}
		if score < 0 {
			score = 0
		}
		a.rs.Score = score
		a.rs.Level = riskLevel(score)
		scores = append(scores, a.rs)
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
	return scores, nil
}

// RiskTrendPoint is one campaign's risk score, used to plot how susceptibility
// changes over time (a falling line means awareness is improving).
type RiskTrendPoint struct {
	CampaignId   int64     `json:"campaign_id"`
	Name         string    `json:"name"`
	Date         time.Time `json:"date"`
	Score        int       `json:"score"`
	Participants int       `json:"participants"`
}

// GetRiskTrend computes a risk score per campaign (same weighting as the
// aggregate score, scoped to that campaign's results) and returns the points in
// chronological order. Campaigns with no results are omitted.
func GetRiskTrend(uid int64) ([]RiskTrendPoint, error) {
	campaigns := []Campaign{}
	if err := db.Where("user_id=?", uid).Find(&campaigns).Error; err != nil {
		return nil, err
	}
	results := []Result{}
	if err := db.Where("user_id=?", uid).Find(&results).Error; err != nil {
		return nil, err
	}
	type agg struct{ sum, n int }
	byCampaign := map[int64]*agg{}
	for _, r := range results {
		a, ok := byCampaign[r.CampaignId]
		if !ok {
			a = &agg{}
			byCampaign[r.CampaignId] = a
		}
		a.sum += resultContribution(r)
		a.n++
	}
	points := make([]RiskTrendPoint, 0, len(campaigns))
	for _, c := range campaigns {
		a, ok := byCampaign[c.Id]
		if !ok || a.n == 0 {
			continue
		}
		avg := float64(a.sum) / float64(a.n)
		score := int(math.Round(avg / float64(riskMaxPerCampaign) * 100))
		if score > 100 {
			score = 100
		}
		if score < 0 {
			score = 0
		}
		points = append(points, RiskTrendPoint{
			CampaignId:   c.Id,
			Name:         c.Name,
			Date:         c.CreatedDate,
			Score:        score,
			Participants: a.n,
		})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Date.Before(points[j].Date) })
	return points, nil
}

// GetGroupRiskScores rolls per-recipient risk up into a score per group. A
// group's score is the average of its members that have participated in at
// least one campaign; groups with no participants are reported as "No data".
func GetGroupRiskScores(uid int64) ([]GroupRisk, error) {
	userScores, err := GetUserRiskScores(uid)
	if err != nil {
		return nil, err
	}
	scoreByEmail := map[string]RiskScore{}
	for _, s := range userScores {
		scoreByEmail[s.Email] = s
	}
	groups, err := GetGroups(uid)
	if err != nil {
		return nil, err
	}
	out := make([]GroupRisk, 0, len(groups))
	for _, g := range groups {
		gr := GroupRisk{Name: g.Name, Members: len(g.Targets), Level: "No data"}
		sum := 0
		for _, t := range g.Targets {
			if s, ok := scoreByEmail[t.Email]; ok {
				sum += s.Score
				gr.Participants++
			}
		}
		if gr.Participants > 0 {
			gr.Score = int(math.Round(float64(sum) / float64(gr.Participants)))
			gr.Level = riskLevel(gr.Score)
		}
		out = append(out, gr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out, nil
}
