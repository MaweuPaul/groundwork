package handler

import (
	"log"
	"net/http"
	"strings"

	"gorm.io/datatypes"

	"groundwork/internal/httpx"
	"groundwork/internal/middleware"
	"groundwork/internal/models"
	"groundwork/internal/service"
)

type ParcelHandler struct {
	service service.ParcelService
}

func NewParcelHandler(service service.ParcelService) *ParcelHandler {
	return &ParcelHandler{service: service}
}

// What a client may send. ID, owner, area and timestamps are never taken from the body.
type createParcelRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Geometry    string         `json:"geometry"`
	Metadata    datatypes.JSON `json:"metadata"`
}

func (h *ParcelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createParcelRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Geometry) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "name and geometry are required")
		return
	}

	// metadata is NOT NULL in the database; an omitted field would be sent as NULL.
	if len(req.Metadata) == 0 {
		req.Metadata = datatypes.JSON("{}")
	}

	parcel := models.Parcel{
		Name:        strings.TrimSpace(req.Name),
		Description: req.Description,
		Geometry:    req.Geometry,
		Metadata:    req.Metadata,
	}

	user := middleware.UserFrom(r.Context())
	if err := h.service.CreateParcel(user.ID, &parcel); err != nil {
		log.Printf("parcels: create: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, parcel)
}

func (h *ParcelHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFrom(r.Context())
	parcels, err := h.service.ListParcels(user.ID)
	if err != nil {
		log.Printf("parcels: list: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, parcels)
}
