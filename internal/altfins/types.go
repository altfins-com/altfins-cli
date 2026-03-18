package altfins

import "time"

type Paging struct {
	Page int
	Size int
	Sort []string
}

type OrderModel struct {
	Property  string `json:"property,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type Page[T any] struct {
	Size             int          `json:"size"`
	Number           int          `json:"number"`
	Sort             []OrderModel `json:"sort"`
	Content          []T          `json:"content"`
	TotalElements    int64        `json:"totalElements"`
	TotalPages       int          `json:"totalPages"`
	NumberOfElements int          `json:"numberOfElements"`
	First            bool         `json:"first"`
	Last             bool         `json:"last"`
}

type AssetInfo struct {
	Name         string `json:"name"`
	FriendlyName string `json:"friendlyName"`
}

type ValueType struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendlyName"`
}

type AnalyticsType struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendlyName"`
	IsNumerical  bool   `json:"isNumerical"`
}

type SignalLabel struct {
	NameBullish   string `json:"nameBullish,omitempty"`
	NameBearish   string `json:"nameBearish,omitempty"`
	TrendSensitive bool   `json:"trendSensitive,omitempty"`
	SignalType    string `json:"signalType,omitempty"`
	SignalKey     string `json:"signalKey,omitempty"`
}

type PermitsInfo struct {
	AvailablePermits        int64 `json:"availablePermits"`
	MonthlyAvailablePermits int64 `json:"monthlyAvailablePermits"`
}

type ScreenerSearchResult struct {
	Symbol         string                 `json:"symbol"`
	Name           string                 `json:"name"`
	LastPrice      string                 `json:"lastPrice"`
	AdditionalData map[string]any         `json:"additionalData"`
}

type SignalFeedItem struct {
	Timestamp  time.Time `json:"timestamp"`
	Direction  string    `json:"direction,omitempty"`
	SignalKey  string    `json:"signalKey,omitempty"`
	SignalName string    `json:"signalName,omitempty"`
	Symbol     string    `json:"symbol,omitempty"`
	LastPrice  string    `json:"lastPrice,omitempty"`
	MarketCap  string    `json:"marketCap,omitempty"`
	PriceChange string   `json:"priceChange,omitempty"`
	SymbolName string    `json:"symbolName,omitempty"`
}

type NewsSummary struct {
	MessageID  int64     `json:"messageId"`
	SourceID   int64     `json:"sourceId"`
	Content    string    `json:"content,omitempty"`
	Title      string    `json:"title,omitempty"`
	URL        string    `json:"url,omitempty"`
	SourceName string    `json:"sourceName,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type OHLCVData struct {
	Symbol string    `json:"symbol"`
	Time   time.Time `json:"time"`
	Open   string    `json:"open"`
	High   string    `json:"high"`
	Low    string    `json:"low"`
	Close  string    `json:"close"`
	Volume string    `json:"volume"`
}

type AnalyticsHistoryData struct {
	Symbol            string    `json:"symbol"`
	Time              time.Time `json:"time"`
	Value             string    `json:"value,omitempty"`
	NonNumericalValue string    `json:"nonNumericalValue,omitempty"`
}

type TechnicalAnalysisSummary struct {
	Symbol          string    `json:"symbol,omitempty"`
	FriendlyName    string    `json:"friendlyName,omitempty"`
	UpdatedDate     time.Time `json:"updatedDate"`
	NearTermOutlook string    `json:"nearTermOutlook,omitempty"`
	PatternType     string    `json:"patternType,omitempty"`
	PatternStage    string    `json:"patternStage,omitempty"`
	Description     string    `json:"description,omitempty"`
	ImgChartURL     string    `json:"imgChartUrl,omitempty"`
	ImgChartURLDark string    `json:"imgChartUrlDark,omitempty"`
	LogoURL         string    `json:"logoUrl,omitempty"`
}

type RequestPreview struct {
	Method     string      `json:"method"`
	URL        string      `json:"url"`
	Query      map[string][]string `json:"query,omitempty"`
	Body       any         `json:"body,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	AuthSource string      `json:"authSource,omitempty"`
}
