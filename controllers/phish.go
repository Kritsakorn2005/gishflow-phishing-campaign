package controllers

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/controllers/api"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jordan-wright/unindexed"
	qrcode "github.com/skip2/go-qrcode"
)

// ErrInvalidRequest is thrown when a request with an invalid structure is
// received
var ErrInvalidRequest = errors.New("Invalid request")

// ErrCampaignComplete is thrown when an event is received for a campaign that
// has already been marked as complete.
var ErrCampaignComplete = errors.New("Event received on completed campaign")

// TransparencyResponse is the JSON response provided when a third-party
// makes a request to the transparency handler.
type TransparencyResponse struct {
	Server         string    `json:"server"`
	ContactAddress string    `json:"contact_address"`
	SendDate       time.Time `json:"send_date"`
}

// TransparencySuffix (when appended to a valid result ID), will cause Gophish
// to return a transparency response.
const TransparencySuffix = "+"

// PhishingServerOption is a functional option that is used to configure the
// the phishing server
type PhishingServerOption func(*PhishingServer)

// PhishingServer is an HTTP server that implements the campaign event
// handlers, such as email open tracking, click tracking, and more.
type PhishingServer struct {
	server         *http.Server
	config         config.PhishServer
	contactAddress string
}

// NewPhishingServer returns a new instance of the phishing server with
// provided options applied.
func NewPhishingServer(config config.PhishServer, options ...PhishingServerOption) *PhishingServer {
	defaultServer := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         config.ListenURL,
	}
	ps := &PhishingServer{
		server: defaultServer,
		config: config,
	}
	for _, opt := range options {
		opt(ps)
	}
	ps.registerRoutes()
	return ps
}

// WithContactAddress sets the contact address used by the transparency
// handlers
func WithContactAddress(addr string) PhishingServerOption {
	return func(ps *PhishingServer) {
		ps.contactAddress = addr
	}
}

