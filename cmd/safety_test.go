package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

// runCLI executes the full command tree the way Execute() does, but returns the
// error instead of printing it, so tests can assert on typed errors.
func runCLI(t *testing.T, stdin []byte, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(bytes.NewReader(stdin))
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return out.String(), err
}

// isolatedConfig points config resolution at a throwaway directory and clears
// ambient altFINS environment variables so auth tests are hermetic.
func isolatedConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ALTFINS_API_KEY", "")
	t.Setenv("ALTFINS_BASE_URL", "")
}

func authHasKey(t *testing.T) bool {
	t.Helper()
	out, err := runCLI(t, nil, "auth", "status", "-o", "json")
	if err != nil {
		t.Fatalf("auth status failed: %v (out=%s)", err, out)
	}
	var status struct {
		HasAPIKey bool `json:"hasApiKey"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		t.Fatalf("parse auth status: %v (out=%s)", err, out)
	}
	return status.HasAPIKey
}

// runnableLeaves returns every executable leaf command (no real subcommands),
// excluding cobra's built-in help/completion commands.
func runnableLeaves() []*cobra.Command {
	root := NewRootCommand()
	var leaves []*cobra.Command
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		var realChildren []*cobra.Command
		for _, child := range c.Commands() {
			switch child.Name() {
			case "help", "completion":
				continue
			}
			realChildren = append(realChildren, child)
		}
		if len(realChildren) == 0 {
			if c.Runnable() {
				leaves = append(leaves, c)
			}
			return
		}
		for _, child := range realChildren {
			walk(child)
		}
	}
	walk(root)
	return leaves
}

func TestEveryRunnableLeafHasSafetyMetadata(t *testing.T) {
	valid := map[OperationType]bool{
		OpRead:        true,
		OpRemoteQuery: true,
		OpLocalWrite:  true,
		OpRemoteWrite: true,
		OpInteractive: true,
	}

	leaves := runnableLeaves()
	if len(leaves) == 0 {
		t.Fatal("expected at least one runnable leaf command")
	}

	for _, leaf := range leaves {
		name := leaf.CommandPath()
		s := safetyFor(leaf)
		if s == nil {
			t.Errorf("command %q has no safety metadata; every runnable command must declare it", name)
			continue
		}
		if !valid[s.OperationType] {
			t.Errorf("command %q has invalid operationType %q", name, s.OperationType)
		}

		// Consistency invariants between operationType and the boolean flags.
		switch s.OperationType {
		case OpInteractive:
			if s.DryRunSupported {
				t.Errorf("command %q is interactive but claims dryRunSupported", name)
			}
		case OpRemoteQuery:
			if s.MutatesRemoteState {
				t.Errorf("command %q is remote_query but claims mutatesRemoteState", name)
			}
			if s.MutatesLocalState {
				t.Errorf("command %q is remote_query but claims mutatesLocalState", name)
			}
			if !s.DryRunSupported {
				t.Errorf("command %q is remote_query but does not support dry-run", name)
			}
		case OpLocalWrite:
			if !s.MutatesLocalState {
				t.Errorf("command %q is local_write but does not set mutatesLocalState", name)
			}
		case OpRead:
			if s.MutatesLocalState || s.MutatesRemoteState {
				t.Errorf("command %q is read but claims to mutate state", name)
			}
		}

		if s.ConfirmationRequired && len(s.ConfirmationFlags) == 0 {
			t.Errorf("command %q requires confirmation but lists no confirmationFlags", name)
		}
	}
}

func TestAuthSetDryRunDoesNotWrite(t *testing.T) {
	isolatedConfig(t)
	out, err := runCLI(t, nil, "--dry-run", "auth", "set", "--api-key", "SECRET12345678", "-o", "json")
	if err != nil {
		t.Fatalf("auth set --dry-run failed: %v (out=%s)", err, out)
	}
	if authHasKey(t) {
		t.Fatal("auth set --dry-run must not write a key to config")
	}
}

func TestAuthClearDryRunDoesNotClear(t *testing.T) {
	isolatedConfig(t)
	if out, err := runCLI(t, nil, "auth", "set", "--api-key", "SEEDKEY1234567", "-o", "json"); err != nil {
		t.Fatalf("seed auth set failed: %v (out=%s)", err, out)
	}
	if !authHasKey(t) {
		t.Fatal("precondition failed: seeded key should be stored")
	}
	if out, err := runCLI(t, nil, "--dry-run", "auth", "clear", "-o", "json"); err != nil {
		t.Fatalf("auth clear --dry-run failed: %v (out=%s)", err, out)
	}
	if !authHasKey(t) {
		t.Fatal("auth clear --dry-run must not remove the stored key")
	}
}

func TestAuthClearRequiresConfirmationNonInteractive(t *testing.T) {
	isolatedConfig(t)
	if _, err := runCLI(t, nil, "auth", "set", "--api-key", "SEEDKEY1234567", "-o", "json"); err != nil {
		t.Fatalf("seed auth set failed: %v", err)
	}

	_, err := runCLI(t, nil, "auth", "clear", "-o", "json")
	if err == nil {
		t.Fatal("auth clear without --yes must fail in non-interactive mode")
	}
	var confErr *app.ConfirmationRequiredError
	if !errors.As(err, &confErr) {
		t.Fatalf("expected *app.ConfirmationRequiredError, got %T: %v", err, err)
	}
	if confErr.ExitCode() != 5 {
		t.Errorf("ConfirmationRequiredError exit code = %d, want 5", confErr.ExitCode())
	}
	if !authHasKey(t) {
		t.Fatal("auth clear must not remove the key when confirmation is missing")
	}

	if _, err := runCLI(t, nil, "auth", "clear", "--yes", "-o", "json"); err != nil {
		t.Fatalf("auth clear --yes failed: %v", err)
	}
	if authHasKey(t) {
		t.Fatal("auth clear --yes must remove the stored key")
	}
}

func TestAuthSetForceRequiredToReplace(t *testing.T) {
	isolatedConfig(t)
	if _, err := runCLI(t, nil, "auth", "set", "--api-key", "FIRSTKEY123456", "-o", "json"); err != nil {
		t.Fatalf("first auth set failed: %v", err)
	}

	_, err := runCLI(t, nil, "auth", "set", "--api-key", "SECONDKEY12345", "-o", "json")
	if err == nil {
		t.Fatal("replacing a stored key without --force must fail")
	}
	var confErr *app.ConfirmationRequiredError
	if !errors.As(err, &confErr) {
		t.Fatalf("expected *app.ConfirmationRequiredError, got %T: %v", err, err)
	}
	if confErr.ExitCode() != 5 {
		t.Errorf("ConfirmationRequiredError exit code = %d, want 5", confErr.ExitCode())
	}

	if _, err := runCLI(t, nil, "auth", "set", "--api-key", "SECONDKEY12345", "--force", "-o", "json"); err != nil {
		t.Fatalf("auth set --force failed: %v", err)
	}
}

// cmdNode mirrors the subset of `af commands -o json` we assert on.
type cmdNode struct {
	Command  string      `json:"command"`
	Safety   *safetyMeta `json:"safety"`
	Children []cmdNode   `json:"children"`
}

func TestCommandsJSONExposesSafety(t *testing.T) {
	isolatedConfig(t)
	out, err := runCLI(t, nil, "commands", "-o", "json")
	if err != nil {
		t.Fatalf("commands -o json failed: %v (out=%s)", err, out)
	}

	var tree cmdNode
	if err := json.Unmarshal([]byte(out), &tree); err != nil {
		t.Fatalf("parse commands JSON: %v (out=%s)", err, out)
	}

	byPath := map[string]*safetyMeta{}
	var walk func(cmdNode)
	walk = func(n cmdNode) {
		byPath[n.Command] = n.Safety
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(tree)

	check := func(path string, wantOp OperationType, wantDryRun bool) {
		s, ok := byPath[path]
		if !ok {
			t.Fatalf("command %q missing from commands output", path)
		}
		if s == nil {
			t.Fatalf("command %q has no safety metadata in JSON output", path)
		}
		if s.OperationType != wantOp {
			t.Errorf("command %q operationType = %q, want %q", path, s.OperationType, wantOp)
		}
		if s.DryRunSupported != wantDryRun {
			t.Errorf("command %q dryRunSupported = %v, want %v", path, s.DryRunSupported, wantDryRun)
		}
	}

	check("af signals list", OpRemoteQuery, true)
	check("af tui signals", OpInteractive, false)
	check("af auth status", OpRead, false)

	if s := byPath["af auth clear"]; s == nil || !s.ConfirmationRequired || len(s.ConfirmationFlags) == 0 {
		t.Errorf("af auth clear must advertise confirmationRequired with flags, got %+v", s)
	}
	if s := byPath["af auth set"]; s == nil || !s.ForceSupported {
		t.Errorf("af auth set must advertise forceSupported, got %+v", s)
	}
}
