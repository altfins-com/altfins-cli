package cmd

import (
	"encoding/json"
	"testing"
)

// mcpPreview mirrors the dry-run RequestPreview shape for the MCP commands.
type mcpPreview struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		} `json:"params"`
	} `json:"body"`
	Headers map[string]string `json:"headers"`
}

func TestCalendarListDryRunBuildsToolCall(t *testing.T) {
	isolatedConfig(t)
	t.Setenv("ALTFINS_API_KEY", "TEST-DUMMY-KEY")
	out, err := runCLI(t, nil, "--dry-run", "calendar", "list",
		"--hot", "--symbols", "BTC,ETH", "--category", "AIRDROP",
		"--event-from", "today", "--sort-field", "dateEvent", "--sort-direction", "asc",
		"-o", "json")
	if err != nil {
		t.Fatalf("calendar list --dry-run failed: %v (out=%s)", err, out)
	}

	var preview mcpPreview
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatalf("parse dry-run preview: %v (out=%s)", err, out)
	}
	if preview.Body.Params.Name != "getCryptoCalendarEvents" {
		t.Errorf("tool name = %q, want getCryptoCalendarEvents", preview.Body.Params.Name)
	}
	if preview.Body.Method != "tools/call" {
		t.Errorf("rpc method = %q, want tools/call", preview.Body.Method)
	}
	args := preview.Body.Params.Arguments
	if args["assetSymbols"] != "BTC,ETH" {
		t.Errorf("assetSymbols = %v, want BTC,ETH", args["assetSymbols"])
	}
	if args["category"] != "AIRDROP" {
		t.Errorf("category = %v, want AIRDROP", args["category"])
	}
	if args["eventFrom"] != "today" {
		t.Errorf("eventFrom = %v, want today (raw passthrough)", args["eventFrom"])
	}
	if args["voteIsHot"] != true {
		t.Errorf("voteIsHot = %v, want true", args["voteIsHot"])
	}
	if args["sortDirection"] != "ASC" {
		t.Errorf("sortDirection = %v, want ASC (uppercased)", args["sortDirection"])
	}
	if preview.Headers["X-Api-Key"] != "redacted" {
		t.Errorf("X-Api-Key header = %q, want redacted", preview.Headers["X-Api-Key"])
	}
	if preview.Headers["User-Agent"] == "" {
		t.Error("User-Agent header must be set (WAF requires it)")
	}
}

func TestCalendarRejectsInvalidCategory(t *testing.T) {
	isolatedConfig(t)
	if _, err := runCLI(t, nil, "--dry-run", "calendar", "list", "--category", "BOGUS"); err == nil {
		t.Fatal("expected error for invalid --category value")
	}
}

func TestCalendarRejectsUnknownFilterKey(t *testing.T) {
	isolatedConfig(t)
	if _, err := runCLI(t, nil, "--dry-run", "calendar", "list", "--filter", `{"badKey":1}`); err == nil {
		t.Fatal("expected error for unknown --filter key (schema is additionalProperties:false)")
	}
}

func TestCalendarRawFilterValidatesEnum(t *testing.T) {
	isolatedConfig(t)
	// An invalid enum value supplied via raw --filter must be rejected too, not
	// only via the first-class --category flag.
	if _, err := runCLI(t, nil, "--dry-run", "calendar", "list", "--filter", `{"category":"NOPE"}`); err == nil {
		t.Fatal("expected error for invalid enum value in raw --filter")
	}
}

func TestCalendarRawFilterRejectsArrayForStringParam(t *testing.T) {
	isolatedConfig(t)
	// assetSymbols is a string (CSV) per the MCP schema; an array must be rejected.
	if _, err := runCLI(t, nil, "--dry-run", "calendar", "list", "--filter", `{"assetSymbols":["BTC","ETH"]}`); err == nil {
		t.Fatal("expected error for array value on a string-typed param")
	}
}

func TestCalendarCategoryCaseInsensitive(t *testing.T) {
	isolatedConfig(t)
	t.Setenv("ALTFINS_API_KEY", "TEST-DUMMY-KEY")
	out, err := runCLI(t, nil, "--dry-run", "calendar", "list", "--category", "airdrop", "-o", "json")
	if err != nil {
		t.Fatalf("lowercase --category should be accepted: %v (out=%s)", err, out)
	}
	var preview mcpPreview
	if err := json.Unmarshal([]byte(out), &preview); err != nil {
		t.Fatalf("parse preview: %v", err)
	}
	if preview.Body.Params.Arguments["category"] != "AIRDROP" {
		t.Errorf("category should be normalized to AIRDROP, got %v", preview.Body.Params.Arguments["category"])
	}
}

func TestCalendarKeylessDryRun(t *testing.T) {
	isolatedConfig(t) // no API key
	if _, err := runCLI(t, nil, "--dry-run", "calendar", "list", "-o", "json"); err != nil {
		t.Fatalf("dry-run must work without an API key, got: %v", err)
	}
}

func TestCalendarCategoriesIsLocal(t *testing.T) {
	isolatedConfig(t)
	out, err := runCLI(t, nil, "calendar", "categories", "-o", "json")
	if err != nil {
		t.Fatalf("calendar categories failed: %v (out=%s)", err, out)
	}
	var got []string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse categories: %v (out=%s)", err, out)
	}
	if len(got) != len(calendarCategories) {
		t.Errorf("got %d categories, want %d", len(got), len(calendarCategories))
	}
}
