package handler

import (
	"errors"
	"net/http"

	"groundwork/internal/httpx"
	"groundwork/internal/logging"
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
	// Optional; a parameter left out gets the methodology's default.
	Parameters map[string]float64 `json:"parameters"`
	// Deprecated: send them in parameters. Used when parameters doesn't set them.
	MaxSceneCloudPct    *float64 `json:"max_scene_cloud_pct"`
	CloudScoreThreshold *float64 `json:"cloud_score_threshold"`
}

func (req createSnapshotRequest) parameters() map[string]float64 {
	params := make(map[string]float64, len(req.Parameters)+2)
	for name, value := range req.Parameters {
		params[name] = value
	}
	legacy := map[string]*float64{
		"max_scene_cloud_pct":   req.MaxSceneCloudPct,
		"cloud_score_threshold": req.CloudScoreThreshold,
	}
	for name, value := range legacy {
		if _, set := params[name]; !set && value != nil {
			params[name] = *value
		}
	}
	return params
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
		ParcelID:           parcelID,
		OwnerID:            user.ID,
		Methodology:        req.Methodology,
		MethodologyVersion: req.MethodologyVersion,
		DateStart:          req.DateStart,
		DateEnd:            req.DateEnd,
		Parameters:         req.parameters(),
	})

	if writeCreateError(w, r, err, "create") {
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
		logging.From(r.Context()).Error("snapshots: get", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if snapshot == nil {
		httpx.WriteError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, snapshot)
}

// Artifact serves one of the snapshot's artifacts ({key}, e.g. ndvi_map) with its media type
// and SHA-256. 404 when there is none, or it isn't the caller's.
func (h *SnapshotHandler) Artifact(w http.ResponseWriter, r *http.Request) {
	h.serveArtifact(w, r, r.PathValue("key"))
}

// NdviImage serves the NDVI map image: the ndvi_map artifact.
func (h *SnapshotHandler) NdviImage(w http.ResponseWriter, r *http.Request) {
	h.serveArtifact(w, r, repository.NdviMapKey)
}

func (h *SnapshotHandler) serveArtifact(w http.ResponseWriter, r *http.Request, key string) {
	id, ok := resolveRef(w, r, h.refs.SnapshotID, "snapshot not found")
	if !ok {
		return
	}

	user := middleware.UserFrom(r.Context())
	artifact, err := h.service.GetArtifact(id, user.ID, key)
	if err != nil {
		logging.From(r.Context()).Error("snapshots: artifact", "key", key, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	if artifact == nil {
		httpx.WriteError(w, http.StatusNotFound, "no such artifact for this snapshot")
		return
	}

	// Artifacts never change once stored, but they're private to the owner.
	w.Header().Set("Content-Type", artifact.MediaType)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-SHA256", artifact.SHA256)
	w.Write(artifact.Content)
}

type createSeriesRequest struct {
	createSnapshotRequest
	WindowMonths   int `json:"window_months"`
	IntervalMonths int `json:"interval_months"`
}

// CreateSeries queues one snapshot per window of the range and answers 202 at once.
// The series is sealed by the database when its last window finishes.
func (h *SnapshotHandler) CreateSeries(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	var req createSeriesRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user := middleware.UserFrom(r.Context())
	ref, err := h.service.CreateSeries(service.CreateSeriesInput{
		CreateSnapshotInput: service.CreateSnapshotInput{
			ParcelID:           parcelID,
			OwnerID:            user.ID,
			Methodology:        req.Methodology,
			MethodologyVersion: req.MethodologyVersion,
			DateStart:          req.DateStart,
			DateEnd:            req.DateEnd,
			Parameters:         req.parameters(),
		},
		WindowMonths:   req.WindowMonths,
		IntervalMonths: req.IntervalMonths,
	})

	if !writeCreateError(w, r, err, "create series") {
		httpx.WriteJSON(w, http.StatusAccepted, ref)
	}
}

type createGroupRequest struct {
	DateStart string `json:"date_start"`
	DateEnd   string `json:"date_end"`
	Members   []struct {
		Methodology        string             `json:"methodology"`
		MethodologyVersion string             `json:"methodology_version"`
		Parameters         map[string]float64 `json:"parameters"`
	} `json:"members"`
}

// CreateGroup queues a combined run: one snapshot per methodology, all for the same window,
// sealed as a group once every member finishes. Answers 202 at once.
func (h *SnapshotHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	var req createGroupRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	members := make([]service.GroupMemberInput, len(req.Members))
	for i, m := range req.Members {
		members[i] = service.GroupMemberInput{Methodology: m.Methodology, MethodologyVersion: m.MethodologyVersion, Parameters: m.Parameters}
	}
	ref, err := h.service.CreateGroup(service.CreateGroupInput{
		ParcelID:  parcelID,
		OwnerID:   middleware.UserFrom(r.Context()).ID,
		DateStart: req.DateStart,
		DateEnd:   req.DateEnd,
		Members:   members,
	})
	if !writeCreateError(w, r, err, "create group") {
		httpx.WriteJSON(w, http.StatusAccepted, ref)
	}
}

// ListGroupsForParcel returns a parcel's combined runs with their members and seals, newest first.
func (h *SnapshotHandler) ListGroupsForParcel(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}
	groups, err := h.service.ListParcelGroups(parcelID, middleware.UserFrom(r.Context()).ID)
	writeParcelRead(w, r, groups, err, "list groups")
}

// VerifyParcel recomputes every sealed record and group of the parcel and reports any that no
// longer match what is stored.
func (h *SnapshotHandler) VerifyParcel(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}
	check, err := h.service.VerifyParcel(parcelID, middleware.UserFrom(r.Context()).ID)
	writeParcelRead(w, r, check, err, "verify parcel")
}

// writeCreateError answers a failed create and reports whether it did; nil writes nothing.
func writeCreateError(w http.ResponseWriter, r *http.Request, err error, action string) bool {
	var invalid *service.ValidationError
	var limit *repository.LimitError
	switch {
	case err == nil:
		return false
	case errors.Is(err, repository.ErrParcelNotFound):
		httpx.WriteError(w, http.StatusNotFound, "parcel not found")
	case errors.Is(err, service.ErrUnknownMethodology):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.As(err, &invalid):
		httpx.WriteError(w, http.StatusBadRequest, invalid.Message)
	case errors.As(err, &limit):
		httpx.WriteError(w, limitStatus(limit), limit.Message)
	default:
		logging.From(r.Context()).Error("snapshots: "+action, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
	}
	return true
}

// writeParcelRead answers a read of one parcel's records; someone else's parcel is a 404.
func writeParcelRead(w http.ResponseWriter, r *http.Request, list any, err error, action string) {
	switch {
	case errors.Is(err, repository.ErrParcelNotFound):
		httpx.WriteError(w, http.StatusNotFound, "parcel not found")
	case err != nil:
		logging.From(r.Context()).Error("snapshots: "+action, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
	default:
		httpx.WriteJSON(w, http.StatusOK, list)
	}
}

// ListSeriesForParcel returns a parcel's series with progress and seals, newest first.
func (h *SnapshotHandler) ListSeriesForParcel(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	series, err := h.service.ListParcelSeries(parcelID, middleware.UserFrom(r.Context()).ID)
	writeParcelRead(w, r, series, err, "list series")
}

// ListForParcel returns a parcel's snapshot history, newest first.
func (h *SnapshotHandler) ListForParcel(w http.ResponseWriter, r *http.Request) {
	parcelID, ok := resolveRef(w, r, h.refs.ParcelID, "parcel not found")
	if !ok {
		return
	}

	snapshots, err := h.service.ListParcelSnapshots(parcelID, middleware.UserFrom(r.Context()).ID)
	writeParcelRead(w, r, snapshots, err, "list for parcel")
}

func (h *SnapshotHandler) ListMethodologies(w http.ResponseWriter, r *http.Request) {
	methodologies, err := h.service.ListMethodologies()
	if err != nil {
		logging.From(r.Context()).Error("methodologies: list", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, methodologies)
}

// limitStatus is 429 for a quota that frees up with time, 422 for a request that's too big.
func limitStatus(e *repository.LimitError) int {
	if e.Quota {
		return http.StatusTooManyRequests
	}
	return http.StatusUnprocessableEntity
}
