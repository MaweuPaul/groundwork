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
    status: SnapshotStatus;
    created_at: string;
    completed_at?: string;
  } | null;
  latest_result: {
    snapshot_id: string;
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
};

export type Snapshot = {
  id: string;
  parcel_id: string;
  status: SnapshotStatus;
  failure_message?: string;
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
  geometry_hash: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  chain_seq?: number;
  record_hash?: string;
  prev_record_hash?: string;
  job?: SnapshotJob;
  measurements: Record<string, number>;
  imagery_sources: ImagerySource[];
};

export type NewSnapshot = {
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
};

export type QueuedSnapshot = {
  id: string;
  status: "pending";
};

export type SnapshotSummary = {
  id: string;
  status: SnapshotStatus;
  methodology: string;
  methodology_version: string;
  date_start: string;
  date_end: string;
  created_at: string;
  completed_at?: string;
  chain_seq?: number;
  record_hash?: string;
  measurements: Record<string, number>;
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
  cloud_mask: string;
  max_cloud_pct: number;
  composite_method: string;
  date_lookback_days: number;
  created_at: string;
};