// Start launches the phishing server, listening on the configured address.
func (ps *PhishingServer) Start() {
	if ps.config.UseTLS {
		// Only support TLS 1.2 and above - ref #1691, #1689
		ps.server.TLSConfig = defaultTLSConfig
		err := util.CheckAndCreateSSL(ps.config.CertPath, ps.config.KeyPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Starting phishing server at https://%s", ps.config.ListenURL)
		log.Fatal(ps.server.ListenAndServeTLS(ps.config.CertPath, ps.config.KeyPath))
	}
	// If TLS isn't configured, just listen on HTTP
	log.Infof("Starting phishing server at http://%s", ps.config.ListenURL)
	log.Fatal(ps.server.ListenAndServe())
}

// Shutdown attempts to gracefully shutdown the server.
func (ps *PhishingServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return ps.server.Shutdown(ctx)
}

// CreatePhishingRouter creates the router that handles phishing connections.
func (ps *PhishingServer) registerRoutes() {
	router := mux.NewRouter()
	fileServer := http.FileServer(unindexed.Dir("./static/endpoint/"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))
	router.HandleFunc("/track", ps.TrackHandler)
	router.HandleFunc("/robots.txt", ps.RobotsHandler)
	// GishFlow: LINE OA webhook — collects recipients' userIds automatically.
	router.HandleFunc("/line/webhook", ps.LineWebhookHandler)
	// GishFlow: hosted QR image, so QR codes in emails render in clients (e.g.
	// Gmail) that strip inline data: images.
	router.HandleFunc("/qr", ps.QRImageHandler)
	router.HandleFunc("/{path:.*}/track", ps.TrackHandler)
	router.HandleFunc("/{path:.*}/report", ps.ReportHandler)
	router.HandleFunc("/report", ps.ReportHandler)
	router.HandleFunc("/{path:.*}", ps.PhishHandler)

	// Setup GZIP compression
	gzipWrapper, _ := gziphandler.NewGzipLevelHandler(gzip.BestCompression)
	phishHandler := gzipWrapper(router)

	// Respect X-Forwarded-For and X-Real-IP headers in case we're behind a
	// reverse proxy.
	phishHandler = handlers.ProxyHeaders(phishHandler)

	// Setup logging
	phishHandler = handlers.CombinedLoggingHandler(log.Writer(), phishHandler)
	ps.server.Handler = phishHandler
}

// QRImageHandler renders a QR code PNG for the URL passed in the "u" query
// parameter. It is public and does no tracking, so QR codes referenced from a
// phishing email render even in clients that strip inline data: images (e.g.
// Gmail). Only the image is returned; the scan itself is tracked when the
// encoded URL is later visited.
func (ps *PhishingServer) QRImageHandler(w http.ResponseWriter, r *http.Request) {
	text := r.URL.Query().Get("u")
	if _, err := url.ParseRequestURI(text); err != nil {
		http.Error(w, "Missing or invalid 'u' parameter", http.StatusBadRequest)
		return
	}
	png, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(png)
}

// TrackHandler tracks emails as they are opened, updating the status for the given Result
func (ps *PhishingServer) TrackHandler(w http.ResponseWriter, r *http.Request) {
	r, err := setupContext(r)
	if err != nil {
		// Log the error if it wasn't something we can safely ignore
		if err != ErrInvalidRequest && err != ErrCampaignComplete {
			log.Error(err)
		}
		http.NotFound(w, r)
		return
	}
	// Check for a preview
	if _, ok := ctx.Get(r, "result").(models.EmailRequest); ok {
		http.ServeFile(w, r, "static/images/pixel.png")
		return
	}
	rs := ctx.Get(r, "result").(models.Result)
	rid := ctx.Get(r, "rid").(string)
	d := ctx.Get(r, "details").(models.EventDetails)

	// Check for a transparency request
	if strings.HasSuffix(rid, TransparencySuffix) {
		ps.TransparencyHandler(w, r)
		return
	}

	err = rs.HandleEmailOpened(d)
	if err != nil {
		log.Error(err)
	}
	http.ServeFile(w, r, "static/images/pixel.png")
}

// ReportHandler tracks emails as they are reported, updating the status for the given Result
func (ps *PhishingServer) ReportHandler(w http.ResponseWriter, r *http.Request) {
	r, err := setupContext(r)
	w.Header().Set("Access-Control-Allow-Origin", "*") // To allow Chrome extensions (or other pages) to report a campaign without violating CORS
	if err != nil {
		// Log the error if it wasn't something we can safely ignore
		if err != ErrInvalidRequest && err != ErrCampaignComplete {
			log.Error(err)
		}
		http.NotFound(w, r)
		return
	}
	// Check for a preview
	if _, ok := ctx.Get(r, "result").(models.EmailRequest); ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rs := ctx.Get(r, "result").(models.Result)
	rid := ctx.Get(r, "rid").(string)
	d := ctx.Get(r, "details").(models.EventDetails)

	// Check for a transparency request
	if strings.HasSuffix(rid, TransparencySuffix) {
		ps.TransparencyHandler(w, r)
		return
	}

	err = rs.HandleEmailReport(d)
	if err != nil {
		log.Error(err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// PhishHandler handles incoming client connections and registers the associated actions performed
// (such as clicked link, etc.)
func (ps *PhishingServer) PhishHandler(w http.ResponseWriter, r *http.Request) {
	r, err := setupContext(r)
	if err != nil {
		// Log the error if it wasn't something we can safely ignore
		if err != ErrInvalidRequest && err != ErrCampaignComplete {
			log.Error(err)
		}
		http.NotFound(w, r)
		return
	}
	w.Header().Set("X-Server", config.ServerName) // Useful for checking if this is a GoPhish server (e.g. for campaign reporting plugins)
	var ptx models.PhishingTemplateContext
	// Check for a preview
	if preview, ok := ctx.Get(r, "result").(models.EmailRequest); ok {
		ptx, err = models.NewPhishingTemplateContext(&preview, preview.BaseRecipient, preview.RId)
		if err != nil {
			log.Error(err)
			http.NotFound(w, r)
			return
		}
		p, err := models.GetPage(preview.PageId, preview.UserId)
		if err != nil {
			log.Error(err)
			http.NotFound(w, r)
			return
		}
		renderPhishResponse(w, r, ptx, p)
		return
	}
	rs := ctx.Get(r, "result").(models.Result)
	rid := ctx.Get(r, "rid").(string)
	c := ctx.Get(r, "campaign").(models.Campaign)
	d := ctx.Get(r, "details").(models.EventDetails)

	// Check for a transparency request
	if strings.HasSuffix(rid, TransparencySuffix) {
		ps.TransparencyHandler(w, r)
		return
	}

	// GishFlow: a submission of the training-page quiz. Grade it, record the
	// completion + score, and show the results page — rather than letting it
	// fall through and be recorded as another landing-page form submit.
	if r.Form.Get("__gfquiz") == "1" && c.TrainingId != 0 {
		tp, terr := models.GetTrainingPage(c.TrainingId, c.UserId)
		if terr == nil && len(tp.Questions()) > 0 {
			answers := parseQuizAnswers(r)
			score, total := tp.GradeQuiz(answers)
			// Record the score in the event details so it's visible on the
			// recipient's timeline in Campaign Results.
			d.Payload.Set("quiz_score", strconv.Itoa(score)+"/"+strconv.Itoa(total))
			if terr = rs.HandleTrainingCompleted(d, score, total); terr != nil {
				log.Error(terr)
			}
			renderQuizResult(w, tp, answers, score, total)
			return
		}
	}

	p, err := models.GetPage(c.PageId, c.UserId)
	if err != nil {
		log.Error(err)
		http.NotFound(w, r)
		return
	}
	switch {
	case r.Method == "GET":
		err = rs.HandleClickedLink(d)
		if err != nil {
			log.Error(err)
		}
		// GishFlow: if the link was reached by scanning a QR code, record a
		// separate "QR Scanned" event so quishing can be told apart from clicks.
		if r.Form.Get("qr") == "1" {
			if qerr := rs.HandleQRScanned(d); qerr != nil {
				log.Error(qerr)
			}
		}
	case r.Method == "POST":
		err = rs.HandleFormSubmit(d)
		if err != nil {
			log.Error(err)
		}
	}
	ptx, err = models.NewPhishingTemplateContext(&c, rs.BaseRecipient, rs.RId)
	if err != nil {
		log.Error(err)
		http.NotFound(w, r)
		return
	}
	// GishFlow teachable moment: once the recipient has interacted, show the
	// campaign's training page instead of the landing page / redirect.
	//   - POST (submitted data): always show it (they fell for it)
	//   - GET (clicked) on a non-capturing landing page: show it too
	if c.TrainingId != 0 {
		showTraining := r.Method == "POST" || (r.Method == "GET" && !p.CaptureCredentials)
		if showTraining {
			tp, terr := models.GetTrainingPage(c.TrainingId, c.UserId)
			if terr == nil && tp.HTML != "" {
				if terr = rs.HandleTrainingViewed(d); terr != nil {
					log.Error(terr)
				}
				body, terr := models.ExecuteTemplate(tp.HTML, ptx)
				if terr != nil {
					log.Error(terr)
					http.NotFound(w, r)
					return
				}
				// If the training page has a quiz, append the interactive form
				// so the recipient can complete the teachable moment.
				if qs := tp.Questions(); len(qs) > 0 {
					body += buildQuizForm(qs, rs.RId)
				}
				// Set the content type explicitly: training content often starts
				// with non-markup text (e.g. Thai), which would otherwise be
				// sniffed as text/plain and shown as raw HTML source.
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(body))
				return
			}
		}
	}
	renderPhishResponse(w, r, ptx, p)
}

// renderPhishResponse handles rendering the correct response to the phishing
// connection. This usually involves writing out the page HTML or redirecting
// the user to the correct URL.
func renderPhishResponse(w http.ResponseWriter, r *http.Request, ptx models.PhishingTemplateContext, p models.Page) {
	// If the request was a form submit and a redirect URL was specified, we
	// should send the user to that URL
	if r.Method == "POST" {
		if p.RedirectURL != "" {
			redirectURL, err := models.ExecuteTemplate(p.RedirectURL, ptx)
			if err != nil {
				log.Error(err)
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, redirectURL, http.StatusFound)
			return
		}
	}
	// Otherwise, we just need to write out the templated HTML
	html, err := models.ExecuteTemplate(p.HTML, ptx)
	if err != nil {
		log.Error(err)
		http.NotFound(w, r)
		return
	}
	w.Write([]byte(html))
}

// RobotsHandler prevents search engines, etc. from indexing phishing materials
func (ps *PhishingServer) RobotsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "User-agent: *\nDisallow: /")
}

// lineWebhookBody is the subset of the LINE webhook payload we care about: the
// list of events, each carrying the source userId of whoever interacted.
type lineWebhookBody struct {
	Events []struct {
		Type   string `json:"type"`
		Source struct {
			Type   string `json:"type"`
			UserID string `json:"userId"`
		} `json:"source"`
	} `json:"events"`
}

// verifyLineSignature checks the X-Line-Signature header, which is the
// base64-encoded HMAC-SHA256 of the raw request body keyed by the channel
// secret. It's a constant-time comparison.
func verifyLineSignature(secret string, body []byte, signature string) bool {
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// LineWebhookHandler receives events from a LINE Official Account (follow,
// message, etc.) and records the sender's userId as a LineContact, so
// recipients can be collected automatically instead of looked up by hand.
// It always answers 200 so LINE considers the webhook healthy.
func (ps *PhishingServer) LineWebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Look up configured LINE profiles: their channel secret (Username) verifies
	// the signature, and their access token (Password) fetches display names.
	profiles, _ := models.GetLineProfiles()
	token := ""
	hasSecret := false
	verified := false
	sig := r.Header.Get("X-Line-Signature")
	for _, p := range profiles {
		if token == "" && p.Password != "" {
			token = p.Password // default token (first profile), for display names
		}
		if p.Username != "" {
			hasSecret = true
			if verifyLineSignature(p.Username, body, sig) {
				verified = true
				token = p.Password // prefer the token of the matching OA
				break
			}
		}
	}
	// If any LINE profile has a channel secret configured, require a valid
	// signature. If none do, accept unverified (keeps setup simple).
	if hasSecret && !verified {
		log.Error("LINE webhook: invalid signature")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var wb lineWebhookBody
	if err := json.Unmarshal(body, &wb); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusOK)
		return
	}
	for _, e := range wb.Events {
		if e.Source.UserID == "" {
			continue
		}
		name := models.GetLineDisplayName(token, e.Source.UserID)
		if err := models.SaveLineContact(e.Source.UserID, name); err != nil {
			log.Error(err)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// quizStyle is the shared, self-contained styling for the quiz form and the
// results page. These are served from the phishing domain, so they can't rely
// on the admin theme stylesheet and must inline everything.
const quizStyle = `<style>
.gf-quiz{max-width:640px;margin:24px auto;padding:28px 30px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  background:#fff;border:1px solid #e5e5ea;border-radius:16px;box-shadow:0 8px 30px rgba(0,0,0,.06);color:#1d1d1f}
.gf-quiz h2{margin:0 0 6px;font-size:22px;font-weight:600}
.gf-quiz p.gf-lead{margin:0 0 20px;color:#6e6e73;font-size:14px}
.gf-q{margin:0 0 20px;padding:0 0 18px;border-bottom:1px solid #f0f0f2}
.gf-q:last-of-type{border-bottom:0}
.gf-q .gf-qt{font-weight:600;margin:0 0 12px;font-size:15px}
.gf-opt{display:block;padding:10px 14px;margin:6px 0;border:1px solid #e5e5ea;border-radius:10px;cursor:pointer;font-size:14px;transition:.15s}
.gf-opt:hover{background:#f5f5f7}
.gf-opt input{margin-right:10px}
.gf-opt.correct{border-color:#30d158;background:#eafbef}
.gf-opt.wrong{border-color:#ff3b30;background:#ffeeed}
.gf-btn{display:inline-block;background:#0071e3;color:#fff;border:0;border-radius:980px;padding:11px 26px;font-size:15px;font-weight:500;cursor:pointer}
.gf-btn:hover{background:#0077ed}
.gf-score{font-size:34px;font-weight:700;margin:4px 0 2px}
.gf-badge{display:inline-block;padding:4px 14px;border-radius:980px;font-size:13px;font-weight:600;margin-bottom:14px}
.gf-pass{background:#eafbef;color:#248a3d}
.gf-fail{background:#fff4e5;color:#c05600}
</style>`

// parseQuizAnswers pulls the recipient's quiz answers out of the request form.
// Inputs are named "gfq_<index>" with the chosen choice index as the value.
func parseQuizAnswers(r *http.Request) map[string]int {
	answers := map[string]int{}
	for k, v := range r.Form {
		if strings.HasPrefix(k, "gfq_") && len(v) > 0 {
			if n, err := strconv.Atoi(v[0]); err == nil {
				answers[strings.TrimPrefix(k, "gfq_")] = n
			}
		}
	}
	return answers
}

// buildQuizForm renders the interactive quiz that is appended below the
// training content. It posts back to the same URL with a "__gfquiz" marker.
func buildQuizForm(qs []models.QuizQuestion, rid string) string {
	var b strings.Builder
	b.WriteString(quizStyle)
	b.WriteString(`<div class="gf-quiz"><h2>Quick knowledge check</h2>`)
	b.WriteString(`<p class="gf-lead">Answer these questions to complete your training.</p>`)
	b.WriteString(`<form method="post" action="">`)
	b.WriteString(`<input type="hidden" name="__gfquiz" value="1">`)
	b.WriteString(`<input type="hidden" name="` + models.RecipientParameter + `" value="` + html.EscapeString(rid) + `">`)
	for i, q := range qs {
		b.WriteString(`<div class="gf-q"><div class="gf-qt">` + strconv.Itoa(i+1) + `. ` + html.EscapeString(q.Q) + `</div>`)
		for j, choice := range q.Choices {
			name := "gfq_" + strconv.Itoa(i)
			b.WriteString(`<label class="gf-opt"><input type="radio" name="` + name + `" value="` + strconv.Itoa(j) + `">` + html.EscapeString(choice) + `</label>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`<button type="submit" class="gf-btn">Submit answers</button></form></div>`)
	return b.String()
}

// renderQuizResult writes the graded results page, showing the recipient's
// score and highlighting the correct answer for each question.
func renderQuizResult(w http.ResponseWriter, tp models.TrainingPage, answers map[string]int, score, total int) {
	qs := tp.Questions()
	passed := total > 0 && score*100/total >= 60
	badgeClass, badgeText := "gf-fail", "Keep practicing"
	if passed {
		badgeClass, badgeText = "gf-pass", "Passed"
	}
	var b strings.Builder
	b.WriteString(quizStyle)
	b.WriteString(`<div class="gf-quiz"><h2>Your results</h2>`)
	b.WriteString(`<span class="gf-badge ` + badgeClass + `">` + badgeText + `</span>`)
	b.WriteString(`<div class="gf-score">` + strconv.Itoa(score) + ` / ` + strconv.Itoa(total) + `</div>`)
	b.WriteString(`<p class="gf-lead">Review the correct answers below.</p>`)
	for i, q := range qs {
		chosen, answered := answers[strconv.Itoa(i)]
		b.WriteString(`<div class="gf-q"><div class="gf-qt">` + strconv.Itoa(i+1) + `. ` + html.EscapeString(q.Q) + `</div>`)
		for j, choice := range q.Choices {
			cls := "gf-opt"
			if j == q.Answer {
				cls += " correct"
			} else if answered && j == chosen {
				cls += " wrong"
			}
			b.WriteString(`<div class="` + cls + `">` + html.EscapeString(choice) + `</div>`)
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(b.String()))
}

// TransparencyHandler returns a TransparencyResponse for the provided result
// and campaign.
func (ps *PhishingServer) TransparencyHandler(w http.ResponseWriter, r *http.Request) {
	rs := ctx.Get(r, "result").(models.Result)
	tr := &TransparencyResponse{
		Server:         config.ServerName,
		SendDate:       rs.SendDate,
		ContactAddress: ps.contactAddress,
	}
	api.JSONResponse(w, tr, http.StatusOK)
}

// setupContext handles some of the administrative work around receiving a new
// request, such as checking the result ID, the campaign, etc.
func setupContext(r *http.Request) (*http.Request, error) {
	err := r.ParseForm()
	if err != nil {
		log.Error(err)
		return r, err
	}
	rid := r.Form.Get(models.RecipientParameter)
	if rid == "" {
		return r, ErrInvalidRequest
	}
	// Since we want to support the common case of adding a "+" to indicate a
	// transparency request, we need to take care to handle the case where the
	// request ends with a space, since a "+" is technically reserved for use
	// as a URL encoding of a space.
	if strings.HasSuffix(rid, " ") {
		// We'll trim off the space
		rid = strings.TrimRight(rid, " ")
		// Then we'll add the transparency suffix
		rid = fmt.Sprintf("%s%s", rid, TransparencySuffix)
	}
	// Finally, if this is a transparency request, we'll need to verify that
	// a valid rid has been provided, so we'll look up the result with a
	// trimmed parameter.
	id := strings.TrimSuffix(rid, TransparencySuffix)
	// Check to see if this is a preview or a real result
	if strings.HasPrefix(id, models.PreviewPrefix) {
		rs, err := models.GetEmailRequestByResultId(id)
		if err != nil {
			return r, err
		}
		r = ctx.Set(r, "result", rs)
		return r, nil
	}
	rs, err := models.GetResult(id)
	if err != nil {
		return r, err
	}
	c, err := models.GetCampaign(rs.CampaignId, rs.UserId)
	if err != nil {
		log.Error(err)
		return r, err
	}
	// Don't process events for completed campaigns
	if c.Status == models.CampaignComplete {
		return r, ErrCampaignComplete
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	// Handle post processing such as GeoIP
	err = rs.UpdateGeo(ip)
	if err != nil {
		log.Error(err)
	}
	d := models.EventDetails{
		Payload: r.Form,
		Browser: make(map[string]string),
	}
	d.Browser["address"] = ip
	d.Browser["user-agent"] = r.Header.Get("User-Agent")

	r = ctx.Set(r, "rid", rid)
	r = ctx.Set(r, "result", rs)
	r = ctx.Set(r, "campaign", c)
	r = ctx.Set(r, "details", d)
	return r, nil
}
