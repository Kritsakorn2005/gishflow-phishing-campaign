package models

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// PhishCue is a single observable property of a phishing message that helps a
// recipient detect it. The NIST Phish Scale counts these: the more cues, the
// easier the message is to spot.
type PhishCue struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Detail   string `json:"detail"`
}

// PhishScaleResult is the outcome of scoring a message with the NIST Phish
// Scale: the detected cues and premise alignment combined into a detection
// difficulty. "Very difficult to detect" is the most dangerous (hardest for a
// recipient to notice); "Least difficult" is the easiest to spot.
type PhishScaleResult struct {
	Cues          []PhishCue `json:"cues"`
	CueCount      int        `json:"cue_count"`
	CueBucket     string     `json:"cue_bucket"`
	Premise       int        `json:"premise"`
	PremiseLevel  string     `json:"premise_level"`
	Difficulty    string     `json:"difficulty"`
	DifficultyKey string     `json:"difficulty_key"`
}

// Cue keyword lists (lowercased), covering English and Thai. These are the
// language/content and tactic cues the detector can reliably match in template
// text; technical cues (sender, links) are detected structurally below.
var (
	psUrgency = []string{"urgent", "immediately", "as soon as possible", "right away",
		"within 24", "within 48", "expire", "expiring", "suspend", "suspended",
		"deactivat", "verify now", "act now", "final notice", "last warning",
		"ด่วน", "เร่งด่วน", "ทันที", "ภายใน 24", "ภายใน 48", "ระงับ", "หมดอายุ", "ยืนยันทันที"}
	psGreeting = []string{"dear user", "dear customer", "dear member", "dear valued",
		"valued customer", "dear sir", "dear madam", "dear account holder",
		"เรียนผู้ใช้", "เรียนลูกค้า", "เรียนสมาชิก", "เรียนท่าน"}
	psCredential = []string{"password", "username", "log in", "login", "sign in",
		"verify your account", "confirm your identity", "credit card", "card number",
		"cvv", "otp", "ssn", "social security", "รหัสผ่าน", "เข้าสู่ระบบ", "ยืนยันตัวตน",
		"เลขบัตร", "บัตรเครดิต", "รหัส otp"}
	psReward = []string{"congratulations", "you have won", "you won", "winner", "prize",
		"reward", "gift card", "free gift", "claim your", "คุณได้รับ", "รางวัล", "ของขวัญ", "ยินดีด้วย"}
	psThreat = []string{"will be closed", "will be deleted", "will be suspended",
		"legal action", "penalty", "police", "lawsuit", "ดำเนินคดี", "ปิดบัญชี", "ลบบัญชี", "ค่าปรับ", "ตำรวจ"}
	psSuspicious = []string{"kindly", "attention required", "important notice regarding your"}
	psFreeMail   = []string{"gmail.com", "googlemail.com", "hotmail.com", "outlook.com",
		"yahoo.com", "live.com", "aol.com", "icloud.com", "protonmail.com", "mail.com"}
	psShortener = []string{"bit.ly", "tinyurl.com", "goo.gl", "t.co", "ow.ly", "is.gd", "buff.ly", "rebrand.ly"}

	psTagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	psIPRe     = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	psDomainRe = regexp.MustCompile(`([a-z0-9-]+\.)+[a-z]{2,}`)
)

func psContainsAny(haystack string, needles []string) (bool, string) {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true, n
		}
	}
	return false, ""
}

func psStripHTML(h string) string {
	if strings.TrimSpace(h) == "" {
		return ""
	}
	if doc, err := goquery.NewDocumentFromReader(strings.NewReader(h)); err == nil {
		return doc.Text()
	}
	return psTagRe.ReplaceAllString(h, " ")
}

func psEmailDomain(addr string) string {
	at := strings.LastIndex(addr, "@")
	if at < 0 || at == len(addr)-1 {
		return ""
	}
	dom := strings.ToLower(strings.Trim(addr[at+1:], " >"))
	return dom
}

// psLinkCues inspects anchors for raw-IP hosts, URL shorteners, and text that
// claims a different domain than the link's destination.
func psLinkCues(htmlBody string) []PhishCue {
	var cues []PhishCue
	if strings.TrimSpace(htmlBody) == "" {
		return cues
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return cues
	}
	seen := map[string]bool{}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		href = strings.TrimSpace(href)
		// Skip empty and Gophish template variables (e.g. {{.URL}}).
		if href == "" || strings.HasPrefix(href, "{{") {
			return
		}
		u, err := url.Parse(href)
		if err != nil {
			return
		}
		host := strings.ToLower(u.Hostname())
		if host == "" {
			return
		}
		if psIPRe.MatchString(host) && !seen["ip"] {
			seen["ip"] = true
			cues = append(cues, PhishCue{"Technical indicator", "Link to a raw IP address", host})
		}
		for _, sh := range psShortener {
			if host == sh && !seen["short"] {
				seen["short"] = true
				cues = append(cues, PhishCue{"Technical indicator", "Link uses a URL shortener", host})
			}
		}
		anchor := strings.ToLower(strings.TrimSpace(s.Text()))
		if ad := psDomainRe.FindString(anchor); ad != "" && ad != host && !seen["mismatch"] {
			seen["mismatch"] = true
			cues = append(cues, PhishCue{"Technical indicator", "Link text doesn't match its destination", anchor + " -> " + host})
		}
	})
	return cues
}

