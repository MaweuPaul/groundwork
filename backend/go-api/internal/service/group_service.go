package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"groundwork/internal/repository"
)

// maxGroupMembers keeps one combined run to a reasonable number of Earth Engine jobs.
const maxGroupMembers = 5

type GroupMemberInput struct {
	Methodology        string // slug
	MethodologyVersion string
	// Optional; any the methodology takes but the request leaves out get their defaults.
	Parameters map[string]float64
}

type CreateGroupInput struct {
	ParcelID  uuid.UUID
	OwnerID   uuid.UUID
	DateStart string // YYYY-MM-DD
	DateEnd   string // YYYY-MM-DD
	Members   []GroupMemberInput
}

// CreateGroup validates a combined run, several methodologies for the same window, and
// queues one snapshot per methodology. The whole request is refused if any member is invalid.
func (s *snapshotService) CreateGroup(in CreateGroupInput) (repository.GroupRef, error) {
	if len(in.Members) < 2 || len(in.Members) > maxGroupMembers {
		return repository.GroupRef{}, &ValidationError{fmt.Sprintf("a combined run needs 2 to %d methodologies", maxGroupMembers)}
	}
	start, end, err := s.parseDates(in.DateStart, in.DateEnd, true)
	if err != nil {
		return repository.GroupRef{}, err
	}

	seen := make(map[string]bool, len(in.Members))
	members := make([]repository.NewSnapshot, 0, len(in.Members))
	for _, member := range in.Members {
		slug := strings.TrimSpace(member.Methodology)
		version := strings.TrimSpace(member.MethodologyVersion)
		if slug == "" || version == "" {
			return repository.GroupRef{}, &ValidationError{"each member needs a methodology and methodology_version"}
		}
		if seen[slug] {
			return repository.GroupRef{}, &ValidationError{fmt.Sprintf("%s appears more than once; a combined run takes each methodology once", slug)}
		}
		seen[slug] = true

		m, params, err := s.methodologyFor(slug, version, start, member.Parameters)
		if err != nil {
			return repository.GroupRef{}, err
		}
		members = append(members, repository.NewSnapshot{
			MethodologyID:      m.ID,
			MethodologyVersion: m.Version,
			Parameters:         params,
			Limits:             s.limits,
		})
	}

	return s.snapshots.CreateGroupWithJobs(repository.NewGroup{
		ParcelID:  in.ParcelID,
		OwnerID:   in.OwnerID,
		DateStart: start,
		DateEnd:   end,
		Members:   members,
		Limits:    s.limits,
	})
}

// ListParcelGroups returns repository.ErrParcelNotFound for someone else's parcel.
func (s *snapshotService) ListParcelGroups(parcelID, ownerID uuid.UUID) ([]repository.GroupSummary, error) {
	return s.snapshots.ListGroupsByParcelForOwner(parcelID, ownerID)
}
