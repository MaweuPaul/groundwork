package service

import (
	"groundwork/internal/models"
	"groundwork/internal/repository"
)

type ParcelService interface {
	CreateParcel(parcel *models.Parcel) error
	ListParcels() ([]models.Parcel, error)
}

type parcelService struct {
	repo repository.ParcelRepository
}

func NewParcelService(repo repository.ParcelRepository) ParcelService {
	return &parcelService{repo: repo}
}

func (s *parcelService) CreateParcel(parcel *models.Parcel) error {
	return s.repo.Create(parcel)
}

func (s *parcelService) ListParcels() ([]models.Parcel, error) {
	return s.repo.GetAll()
}
