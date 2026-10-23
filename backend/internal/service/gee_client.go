package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// MethodologyConfig defines the configuration to send to the GEE python service
type MethodologyConfig struct {
	Satellite       string   `json:"satellite"`
	IndexType       string   `json:"index_type"`
	Bands           []string `json:"bands"`
	CloudMask       string   `json:"cloud_mask"`
	MaxCloudPct     float64  `json:"max_cloud_pct"`
	CompositeMethod string   `json:"composite_method"`
}

// SnapshotRequest is the payload sent to the GEE python service
type SnapshotRequest struct {
	GeometryGeoJSON map[string]interface{} `json:"geometry_geojson"`
	DateStart       string                 `json:"date_start"` // YYYY-MM-DD
	DateEnd         string                 `json:"date_end"`   // YYYY-MM-DD
	Methodology     MethodologyConfig      `json:"methodology"`
}

// ImagerySource represents a single satellite scene used in the snapshot
type ImagerySource struct {
	Provider        string                 `json:"provider"`
	Satellite       string                 `json:"satellite"`
	SceneID         string                 `json:"scene_id"`
	AcquisitionTime string                 `json:"acquisition_time"`
	CloudCoverPct   *float64               `json:"cloud_cover_pct"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SnapshotResponse is the payload received from the GEE python service
type SnapshotResponse struct {
	Measurements   map[string]float64 `json:"measurements"`
	ImagerySources []ImagerySource    `json:"imagery_sources"`
}

// GEEServiceClient defines the interface for communicating with the GEE microservice
type GEEServiceClient interface {
	CalculateSnapshot(ctx context.Context, req SnapshotRequest) (*SnapshotResponse, error)
}

type geeServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewGEEServiceClient creates a new client for the GEE Python microservice.
// url should be the base URL, e.g., "http://localhost:8000"
func NewGEEServiceClient(url string) GEEServiceClient {
	return &geeServiceClient{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // GEE processing can take a while
		},
	}
}

func (c *geeServiceClient) CalculateSnapshot(ctx context.Context, req SnapshotRequest) (*SnapshotResponse, error) {
	endpoint := fmt.Sprintf("%s/calculate-snapshot", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gee service returned status %d", resp.StatusCode)
	}

	var snapResp SnapshotResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &snapResp, nil
}
