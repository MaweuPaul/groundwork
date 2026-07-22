package handler

import (
	"encoding/json"
	"net/http"

	"groundwork/internal/models"
	"groundwork/internal/service"
)

type ParcelHandler struct {
	service service.ParcelService
}

func NewParcelHandler(service service.ParcelService) *ParcelHandler {
	return &ParcelHandler{service: service}
}

func (h *ParcelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var parcel models.Parcel
	if err := json.NewDecoder(r.Body).Decode(&parcel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.service.CreateParcel(&parcel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(parcel)
}

func (h *ParcelHandler) List(w http.ResponseWriter, r *http.Request) {
	parcels, err := h.service.ListParcels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(parcels)
}
