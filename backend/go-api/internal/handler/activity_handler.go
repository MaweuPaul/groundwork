package handler

import (
	"net/http"
	"strconv"

	"groundwork/internal/httpx"
	"groundwork/internal/logging"
	"groundwork/internal/middleware"
	"groundwork/internal/service"
)

type ActivityHandler struct {
	service service.ActivityService
}

func NewActivityHandler(service service.ActivityService) *ActivityHandler {
	return &ActivityHandler{service: service}
}

// List returns the signed-in user's recent activity. ?limit= is optional.
func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "limit must be a positive whole number")
			return
		}
		limit = n
	}

	user := middleware.UserFrom(r.Context())
	events, err := h.service.Recent(user.ID, limit)
	if err != nil {
		logging.From(r.Context()).Error("activity: list", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, events)
}
