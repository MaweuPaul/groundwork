package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGEEServiceClient_CalculateSnapshot_Success(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and URL
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/calculate-snapshot" {
			t.Errorf("Expected path /calculate-snapshot, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode the request body to ensure it's valid
		var reqBody SnapshotRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify some of the incoming data
		if reqBody.Methodology.IndexType != "ndvi" {
			t.Errorf("Expected index_type ndvi, got %s", reqBody.Methodology.IndexType)
		}

		// Send a mock successful response
		resp := SnapshotResponse{
			Measurements: map[string]float64{
				"ndvi_mean": 0.65,
				"ndvi_max":  0.80,
			},
			ImagerySources: []ImagerySource{
				{
					Provider:        "earthengine",
					Satellite:       "sentinel-2",
					SceneID:         "mock_scene_1",
					AcquisitionTime: "2023-01-15T10:00:00Z",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	// Initialize the client with the mock server URL
	client := NewGEEServiceClient(mockServer.URL)

	// Create a dummy request
	req := SnapshotRequest{
		GeometryGeoJSON: map[string]interface{}{
			"type": "Polygon",
			"coordinates": [][][]float64{
				{{-122.0, 37.0}, {-121.0, 37.0}, {-121.0, 38.0}, {-122.0, 38.0}, {-122.0, 37.0}},
			},
		},
		DateStart: "2023-01-01",
		DateEnd:   "2023-01-31",
		Methodology: MethodologyConfig{
			Satellite:       "sentinel-2",
			IndexType:       "ndvi",
			CompositeMethod: "median",
		},
	}

	// Call the client
	resp, err := client.CalculateSnapshot(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Assert the response matches the mock
	if resp == nil {
		t.Fatal("Expected a response, got nil")
	}
	if resp.Measurements["ndvi_mean"] != 0.65 {
		t.Errorf("Expected ndvi_mean 0.65, got %v", resp.Measurements["ndvi_mean"])
	}
	if len(resp.ImagerySources) != 1 {
		t.Errorf("Expected 1 imagery source, got %d", len(resp.ImagerySources))
	}
	if resp.ImagerySources[0].SceneID != "mock_scene_1" {
		t.Errorf("Expected SceneID mock_scene_1, got %s", resp.ImagerySources[0].SceneID)
	}
}

func TestGEEServiceClient_CalculateSnapshot_ErrorStatus(t *testing.T) {
	// Mock a 500 error response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"detail":"Earth Engine calculation failed"}`))
	}))
	defer mockServer.Close()

	client := NewGEEServiceClient(mockServer.URL)

	// Call the client
	resp, err := client.CalculateSnapshot(context.Background(), SnapshotRequest{})
	
	// We expect an error
	if err == nil {
		t.Fatal("Expected an error for 500 status code, got nil")
	}
	if resp != nil {
		t.Errorf("Expected nil response on error, got %+v", resp)
	}
}

func TestGEEServiceClient_CalculateSnapshot_InvalidJSON(t *testing.T) {
	// Mock a response with broken JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ invalid_json:`))
	}))
	defer mockServer.Close()

	client := NewGEEServiceClient(mockServer.URL)

	resp, err := client.CalculateSnapshot(context.Background(), SnapshotRequest{})
	if err == nil {
		t.Fatal("Expected an error when decoding invalid JSON, got nil")
	}
	if resp != nil {
		t.Errorf("Expected nil response on error, got %+v", resp)
	}
}