// ScorePhishScale scores a message with the NIST Phish Scale. premise is the
// premise-alignment rating (1-5): how well the message fits the target's work
// context. More cues make a message easier to detect; higher premise alignment
// makes it harder.
func ScorePhishScale(subject, htmlBody, text, fromAddress string, premise int) PhishScaleResult {
	if premise < 1 {
		premise = 1
	}
	if premise > 5 {
		premise = 5
	}
	res := PhishScaleResult{Premise: premise}

	raw := subject + "\n" + htmlBody + "\n" + text
	corpus := strings.ToLower(subject + "\n" + text + "\n" + psStripHTML(htmlBody))
	personalized := strings.Contains(raw, "{{.FirstName}}") || strings.Contains(raw, "{{.LastName}}")

	add := func(cat, name, detail string) {
		res.Cues = append(res.Cues, PhishCue{Category: cat, Name: name, Detail: detail})
	}

	if ok, kw := psContainsAny(corpus, psUrgency); ok {
		add("Language & content", "Urgency or pressure", "contains \""+kw+"\"")
	}
	if ok, kw := psContainsAny(corpus, psGreeting); ok && !personalized {
		add("Language & content", "Generic, impersonal greeting", "uses \""+kw+"\" with no personalization")
	}
	if ok, kw := psContainsAny(corpus, psCredential); ok {
		add("Language & content", "Requests credentials or sensitive info", "mentions \""+kw+"\"")
	}
	if ok, kw := psContainsAny(corpus, psReward); ok {
		add("Common tactic", "Reward / too good to be true", "mentions \""+kw+"\"")
	}
	if ok, kw := psContainsAny(corpus, psThreat); ok {
		add("Common tactic", "Threat or consequence", "mentions \""+kw+"\"")
	}
	if ok, kw := psContainsAny(corpus, psSuspicious); ok {
		add("Error & style", "Suspicious phrasing", "contains \""+kw+"\"")
	}
	if dom := psEmailDomain(fromAddress); dom != "" {
		for _, fm := range psFreeMail {
			if dom == fm {
				add("Technical indicator", "Sender uses a free email provider", dom)
				break
			}
		}
	}
	res.Cues = append(res.Cues, psLinkCues(htmlBody)...)

	res.CueCount = len(res.Cues)
	res.CueBucket = psCueBucket(res.CueCount)
	res.PremiseLevel = psPremiseLevel(premise)
	res.Difficulty, res.DifficultyKey = psDifficulty(res.CueBucket, res.PremiseLevel)
	return res
}

// psCueBucket categorizes the cue count. Operationalized for the detectable cue
// set (NIST groups cues as few/medium/many).
func psCueBucket(n int) string {
	switch {
	case n <= 2:
		return "Few"
	case n <= 5:
		return "Some"
	default:
		return "Many"
	}
}

func psPremiseLevel(p int) string {
	switch {
	case p <= 2:
		return "Low"
	case p == 3:
		return "Medium"
	default:
		return "High"
	}
}

// psDifficulty combines cue count and premise alignment into a detection
// difficulty, following the NIST Phish Scale: few cues + high alignment is the
// hardest to detect; many cues + low alignment is the easiest.
func psDifficulty(cueBucket, premiseLevel string) (string, string) {
	matrix := map[string]map[string][2]string{
		"Few": {
			"Low":    {"Moderate difficulty to detect", "moderate"},
			"Medium": {"Very difficult to detect", "very"},
			"High":   {"Very difficult to detect", "very"},
		},
		"Some": {
			"Low":    {"Least difficult to detect", "least"},
			"Medium": {"Moderate difficulty to detect", "moderate"},
			"High":   {"Very difficult to detect", "very"},
		},
		"Many": {
			"Low":    {"Least difficult to detect", "least"},
			"Medium": {"Least difficult to detect", "least"},
			"High":   {"Moderate difficulty to detect", "moderate"},
		},
	}
	d := matrix[cueBucket][premiseLevel]
	return d[0], d[1]
}
