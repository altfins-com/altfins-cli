package altfins

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBilledGETIsSentOnce(t *testing.T) {
	for _, status := range []int{0, 200, 202, 307, 429, 503} {
		calls := 0
		c := NewClient(ClientConfig{HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) < 59*time.Second || r.Method != "GET" || r.URL.Query().Get("symbol") != "SOL" {
				t.Errorf("request %v deadline %v", r.URL, deadline)
			}
			if status == 0 {
				return nil, context.DeadlineExceeded
			}
			payload := `{"id":23,"status":"NEW","reused":true}`
			if status == 202 {
				payload = `{"id":23,"status":"PENDING"}`
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Location": []string{"/another"}}, Body: io.NopCloser(strings.NewReader(payload))}, nil
		})}})
		row, err := c.GenerateSetup(context.Background(), "SOL", "DAILY")
		if calls != 1 || (err == nil) != (status == 200 || status == 202) {
			t.Fatal(status, calls, row, err)
		}
		if status == 202 && row["status"] != "PENDING" || status == 200 && row["reused"] != true {
			t.Fatal(row)
		}
	}
}
func TestQuotaWireFormats(t *testing.T) {
	for _, tc := range []struct {
		payload string
		monthly int64
		missing bool
	}{{"0", 0, false}, {"19", 19, false}, {`{"monthlyAvailablePermits":0,"monthlyAvailableAiTradeSetupPrompts":7}`, 0, false}, {`{}`, 0, true}} {
		c := NewClient(ClientConfig{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.payload))}, nil
		})}})
		got, err := c.MonthlyAvailablePermits(context.Background())
		if (err != nil) != tc.missing || got != tc.monthly {
			t.Fatal(tc, got, err)
		}
	}
}
func TestAuthenticatedRedirectPolicy(t *testing.T) {
	origin, _ := url.Parse("https://example.test/start")
	for _, client := range []*http.Client{NewClient(ClientConfig{}).httpClient, NewMCPClient(MCPClientConfig{}).httpClient} {
		for _, target := range []string{"http://example.test/end", "https://other.test/end", "https://example.test:42/end", "https://example.test/end"} {
			u, _ := url.Parse(target)
			err := client.CheckRedirect(&http.Request{URL: u}, []*http.Request{{URL: origin}})
			if errors.Is(err, http.ErrUseLastResponse) != (u.Host != origin.Host || u.Scheme != origin.Scheme) {
				t.Fatal(target, err)
			}
		}
	}
}
