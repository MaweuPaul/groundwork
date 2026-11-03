package repository

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// NdviImageInfo says where to draw a snapshot's NDVI image and how it was coloured.
// The PNG itself is served separately (GET /snapshots/{id}/ndvi.png).
type NdviImageInfo struct {
	Bounds        [4]float64     `json:"bounds"` // west, south, east, north (WGS84)
	Width         int            `json:"width"`
	Height        int            `json:"height"`
	Visualization datatypes.JSON `json:"visualization"` // {"min", "max", "palette"}
}

// ndviImagesBySnapshot returns the rendered images among ids; failed renders are left out.
func ndviImagesBySnapshot(db *gorm.DB, ids []uuid.UUID) (map[uuid.UUID]*NdviImageInfo, error) {
	images := map[uuid.UUID]*NdviImageInfo{}
	if len(ids) == 0 {
		return images, nil
	}

	var rows []struct {
		SnapshotID               uuid.UUID
		West, South, East, North float64
		Width, Height            int
		Visualization            datatypes.JSON
	}
	err := db.Raw(`
		SELECT snapshot_id, west, south, east, north, width, height, visualization
		FROM snapshot_ndvi_images
		WHERE snapshot_id IN ? AND png IS NOT NULL`,
		ids,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, r := range rows {
		images[r.SnapshotID] = &NdviImageInfo{
			Bounds:        [4]float64{r.West, r.South, r.East, r.North},
			Width:         r.Width,
			Height:        r.Height,
			Visualization: r.Visualization,
		}
	}
	return images, nil
}

// NdviPNGForOwner returns the snapshot's NDVI image, or nil when there is none or the
// snapshot isn't the owner's.
func (r *snapshotRepository) NdviPNGForOwner(id, ownerID uuid.UUID) ([]byte, error) {
	// A struct, not a bare []byte: GORM reads a slice destination as a list of rows.
	var row struct{ Png []byte }
	res := r.db.Raw(`
		SELECT i.png
		FROM snapshot_ndvi_images i
		JOIN snapshots s ON s.id = i.snapshot_id
		JOIN parcels p ON p.id = s.parcel_id
		WHERE i.snapshot_id = ? AND p.owner_id = ? AND i.png IS NOT NULL`,
		id, ownerID,
	).Scan(&row)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	return row.Png, nil
}
