package api

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
)

// RiskUsers returns the aggregated per-recipient risk scores for the user.
func (as *Server) RiskUsers(w http.ResponseWriter, r *http.Request) {
	scores, err := models.GetUserRiskScores(ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, scores, http.StatusOK)
}

// RiskGroups returns the aggregated per-group ("department") risk scores.
func (as *Server) RiskGroups(w http.ResponseWriter, r *http.Request) {
	scores, err := models.GetGroupRiskScores(ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, scores, http.StatusOK)
}

// RiskTrend returns the per-campaign risk score over time for the user.
func (as *Server) RiskTrend(w http.ResponseWriter, r *http.Request) {
	points, err := models.GetRiskTrend(ctx.Get(r, "user_id").(int64))
	if err != nil {
		log.Error(err)
		JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	JSONResponse(w, points, http.StatusOK)
}
