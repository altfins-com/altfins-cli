package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/altfins"
	"github.com/altfins-com/altfins-cli/internal/app"
)

const (
	annotationEndpointMethod = "altfins:method"
	annotationEndpointPath   = "altfins:path"
)

type pagingFlags struct {
	page int
	size int
	sort []string
}

type jsonBodyFlags struct {
	filter    string
	stdinJSON bool
}

func (f *pagingFlags) bind(cmd *cobra.Command) {
	cmd.Flags().IntVar(&f.page, "page", 0, "Zero-based page index")
	cmd.Flags().IntVar(&f.size, "size", 0, "Page size")
	cmd.Flags().StringSliceVar(&f.sort, "sort", nil, "Sort expressions (repeatable)")
}

func (f *pagingFlags) value() altfins.Paging {
	return altfins.Paging{
		Page: f.page,
		Size: f.size,
		Sort: f.sort,
	}
}

func (f *jsonBodyFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.filter, "filter", "", "JSON filter or @path/to/filter.json")
	cmd.Flags().BoolVar(&f.stdinJSON, "stdin-json", false, "Read filter JSON from stdin")
}

func loadBodyFlags(cmd *cobra.Command, flags jsonBodyFlags) (map[string]any, error) {
	return app.LoadJSONObject(flags.filter, flags.stdinJSON, cmd.InOrStdin())
}

func csvValues(value string) []string {
	return app.ParseCSV(value)
}

func factoryFor(cmd *cobra.Command) (*app.Factory, error) {
	factory := app.FactoryFromContext(cmd.Context())
	if factory == nil {
		return nil, fmt.Errorf("internal error: command factory not initialized")
	}
	return factory, nil
}

func clientFor(cmd *cobra.Command) (*altfins.Client, error) {
	factory, err := factoryFor(cmd)
	if err != nil {
		return nil, err
	}
	return factory.NewClient()
}

func mcpClientFor(cmd *cobra.Command) (*altfins.MCPClient, error) {
	factory, err := factoryFor(cmd)
	if err != nil {
		return nil, err
	}
	return factory.NewMCPClient()
}

// mcpListValue normalizes a decoded MCP tool payload into the shape the output
// layer renders best: a list of objects when the payload is an array (or an
// object wrapping a `content`/`data`/`items` array), otherwise the object (or
// the raw value) as-is. It keeps unknown response shapes printable without
// pinning typed structs the server has not been sampled against yet.
func mcpListValue(raw json.RawMessage) any {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return []map[string]any{}
	}
	var asList []map[string]any
	if err := json.Unmarshal(raw, &asList); err == nil {
		return asList
	}
	var asObject map[string]any
	if err := json.Unmarshal(raw, &asObject); err == nil {
		for _, key := range []string{"content", "data", "items", "results"} {
			if inner, ok := asObject[key].([]any); ok {
				return toMapSlice(inner)
			}
		}
		return asObject
	}
	var anyVal any
	if err := json.Unmarshal(raw, &anyVal); err == nil {
		return anyVal
	}
	return trimmed
}

func toMapSlice(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		} else {
			out = append(out, map[string]any{"value": item})
		}
	}
	return out
}

func handleResult(cmd *cobra.Command, data any, err error) error {
	factory, factoryErr := factoryFor(cmd)
	if factoryErr != nil {
		return factoryErr
	}
	return factory.HandleCommandResult(data, err)
}

func annotateEndpoint(cmd *cobra.Command, method, path string) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[annotationEndpointMethod] = method
	cmd.Annotations[annotationEndpointPath] = path
}

func endpointFor(cmd *cobra.Command) map[string]string {
	method := cmd.Annotations[annotationEndpointMethod]
	path := cmd.Annotations[annotationEndpointPath]
	if method == "" || path == "" {
		return nil
	}
	return map[string]string{
		"method": method,
		"path":   path,
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func writePlainTable(w io.Writer, headers []string, rows [][]string) error {
	return app.WriteOutput(w, []map[string]any{
		{"_headers": strings.Join(headers, ","), "_rows": mustJSON(rows)},
	}, "json", nil)
}
