package altfins

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const PublicAPI = "/api/v2/public/"
const PresetPath = PublicAPI + "screener-data/preset-filter"
const SetupsPath = PublicAPI + "ai-trade-setup/"
const PatternsPath = PublicAPI + "patterns-data/search-requests"

// Record preserves optional and nested fields as delivered by the provider.
type Record map[string]any

func (c *Client) MarketsPreset(ctx context.Context, p Paging, body map[string]any) (Page[ScreenerSearchResult], error) {
	q := url.Values{}
	p.apply(q)
	var out Page[ScreenerSearchResult]
	return out, c.do(ctx, http.MethodPost, PresetPath, q, body, &out)
}
func (c *Client) Patterns(ctx context.Context, p Paging, body map[string]any) (Page[map[string]any], error) {
	q := url.Values{}
	p.apply(q)
	var out Page[map[string]any]
	return out, c.do(ctx, http.MethodPost, PatternsPath, q, body, &out)
}
func (c *Client) Setups(ctx context.Context, p Paging, q url.Values) (Page[map[string]any], error) {
	p.apply(q)
	var out Page[map[string]any]
	return out, c.do(ctx, http.MethodGet, SetupsPath+"search-requests", q, nil, &out)
}
func (c *Client) SetupReport(ctx context.Context, id int64) (Record, error) {
	var out Record
	return out, c.do(ctx, http.MethodGet, fmt.Sprintf("%s%d/report", SetupsPath, id), nil, nil, &out)
}

func (c *Client) GenerateSetup(ctx context.Context, symbol, interval string) (Record, error) {
	client := *c.httpClient
	if client.Timeout < 60*time.Second {
		client.Timeout = 65 * time.Second
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	// net/http can transparently replay GET on reused connections or HTTP/2 streams.
	// A fresh HTTP/1 transport gives the billed operation a single connection attempt.
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if t, ok := transport.(*http.Transport); ok {
		one := t.Clone()
		one.DisableKeepAlives = true
		one.Protocols = new(http.Protocols)
		one.Protocols.SetHTTP1(true)
		client.Transport = one
		defer one.CloseIdleConnections()
	}
	q := url.Values{"symbol": {symbol}, "timeInterval": {interval}}
	var out Record
	return out, c.doRequest(ctx, http.MethodGet, SetupsPath+"generate", q, nil, &out, &client, 1)
}
