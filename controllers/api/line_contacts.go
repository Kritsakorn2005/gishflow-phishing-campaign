package api

import (
	"net/http"
	"strconv"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// LineContacts handles requests for the /api/line_contacts endpoint: listing the
// LINE userIds collected automatically from the OA webhook.
func (as *Server) LineContacts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cs, err := models.GetLineContacts()
		if err != nil {
			JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, cs, http.StatusOK)
	}
}

// LineContact handles requests for a single /api/line_contacts/:id, currently
// just deletion of a collected contact.
func (as *Server) LineContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.ParseInt(vars["id"], 0, 64)
	switch r.Method {
	case http.MethodDelete:
		if err := models.DeleteLineContact(id); err != nil {
			log.Error(err)
			JSONResponse(w, models.Response{Success: false, Message: "Error deleting contact"}, http.StatusInternalServerError)
			return
		}
		JSONResponse(w, models.Response{Success: true, Message: "Contact deleted"}, http.StatusOK)
	}
}
