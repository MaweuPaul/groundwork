package service

import (
	"groundwork/internal/models"
	"groundwork/internal/repository"

	"github.com/google/uuid"
)

type ParcelService interface {
	CreateParcel(ownerID uuid.UUID, parcel *models.Parcel) error
	ListParcels(ownerID uuid.UUID) ([]models.Parcel, error)
}

func (s *parcelService) CreateParcel(ownerID uuid.UUID, parcel *models.Parcel) error {
	parcel.OwnerID = ownerID // the owner always comes from the signed-in user
	return s.repo.Create(parcel)
}

func (s *parcelService) ListParcels(ownerID uuid.UUID) ([]models.Parcel, error) {
	return s.repo.ListByOwner(ownerID)
}

type parcelService struct {
	repo repository.ParcelRepository
}

func NewParcelService(repo repository.ParcelRepository) ParcelService {
	return &parcelService{repo: repo}
}
