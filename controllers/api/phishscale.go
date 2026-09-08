package api

import (
	"encoding/json"
	"net/http"

	"github.com/gophish/gophish/models"
)

// phishScaleRequest is the body posted to /api/phishscale to score a draft
// message with the NIST Phish Scale.
type phishScaleRequest struct {
	Subject     string `json:"subject"`
	HTML        string `json:"html"`
	Text        string `json:"text"`
	FromAddress string `json:"from_address"`
	Premise     int    `json:"premise"`
}

// PhishScale scores a message with the NIST Phish Scale and returns the detected
// cues and detection difficulty.
func (as *Server) PhishScale(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	var req phishScaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	if req.Premise == 0 {
		req.Premise = 3 // default to medium alignment when unspecified
	}
	result := models.ScorePhishScale(req.Subject, req.HTML, req.Text, req.FromAddress, req.Premise)
	JSONResponse(w, result, http.StatusOK)
}
