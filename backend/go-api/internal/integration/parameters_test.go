package integration

import (
	"testing"

	"gorm.io/gorm"

	"groundwork/internal/repository"
)

func TestParametersAreResolvedAndMirrored(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Parameters parcel")
	id := queueSnapshot(t, tx, owner, parcel, repository.Parameters{"max_scene_cloud_pct": 60})

	var row struct {
		Parameters          repository.Parameters
		MaxSceneCloudPct    float64
		CloudScoreThreshold float64
		RecordFormat        int
	}
	if err := tx.Raw(`
		SELECT parameters, max_scene_cloud_pct::float8, cloud_score_threshold::float8, record_format
		FROM snapshots WHERE id = ?`, id).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	want := repository.Parameters{"max_scene_cloud_pct": 60, "cloud_score_threshold": 0.6, "min_clear_pct": 50}
	if len(row.Parameters) != len(want) {
		t.Errorf("parameters = %v, want %v", row.Parameters, want)
	}
	for name, value := range want {
		if row.Parameters[name] != value {
			t.Errorf("parameter %s = %v, want %v", name, row.Parameters[name], value)
		}
	}
	if row.MaxSceneCloudPct != 60 || row.CloudScoreThreshold != 0.6 || row.RecordFormat != 3 {
		t.Errorf("legacy columns %v / %v, format %d", row.MaxSceneCloudPct, row.CloudScoreThreshold, row.RecordFormat)
	}
}

func TestLegacyCloudColumnsBecomeParameters(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Legacy parcel")
	m := ndvi(t, tx)

	var row struct{ Parameters repository.Parameters }
	err := tx.Raw(`
		INSERT INTO snapshots (parcel_id, methodology_id, methodology_version, date_start, date_end,
		                       parcel_geometry, geometry_hash, max_scene_cloud_pct, cloud_score_threshold)
		SELECT p.id, ?, ?, '2026-10-01', '2026-10-30', p.geometry,
		       encode(sha256(ST_AsEWKB(p.geometry, 'NDR')), 'hex'), 35, 0.5
		FROM parcels p WHERE p.id = ?
		RETURNING parameters`, m.ID, m.Version, parcel.ID).Scan(&row).Error
	if err != nil {
		t.Fatal(err)
	}
	params := row.Parameters
	if params["max_scene_cloud_pct"] != 35 || params["cloud_score_threshold"] != 0.5 || params["min_clear_pct"] != 50 {
		t.Errorf("parameters from legacy columns = %v", params)
	}
}

func TestDatabaseRejectsBadParameters(t *testing.T) {
	tx := begin(t)
	owner := newUser(t, tx, "asset_holder")
	parcel := newParcel(t, tx, owner, "Bad parameters parcel")
	m := ndvi(t, tx)

	cases := []struct {
		name   string
		params string
		want   string
	}{
		{"unknown name", `{"rainfall_mm": 3}`, `Unknown parameter "rainfall_mm"`},
		{"not a number", `{"min_clear_pct": "lots"}`, `"min_clear_pct" must be a number`},
		{"above the maximum", `{"cloud_score_threshold": 0.95}`, `"cloud_score_threshold" must be between`},
		{"below the minimum", `{"min_clear_pct": -1}`, `"min_clear_pct" must be between`},
		{"not an object", `[1, 2]`, `must be a JSON object`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			expectError(t, tx, c.want, func(tx *gorm.DB) error {
				return tx.Exec(`SELECT resolve_methodology_parameters(?, ?::jsonb)`, m.ID, c.params).Error
			})
		})
	}

	// The same rule guards the insert itself, not only the API.
	expectError(t, tx, `Unknown parameter "rainfall_mm"`, func(tx *gorm.DB) error {
		_, err := repository.NewSnapshotRepository(tx).CreateWithJob(repository.NewSnapshot{
			ParcelID: parcel.ID, OwnerID: owner, MethodologyID: m.ID, MethodologyVersion: m.Version,
			DateStart: day("2026-10-01"), DateEnd: day("2026-10-30"),
			Parameters: repository.Parameters{"rainfall_mm": 3},
		})
		return err
	})
}

func TestMethodologiesListTheirParameterSpecs(t *testing.T) {
	tx := begin(t)
	methodologies, err := repository.NewMethodologyRepository(tx).List()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range methodologies {
		if m.Slug != "ndvi-s2-median" {
			continue
		}
		specs := map[string][3]float64{}
		for _, p := range m.Parameters {
			specs[p.Name] = [3]float64{p.Min, p.Max, p.Default}
		}
		want := map[string][3]float64{
			"cloud_score_threshold": {0.4, 0.8, 0.6},
			"max_scene_cloud_pct":   {0.01, 100, 20},
			"min_clear_pct":         {0, 100, 50},
		}
		for name, w := range want {
			if specs[name] != w {
				t.Errorf("%s spec = %v, want min/max/default %v", name, specs[name], w)
			}
		}
		return
	}
	t.Fatal("ndvi-s2-median is not listed")
}
