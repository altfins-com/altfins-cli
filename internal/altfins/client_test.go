package altfins

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSignalsSearchBuildsRequest(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL:    "https://altfins.test",
		APIKey:     "secret",
		AuthSource: "config",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got, want := r.URL.Path, "/api/v2/public/signals-feed/search-requests"; got != want {
			t.Fatalf("path mismatch: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("page"), "1"; got != want {
			t.Fatalf("page mismatch: got %q want %q", got, want)
		}
		if got, want := r.URL.Query().Get("size"), "25"; got != want {
			t.Fatalf("size mismatch: got %q want %q", got, want)
		}
		if got, want := r.Header.Get("X-Api-Key"), "secret"; got != want {
			t.Fatalf("api key mismatch: got %q want %q", got, want)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got, want := body["signalDirection"], "BULLISH"; got != want {
			t.Fatalf("body mismatch: got %v want %v", got, want)
		}

		payload, err := json.Marshal(map[string]any{
			"content": []map[string]any{
				{
					"symbol":     "BTC",
					"signalName": "Bullish MACD",
				},
			},
			"size":             25,
			"number":           1,
			"sort":             []map[string]any{},
			"totalElements":    1,
			"totalPages":       1,
			"numberOfElements": 1,
			"first":            false,
			"last":             true,
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader(payload)),
		}, nil
		})},
	})

	page, err := client.SignalsSearch(context.Background(), Paging{
		Page: 1,
		Size: 25,
		Sort: []string{"timestamp,desc"},
	}, map[string]any{
		"signalDirection": "BULLISH",
	})
	if err != nil {
		t.Fatalf("signals search: %v", err)
	}
	if got, want := len(page.Content), 1; got != want {
		t.Fatalf("content length mismatch: got %d want %d", got, want)
	}
}

func TestDryRunReturnsPreview(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL:    "https://altfins.test",
		APIKey:     "secret",
		AuthSource: "env",
		DryRun:     true,
	})

	_, err := client.MarketsSearch(context.Background(), Paging{Size: 10}, map[string]any{
		"symbols": []string{"BTC"},
	})
	dryRun, ok := IsDryRun(err)
	if !ok {
		t.Fatalf("expected dry run error, got %v", err)
	}
	if got, want := dryRun.Preview.Method, http.MethodPost; got != want {
		t.Fatalf("method mismatch: got %q want %q", got, want)
	}
	if got, want := dryRun.Preview.AuthSource, "env"; got != want {
		t.Fatalf("auth source mismatch: got %q want %q", got, want)
	}
}
