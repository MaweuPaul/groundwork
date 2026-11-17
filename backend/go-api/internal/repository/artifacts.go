package repository

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// NdviMapKey is the artifact key of the NDVI plugin's map image.
const NdviMapKey = "ndvi_map"

// ArtifactInfo describes a file a methodology produced with its result. The content is
// served separately (GET /snapshots/{id}/artifacts/{key}).
type ArtifactInfo struct {
	Key            string         `json:"key"`
	MediaType      string         `json:"media_type"`
	SHA256         string         `json:"sha256"`
	MetadataSHA256 string         `json:"metadata_sha256"`
	ByteSize       int            `json:"byte_size"`
	Metadata       datatypes.JSON `json:"metadata"`
	// Whether the record hash covers it: true from record format 5. Earlier snapshots' images
	// were drawn after sealing.
	Sealed bool `json:"sealed"`
}

// Artifact is an artifact's content, for serving.
type Artifact struct {
	MediaType string
	SHA256    string
	Content   []byte
}

// NdviImageInfo says where to draw a snapshot's NDVI image and how it was coloured; it is the
// ndvi_map artifact's metadata. The PNG is served at GET /snapshots/{id}/ndvi.png.
type NdviImageInfo struct {
	Bounds        [4]float64     `json:"bounds"` // west, south, east, north (WGS84)
	Width         int            `json:"width"`
	Height        int            `json:"height"`
	Visualization datatypes.JSON `json:"visualization"` // {"min", "max", "palette"}
}

// artifactsBySnapshot lists the artifacts of each of ids, by key, without their content.
func artifactsBySnapshot(db *gorm.DB, ids []uuid.UUID) (map[uuid.UUID][]ArtifactInfo, error) {
	found := map[uuid.UUID][]ArtifactInfo{}
	if len(ids) == 0 {
		return found, nil
	}

	var rows []struct {
		SnapshotID uuid.UUID
		ArtifactInfo
	}
	err := db.Raw(`
		SELECT a.snapshot_id, a.key, a.media_type, a.sha256, a.metadata_sha256, a.byte_size, a.metadata,
		       s.record_format >= 5 AS sealed
		FROM artifacts a
		JOIN snapshots s ON s.id = a.snapshot_id
		WHERE a.snapshot_id IN ?
		ORDER BY a.key`,
		ids,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		found[r.SnapshotID] = append(found[r.SnapshotID], r.ArtifactInfo)
	}
	return found, nil
}

// ndviImage is the map placement of the ndvi_map artifact, or nil when there is none.
func ndviImage(artifacts []ArtifactInfo) *NdviImageInfo {
	for _, a := range artifacts {
		if a.Key != NdviMapKey {
			continue
		}
		var info NdviImageInfo
		if json.Unmarshal(a.Metadata, &info) != nil {
			return nil
		}
		return &info
	}
	return nil
}

func artifactsOrEmpty(a []ArtifactInfo) []ArtifactInfo {
	if a == nil {
		return []ArtifactInfo{}
	}
	return a
}

// ArtifactForOwner returns one artifact, or nil when there is none or the snapshot isn't the owner's.
func (r *snapshotRepository) ArtifactForOwner(id, ownerID uuid.UUID, key string) (*Artifact, error) {
	var rows []Artifact
	err := r.db.Raw(`
		SELECT a.media_type, a.sha256, a.content
		FROM artifacts a
		JOIN snapshots s ON s.id = a.snapshot_id
		JOIN parcels p ON p.id = s.parcel_id
		WHERE a.snapshot_id = ? AND a.key = ? AND p.owner_id = ?`,
		id, key, ownerID,
	).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}
