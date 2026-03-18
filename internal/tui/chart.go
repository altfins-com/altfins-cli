package tui

import (
	"strconv"

	"github.com/NimbleMarkets/ntcharts/sparkline"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

func renderSparkline(items []altfins.OHLCVData) string {
	if len(items) == 0 {
		return ""
	}
	values := make([]float64, 0, len(items))
	for _, item := range items {
		value, err := strconv.ParseFloat(item.Close, 64)
		if err != nil {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return ""
	}

	sl := sparkline.New(max(12, len(values)), 4)
	sl.PushAll(values)
	sl.Draw()
	return sl.View()
}
