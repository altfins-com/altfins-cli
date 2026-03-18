package altfins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ClientConfig struct {
	BaseURL    string
	APIKey     string
	AuthSource string
	DryRun     bool
	HTTPClient *http.Client
}

type Client struct {
	baseURL    string
	apiKey     string
	authSource string
	dryRun     bool
	httpClient *http.Client
}

type APIError struct {
	StatusCode int             `json:"statusCode"`
	Method     string          `json:"method"`
	URL        string          `json:"url"`
	Body       json.RawMessage `json:"body,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("altFINS API error: %s %s returned %d", e.Method, e.URL, e.StatusCode)
}

func (e *APIError) ExitCode() int {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return 3
	default:
		return 4
	}
}

type DryRunError struct {
	Preview RequestPreview
}

func (e *DryRunError) Error() string {
	return "dry run"
}

func NewClient(cfg ClientConfig) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://altfins.com"
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		authSource: strings.TrimSpace(cfg.AuthSource),
		dryRun:     cfg.DryRun,
		httpClient: httpClient,
	}
}

func IsDryRun(err error) (*DryRunError, bool) {
	var out *DryRunError
	if errors.As(err, &out) {
		return out, true
	}
	return nil, false
}

func (p Paging) apply(values url.Values) {
	if p.Page > 0 {
		values.Set("page", fmt.Sprintf("%d", p.Page))
	}
	if p.Size > 0 {
		values.Set("size", fmt.Sprintf("%d", p.Size))
	}
	for _, item := range p.Sort {
		item = strings.TrimSpace(item)
		if item != "" {
			values.Add("sort", item)
		}
	}
}

func (c *Client) Symbols(ctx context.Context) ([]AssetInfo, error) {
	var out []AssetInfo
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/symbols", nil, nil, &out)
}

func (c *Client) Intervals(ctx context.Context) ([]string, error) {
	var out []string
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/intervals", nil, nil, &out)
}

func (c *Client) AvailablePermits(ctx context.Context) (int64, error) {
	var out int64
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/available-permits", nil, nil, &out)
}

func (c *Client) MonthlyAvailablePermits(ctx context.Context) (int64, error) {
	var out int64
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/monthly-available-permits", nil, nil, &out)
}

func (c *Client) AllAvailablePermits(ctx context.Context) (PermitsInfo, error) {
	var out PermitsInfo
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/all-available-permits", nil, nil, &out)
}

func (c *Client) MarketsFields(ctx context.Context) ([]ValueType, error) {
	var out []ValueType
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/screener-data/value-types", nil, nil, &out)
}

func (c *Client) SignalKeys(ctx context.Context) ([]SignalLabel, error) {
	var out []SignalLabel
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/signals-feed/signal-keys", nil, nil, &out)
}

func (c *Client) AnalyticsTypes(ctx context.Context) ([]AnalyticsType, error) {
	var out []AnalyticsType
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/analytics/types", nil, nil, &out)
}

func (c *Client) MarketsSearch(ctx context.Context, paging Paging, body any) (Page[ScreenerSearchResult], error) {
	var out Page[ScreenerSearchResult]
	q := url.Values{}
	paging.apply(q)
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/screener-data/search-requests", q, body, &out)
}

func (c *Client) OHLCVSnapshot(ctx context.Context, body any) ([]OHLCVData, error) {
	var out []OHLCVData
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/ohlcv/snapshot-requests", nil, body, &out)
}

func (c *Client) OHLCVHistory(ctx context.Context, paging Paging, body any) (Page[OHLCVData], error) {
	var out Page[OHLCVData]
	q := url.Values{}
	paging.apply(q)
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/ohlcv/history-requests", q, body, &out)
}

func (c *Client) AnalyticsHistory(ctx context.Context, paging Paging, body any) (Page[AnalyticsHistoryData], error) {
	var out Page[AnalyticsHistoryData]
	q := url.Values{}
	paging.apply(q)
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/analytics/search-requests", q, body, &out)
}

func (c *Client) TechnicalAnalysis(ctx context.Context, paging Paging, symbol string) (Page[TechnicalAnalysisSummary], error) {
	var out Page[TechnicalAnalysisSummary]
	q := url.Values{}
	paging.apply(q)
	if strings.TrimSpace(symbol) != "" {
		q.Set("symbol", strings.TrimSpace(symbol))
	}
	return out, c.do(ctx, http.MethodGet, "/api/v2/public/technical-analysis/data", q, nil, &out)
}

func (c *Client) SignalsSearch(ctx context.Context, paging Paging, body any) (Page[SignalFeedItem], error) {
	var out Page[SignalFeedItem]
	q := url.Values{}
	paging.apply(q)
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/signals-feed/search-requests", q, body, &out)
}

func (c *Client) NewsSearch(ctx context.Context, paging Paging, body any) (Page[NewsSummary], error) {
	var out Page[NewsSummary]
	q := url.Values{}
	paging.apply(q)
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/news-summary/search-requests", q, body, &out)
}

func (c *Client) NewsGet(ctx context.Context, messageID, sourceID int64) (NewsSummary, error) {
	var out NewsSummary
	q := url.Values{}
	q.Set("MessageId", fmt.Sprintf("%d", messageID))
	q.Set("SourceId", fmt.Sprintf("%d", sourceID))
	return out, c.do(ctx, http.MethodPost, "/api/v2/public/news-summary/find-summary", q, nil, &out)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	bodyBytes, preview, err := c.prepare(method, path, query, body)
	if err != nil {
		return err
	}
	if c.dryRun {
		return &DryRunError{Preview: preview}
	}

	for attempt := range 3 {
		req, err := http.NewRequestWithContext(ctx, method, preview.URL, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if len(bodyBytes) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.apiKey != "" {
			req.Header.Set("X-Api-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("execute request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt < 2 {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
				continue
			}
		}

		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			payload, _ := io.ReadAll(resp.Body)
			return &APIError{
				StatusCode: resp.StatusCode,
				Method:     method,
				URL:        preview.URL,
				Body:       payload,
			}
		}
		if out == nil {
			io.Copy(io.Discard, resp.Body)
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		return nil
	}

	return fmt.Errorf("request failed after retries")
}

func (c *Client) prepare(method, path string, query url.Values, body any) ([]byte, RequestPreview, error) {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, RequestPreview{}, fmt.Errorf("marshal request body: %w", err)
		}
		bodyBytes = payload
	}

	headers := map[string]string{
		"Accept": "application/json",
	}
	if len(bodyBytes) > 0 {
		headers["Content-Type"] = "application/json"
	}
	if c.apiKey != "" {
		headers["X-Api-Key"] = "redacted"
	}

	preview := RequestPreview{
		Method:     method,
		URL:        fullURL,
		Query:      map[string][]string(query),
		Body:       body,
		Headers:    headers,
		AuthSource: c.authSource,
	}
	return bodyBytes, preview, nil
}
