package handler

import (
	"log"
	"net/http"

	"groundwork/internal/httpx"
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
	if err := httpx.DecodeJSON(w, r, &parcel); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.CreateParcel(&parcel); err != nil {
		log.Printf("parcels: create: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, parcel)
}

func (h *ParcelHandler) List(w http.ResponseWriter, r *http.Request) {
	parcels, err := h.service.ListParcels()
	if err != nil {
		log.Printf("parcels: list: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, parcels)
}
