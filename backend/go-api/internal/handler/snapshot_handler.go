package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"

	"groundwork/internal/httpx"
	"groundwork/internal/middleware"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

type SnapshotHandler struct {
	service service.SnapshotService
}

func NewSnapshotHandler(service service.SnapshotService) *SnapshotHandler {
	return &SnapshotHandler{service: service}
}

type createSnapshotRequest struct {
	Methodology        string `json:"methodology"`
	MethodologyVersion string `json:"methodology_version"`
	DateStart          string `json:"date_start"`
	DateEnd            string `json:"date_end"`
}

// Create queues a snapshot of one of the caller's parcels and answers 202 at once;
// the Python worker computes it. Poll the Location URL for the result.
func (h *SnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	parcelID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "parcel not found")
		return
	}

	var req createSnapshotRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFrom(r.Context())
	id, err := h.service.CreateSnapshot(service.CreateSnapshotInput{
		ParcelID:           parcelID,
		OwnerID:            user.ID,
		Methodology:        req.Methodology,
		MethodologyVersion: req.MethodologyVersion,
		DateStart:          req.DateStart,
		DateEnd:            req.DateEnd,
	})

	var invalid *service.ValidationError
	switch {
	case errors.Is(err, repository.ErrParcelNotFound):
		httpx.WriteError(w, http.StatusNotFound, "parcel not found")
		return
	case errors.Is(err, service.ErrUnknownMethodology):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	case errors.As(err, &invalid):
		httpx.WriteError(w, http.StatusBadRequest, invalid.Message)
		return
	case err != nil:
		log.Printf("snapshots: create: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	// Headers must be set before the body is written.
	w.Header().Set("Location", "/snapshots/"+id.String())
	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{"id": id, "status": "pending"})
}

// Get returns one snapshot with its status, results and hash. Clients poll it after
// Create until status is completed or failed. 404 covers missing and someone else's.
func (h *SnapshotHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	user := middleware.UserFrom(r.Context())
	snapshot, err := h.service.GetSnapshot(id, user.ID)
	if err != nil {
		log.Printf("snapshots: get: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if snapshot == nil {
		httpx.WriteError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, snapshot)
}
