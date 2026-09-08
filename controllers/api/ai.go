package api

import (
	"encoding/json"
	"net/http"

	"github.com/gophish/gophish/models"
)

// aiPretextRequest is the body posted to /api/ai/pretext.
type aiPretextRequest struct {
	Scenario string `json:"scenario"`
	Context  string `json:"context"`
	Language string `json:"language"`
	Count    int    `json:"count"`
}

// AIPretext generates simulated phishing pretext variants with the configured
// AI provider.
func (as *Server) AIPretext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONResponse(w, models.Response{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}
	if !models.AIConfigured() {
		JSONResponse(w, models.Response{Success: false, Message: models.ErrAINotConfigured.Error()}, http.StatusServiceUnavailable)
		return
	}
	var req aiPretextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSONResponse(w, models.Response{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	if req.Count == 0 {
		req.Count = 3
	}
	variants, err := models.GeneratePretexts(req.Scenario, req.Context, req.Language, req.Count)
	if err != nil {
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusBadGateway)
		return
	}
	JSONResponse(w, variants, http.StatusOK)
}
