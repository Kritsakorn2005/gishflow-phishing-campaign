package models

import (
	"math"
	"sort"
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
	sum int
	n   int
}

// GetUserRiskScores computes a 0-100 risk score for every recipient that has
// participated in one of the user's campaigns. The score is the average of
// each campaign's contribution (worst action, mitigated by reporting/training),
// normalized against the maximum possible per-campaign contribution.
func GetUserRiskScores(uid int64) ([]RiskScore, error) {
	results := []Result{}
	err := db.Where("user_id=?", uid).Find(&results).Error
	if err != nil {
		return nil, err
	}
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

		base := 0
		switch r.Status {
		case EventDataSubmit:
			base = riskSubmit
			a.rs.Submitted++
		case EventClicked:
			base = riskClick
			a.rs.Clicked++
		case EventOpened:
			base = riskOpen
			a.rs.Opened++
		}
		if r.Reported {
			a.rs.Reported++
		}
		if r.Trained {
			a.rs.Trained++
		}
		contribution := base
		if r.Reported {
			contribution += riskReport
		}
		if r.Trained {
			contribution += riskTrained
		}
		if contribution < 0 {
			contribution = 0
		}
		a.sum += contribution
		a.n++
		a.rs.Campaigns = a.n
	}
	scores := make([]RiskScore, 0, len(byEmail))
	for _, a := range byEmail {
		avg := 0.0
		if a.n > 0 {
			avg = float64(a.sum) / float64(a.n)
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
