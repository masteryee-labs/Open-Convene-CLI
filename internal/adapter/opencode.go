// opencode.go — OpenCode CLI adapter.
//
// OpenCodeAdapter wraps the OpenCode CLI. The `run` subcommand executes a
// single prompt non-interactively.
//
// read_only capability: "maybe"
//   - The run subcommand provides non-interactive execution, but OpenCode
//     is inherently agentic. It may still execute tools with side effects.
//   - SupportsReadOnly() returns false ("maybe" is not "true").
//
// Command template (from config):
//   respond:  opencode run "{prompt}"
//   execute:  opencode run "{prompt}"  (same as respond; configurable)
//
// Not installed on this system — command template based on task spec and
// Docs/03-Model-Adapters.md. Marked OPENCODE_UNVERIFIED in handoff.
// Install: see https://opencode.ai/docs/cli/

package adapter

// OpenCodeAdapter implements Adapter for the OpenCode CLI.
type OpenCodeAdapter struct {
	BaseAdapter
}
