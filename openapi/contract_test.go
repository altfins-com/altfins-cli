package openapi

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/altfins-com/altfins-cli/cmd"
)

// Compare the command discovery payload with every public v2 REST operation.
func TestPublicRESTCoverage(t *testing.T) {
	var spec struct {
		Paths map[string]map[string]json.RawMessage
	}
	b, err := os.ReadFile("altfins-openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &spec); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for path, ops := range spec.Paths {
		if strings.HasPrefix(path, "/api/v2/public/") {
			for method := range ops {
				want[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	if len(want) != 21 {
		t.Fatalf("snapshot has %d v2 operations", len(want))
	}
	cli := cmd.NewRootCommand()
	var out bytes.Buffer
	cli.SetOut(&out)
	cli.SetArgs([]string{"commands", "-o", "json"})
	if err := cli.Execute(); err != nil {
		t.Fatal(err)
	}
	type node struct {
		Endpoint struct{ Method, Path string }
		Children []node
	}
	var root node
	if err := json.Unmarshal(out.Bytes(), &root); err != nil {
		t.Fatal(err)
	}
	var walk func(node)
	walk = func(n node) {
		if e := n.Endpoint; e.Method == "GET" || e.Method == "POST" {
			key := e.Method + " " + e.Path
			if !want[key] {
				t.Errorf("unexpected operation %s", key)
			}
			delete(want, key)
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)
	if len(want) > 0 {
		t.Errorf("missing CLI commands for %v", want)
	}
}
