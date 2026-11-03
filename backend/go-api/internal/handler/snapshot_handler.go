package handler

import (
	"errors"
	"log"
	"net/http"

	"groundwork/internal/httpx"
	"groundwork/internal/middleware"
	"groundwork/internal/repository"
	"groundwork/internal/service"
)

type SnapshotHandler struct {
	service service.SnapshotService
	refs    service.RefService
}

func NewSnapshotHandler(service service.SnapshotService, refs service.RefService) *SnapshotHandler {
	return &SnapshotHandler{service: service, refs: refs}
}

type createSnapshotRequest struct {
	Methodology        string `json:"methodology"`
	MethodologyVersion string `json:"methodology_version"`
	DateStart          string `json:"date_start"`
	DateEnd            string `json:"date_end"`
	// Optional; omitted means the methodology's default.
	MaxSceneCloudPct    *float64 `json:"max_scene_cloud_pct"`
	CloudScoreThreshold *float64 `json:"cloud_score_threshold"`
}

// Create queues a snapshot of one of the caller's parcels ({id} is its UUID or short ID) and answers 202 at once;
// the Python worker computes it. Poll the Location URL for the result.
func (h *SnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	var req createSnapshotRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFrom(r.Context())
	id, err := h.service.CreateSnapshot(service.CreateSnapshotInput{
		ParcelID:            parcelID,
		OwnerID:             user.ID,
		Methodology:         req.Methodology,
		MethodologyVersion:  req.MethodologyVersion,
		DateStart:           req.DateStart,
		DateEnd:             req.DateEnd,
		MaxSceneCloudPct:    req.MaxSceneCloudPct,
		CloudScoreThreshold: req.CloudScoreThreshold,
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

// Get returns one snapshot ({id} is its UUID or short ID) with its status, results and
// hash. Clients poll it after Create until status is completed or failed. 404 covers
// missing and someone else's.
func (h *SnapshotHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := resolveRef(w, r, h.refs.SnapshotID, "snapshot not found")
	if !ok {
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

// NdviImage serves the snapshot's NDVI map image (a PNG rendered by the worker after the
// snapshot completed). 404 when there is none yet, or it isn't the caller's.
func (h *SnapshotHandler) NdviImage(w http.ResponseWriter, r *http.Request) {
	id, ok := resolveRef(w, r, h.refs.SnapshotID, "snapshot not found")
	if !ok {
		return
	}

	user := middleware.UserFrom(r.Context())
	png, err := h.service.GetNdviImage(id, user.ID)
	if err != nil {
		log.Printf("snapshots: ndvi image: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if png == nil {
		httpx.WriteError(w, http.StatusNotFound, "no NDVI image for this snapshot")
		return
	}

	// An image never changes once rendered, but it's private to the owner.
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(png)
}

// ListForParcel returns a parcel's snapshot history, newest first.
func (h *SnapshotHandler) ListForParcel(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	user := middleware.UserFrom(r.Context())
	snapshots, err := h.service.ListParcelSnapshots(parcelID, user.ID)
	if errors.Is(err, repository.ErrParcelNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "parcel not found")
		return
	}
	if err != nil {
		log.Printf("snapshots: list for parcel: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, snapshots)
}

func (h *SnapshotHandler) ListMethodologies(w http.ResponseWriter, r *http.Request) {
	methodologies, err := h.service.ListMethodologies()
	if err != nil {
		log.Printf("methodologies: list: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, methodologies)
}
