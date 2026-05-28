package cmd

import (
	"encoding/json"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// OperationType classifies what a command does, so that autonomous agents can
// decide how to call it safely without parsing help prose.
type OperationType string

const (
	// OpRead is a local, non-mutating read (e.g. inspect local config or command metadata).
	OpRead OperationType = "read"
	// OpRemoteQuery reads data from the altFINS API. It mutates nothing and is safe to --dry-run.
	OpRemoteQuery OperationType = "remote_query"
	// OpLocalWrite mutates local state such as the stored config / API key.
	OpLocalWrite OperationType = "local_write"
	// OpRemoteWrite mutates remote state via the API. No such command exists yet; reserved for future use.
	OpRemoteWrite OperationType = "remote_write"
	// OpInteractive launches a long-running interactive UI (TUI). It cannot be --dry-run.
	OpInteractive OperationType = "interactive"
)

const annotationSafety = "altfins:safety"

// safetyMeta is the machine-readable safety contract exposed through
// `af commands -o json`. The field set is intentionally small and
// safety-focused rather than a general policy engine.
type safetyMeta struct {
	OperationType          OperationType `json:"operationType"`
	MutatesLocalState      bool          `json:"mutatesLocalState"`
	MutatesRemoteState     bool          `json:"mutatesRemoteState"`
	DryRunSupported        bool          `json:"dryRunSupported"`
	ConfirmationRequired   bool          `json:"confirmationRequired"`
	ConfirmationFlags      []string      `json:"confirmationFlags,omitempty"`
	ForceSupported         bool          `json:"forceSupported"`
	ForceRequiredToReplace bool          `json:"forceRequiredToReplace,omitempty"`
	// ConfirmationExitCode is the exit code the command returns when a required
	// confirmation/force flag is missing in non-interactive mode (0 when N/A).
	ConfirmationExitCode int `json:"confirmationExitCode,omitempty"`
}

// annotateSafety records the safety contract for a command. It is stored as a
// JSON-encoded Cobra annotation, mirroring the existing endpoint annotations,
// so there is no parallel registry to keep in sync with the command tree.
func annotateSafety(cmd *cobra.Command, meta safetyMeta) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		// safetyMeta only holds JSON-safe scalars and a string slice, so
		// marshaling cannot realistically fail. Leave the annotation unset
		// rather than panicking if it ever does.
		return
	}
	cmd.Annotations[annotationSafety] = string(payload)
}

// safetyFor decodes the safety contract for a command, or nil if none was set.
func safetyFor(cmd *cobra.Command) *safetyMeta {
	raw, ok := cmd.Annotations[annotationSafety]
	if !ok || raw == "" {
		return nil
	}
	var meta safetyMeta
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return &meta
}

// safetyForOrDefault returns the command's safety contract, defaulting any
// un-annotated node to a local read. Only navigational/help nodes are
// un-annotated (the root command, command groups, and cobra's generated
// help/completion commands); they perform no operation, so classifying them as
// `read` keeps every command emitted by `af commands` carrying an explicit
// safety contract. Real operations are always annotated explicitly, and
// cmd/safety_test.go enforces that on every runnable leaf.
func safetyForOrDefault(cmd *cobra.Command) *safetyMeta {
	if s := safetyFor(cmd); s != nil {
		return s
	}
	return &safetyMeta{OperationType: OpRead}
}

// markRemoteQuery annotates a networked read command: it sets the endpoint
// metadata and a remote_query safety contract that is safe to --dry-run.
func markRemoteQuery(cmd *cobra.Command, method, path string) {
	annotateEndpoint(cmd, method, path)
	annotateSafety(cmd, safetyMeta{
		OperationType:   OpRemoteQuery,
		DryRunSupported: true,
	})
}

// markInteractive annotates a TUI command. Interactive UIs cannot be dry-run.
func markInteractive(cmd *cobra.Command, path string) {
	annotateEndpoint(cmd, "TUI", path)
	annotateSafety(cmd, safetyMeta{
		OperationType:   OpInteractive,
		DryRunSupported: false,
	})
}

// markLocalRead annotates a command that only reads local state.
func markLocalRead(cmd *cobra.Command) {
	annotateSafety(cmd, safetyMeta{
		OperationType:   OpRead,
		DryRunSupported: false,
	})
}

// isInteractiveStdin reports whether the command's stdin is a real terminal. It
// decides whether a destructive command may prompt for confirmation (human at a
// TTY) or must require an explicit confirmation flag (automation, agents). All
// non-terminal stdin — a pipe, a redirected file, or /dev/null — is treated as
// non-interactive. A plain ModeCharDevice check is not enough because /dev/null
// is itself a character device that is not a terminal, so we use a proper isatty
// check. A non-*os.File reader (e.g. a test buffer) is also non-interactive.
func isInteractiveStdin(cmd *cobra.Command) bool {
	file, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	fd := file.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
