export type Role = "asset_holder" | "auditor";

export type User = {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  username: string;
  role: Role;
  created_at: string;
  updated_at: string;
};

export type MultiPolygon = {
  type: "MultiPolygon";
  coordinates: [number, number][][][];
};

export type Polygon = {
  type: "Polygon";
  coordinates: [number, number][][];
};

export type Parcel = {
  id: string;
  short_id: string;
  owner_id: string;
  name: string;
  description: string;
  geometry: MultiPolygon;
  area_ha: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  centroid_lon: number;
  centroid_lat: number;
  latest_snapshot: {
    id: string;
    short_id: string;
    status: SnapshotStatus;
    created_at: string;
    completed_at?: string;
  } | null;
  latest_result: {
    snapshot_id: string;
    methodology: string; // slug
    completed_at: string;
    measurements: Record<string, number>;
  } | null;
};

export type NewParcel = {
  name: string;
  description?: string;
  geometry: Polygon | MultiPolygon;
  metadata?: Record<string, unknown>;
};

// The rendered NDVI composite of a completed snapshot, drawn over the map between bounds.
export type NdviImage = {
  bounds: [west: number, south: number, east: number, north: number];
  width: number;
  height: number;
  visualization: { min: number; max: number; palette: string[] };
};

export type SnapshotStatus = "pending" | "processing" | "completed" | "failed";

export type SnapshotJob = {
  status: "queued" | "running" | "succeeded" | "failed";
  attempts: number;
  max_attempts: number;
  next_attempt_at?: string;
  last_error_code?: string;
};

export type ImagerySource = {
  scene_id: string;
  satellite: string;
  acquisition_time: string;
  cloud_cover_pct?: number;
  // The provider's version of the source when it was read; sealed from record format 3.
  source_version?: string;
};

export type Snapshot = {
  id: string;
  short_id: string;
  parcel_id: string;
  parcel_short_id: string;
  parcel_name: string;
  status: SnapshotStatus;
  failure_message?: string;
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
  parameters: Record<string, number>;
  flags: SnapshotFlag[];
  // Only NDVI snapshots have these; read parameters instead.
  max_scene_cloud_pct: number | null;
  cloud_score_threshold: number | null;
  record_format: number;
  geometry_hash: string;
  // Sealed from record format 4.
  code_version?: string;
  manifest_sha256?: string;
  artifacts: Artifact[];
  created_at: string;
  started_at?: string;
  completed_at?: string;
  chain_seq?: number;
  // Position in the parcel's own hash chain.
  parcel_seq?: number;
  record_hash?: string;
  prev_record_hash?: string;
  job?: SnapshotJob;
  measurements: Record<string, number>;
  // Name, unit and description of each measurement, by slug.
  measurement_types: Record<string, MeasurementType>;
  imagery_sources: ImagerySource[];
  ndvi_image?: NdviImage;
};

export type MeasurementType = {
  name: string;
  unit: string | null;
  description: string | null;
};

// A quality warning raised by the worker and sealed with the record, e.g. low_coverage.
// A flagged snapshot still completed; the flag says how far to trust it.
export type SnapshotFlag = {
  flag: string;
  detail?: string;
};

export type NewSnapshot = {
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
  // Any the methodology takes but this leaves out get their defaults; all are sealed into the record.
  parameters?: Record<string, number>;
};

export type QueuedSnapshot = {
  id: string;
  status: "pending";
};

export type SnapshotSummary = {
  id: string;
  short_id: string;
  status: SnapshotStatus;
  failure_message?: string;
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
  parameters: Record<string, number>;
  flags: SnapshotFlag[];
  // Only NDVI snapshots have these; read parameters instead.
  max_scene_cloud_pct: number | null;
  cloud_score_threshold: number | null;
  created_at: string;
  completed_at?: string;
  chain_seq?: number;
  // Position in the parcel's own hash chain.
  parcel_seq?: number;
  record_hash?: string;
  measurements: Record<string, number>;
  ndvi_image?: NdviImage;
  series_short_id?: string;
  group_short_id?: string;
};

