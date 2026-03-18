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
