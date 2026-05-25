package openapi

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOpenAPISnapshotContainsRequiredPaths(t *testing.T) {
	data, err := os.ReadFile("altfins-openapi.json")
	if err != nil {
		t.Fatalf("read openapi snapshot: %v", err)
	}

	var doc struct {
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode openapi snapshot: %v", err)
	}

	required := []string{
		"/api/v2/public/symbols",
		"/api/v2/public/available-permits",
		"/api/v2/public/screener-data/search-requests",
		"/api/v2/public/technical-analysis/data",
		"/api/v2/public/signals-feed/search-requests",
		"/api/v2/public/news-summary/search-requests",
	}
	for _, path := range required {
		if _, ok := doc.Paths[path]; !ok {
			t.Fatalf("missing required path %q", path)
		}
	}

	signalFeed, ok := doc.Components.Schemas["ApiSignalFeed"]
	if !ok {
		t.Fatal("missing ApiSignalFeed schema")
	}
	if _, ok := signalFeed.Properties["bullish"]; !ok {
		t.Fatal("ApiSignalFeed schema must expose bullish response field")
	}
}
