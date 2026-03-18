package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func LoadJSONObject(filter string, stdinJSON bool, stdin io.Reader) (map[string]any, error) {
	switch {
	case strings.TrimSpace(filter) != "":
		return parseFilterSource(filter)
	case stdinJSON:
		var out map[string]any
		if err := json.NewDecoder(stdin).Decode(&out); err != nil {
			return nil, fmt.Errorf("decode stdin json: %w", err)
		}
		return out, nil
	default:
		return map[string]any{}, nil
	}
}

func MergeJSON(base, overlay map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func NormalizeTimeInput(value string, upperBound bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err != nil {
			continue
		}
		if layout == "2006-01-02" {
			if upperBound {
				parsed = parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			}
		}
		return parsed.UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("unsupported datetime value %q", value)
}

func parseFilterSource(source string) (map[string]any, error) {
	var data []byte
	var err error
	if strings.HasPrefix(source, "@") {
		data, err = os.ReadFile(strings.TrimPrefix(source, "@"))
		if err != nil {
			return nil, fmt.Errorf("read filter file: %w", err)
		}
	} else {
		data = []byte(source)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode filter json: %w", err)
	}
	return out, nil
}