export type NewSeries = NewSnapshot & { window_months: number; interval_months: number };

// A combined run: several methodologies for the same window, each with its own settings.
export type NewGroup = {
  date_start: string;
  date_end: string;
  members: Pick<NewSnapshot, "methodology" | "methodology_version" | "parameters">[];
};

export type GroupRef = { id: string; short_id: string; members: number };

// Sealed as a set, with one hash over every member's result, once every member finishes.
export type GroupSummary = {
  id: string;
  short_id: string;
  date_start: string;
  date_end: string;
  record_hash?: string;
  sealed_at?: string;
  created_at: string;
  members: GroupMember[];
};

export type GroupMember = {
  snapshot_id: string;
  short_id: string;
  methodology: string;
  methodology_version: string;
  status: SnapshotStatus;
  record_hash?: string;
  measurements: Record<string, number>;
};

// Every sealed record and group of a parcel, recomputed from what is stored now.
export type ChainCheck = {
  intact: boolean;
  records: number;
  groups: number;
  head_seq: number | null;
  head_hash: string | null;
  problems: ChainProblem[];
  checked_at: string;
};

// Names the record or group that failed; a broken chain head has neither.
export type ChainProblem = {
  seq?: number;
  snapshot_id?: string;
  short_id?: string;
  group_short_id?: string;
  problem: string;
};

export type SeriesRef = { id: string; short_id: string; windows: number };

// A time series: one snapshot per window, sealed as a set once every window finishes.
export type SeriesSummary = {
  id: string;
  short_id: string;
  date_start: string;
  date_end: string;
  window_months: number;
  interval_months: number;
  parameters: Record<string, number>;
  // Only NDVI snapshots have these; read parameters instead.
  max_scene_cloud_pct: number | null;
  cloud_score_threshold: number | null;
  windows: number;
  completed: number;
  failed: number;
  record_hash?: string;
  sealed_at?: string;
  created_at: string;
};

export type Methodology = {
  id: string;
  name: string;
  slug: string;
  version: string;
  description: string;
  index_type: string;
  satellite: string;
  bands: string[];
  cloud_mask: string | null;
  max_cloud_pct: number;
  composite_method: string;
  date_lookback_days: number;
  data_start: string; // first day the methodology's data covers
  created_at: string;
  parameters: MethodologyParameter[];
  measurements: MethodologyMeasurement[];
  // The published spec; every record made with this methodology seals its hash.
  manifest_text: string;
  manifest_sha256: string;
  records: number; // completed snapshots the signed-in user has made with it
};

// One measurement a methodology produces, in display order.
export type MethodologyMeasurement = {
  slug: string;
  name: string;
  unit: string | null;
  description: string;
};

// A setting a methodology accepts. The API rejects values outside min to max.
export type MethodologyParameter = {
  name: string;
  label: string;
  type: string;
  min: number;
  max: number;
  default: number;
};

// One line of the recent-activity feed. Fields that don't apply to a kind are left out.
export type ActivityEvent = {
  kind:
    | 'parcel_registered'
    | 'snapshot_queued'
    | 'snapshot_completed'
    | 'snapshot_failed'
    | 'series_started'
    | 'series_sealed';
  at: string;
  parcel_id: string;
  parcel_short_id: string;
  parcel_name: string;
  snapshot_short_id?: string;
  series_short_id?: string;
  record_hash?: string;
  methodology?: string; // slug, on completed snapshots
  measurements?: Record<string, number>;
  windows?: number;
  flags?: string[];
  detail?: string;
};

// A file a methodology produced with its result, e.g. the NDVI map image. Sealed into the
// record from format 5; earlier snapshots' images were drawn after sealing.
export type Artifact = {
  key: string;
  media_type: string;
  sha256: string;
  metadata_sha256: string;
  byte_size: number;
  metadata: Record<string, unknown>;
  sealed: boolean;
};
