// hermes.go — Hermes Agent CLI adapter.
//
// HermesAdapter wraps the Hermes Agent CLI. The `chat` subcommand with the
// -q flag runs a single query (non-interactive) and prints the response.
//
// read_only capability: "maybe"
//   - The chat -q flag provides a single-query non-interactive mode, but
//     Hermes is inherently agentic. It may still execute tools.
//   - SupportsReadOnly() returns false ("maybe" is not "true").
//
// Command template (from config):
//   respond:  hermes chat -q "{prompt}"
//   execute:  hermes chat -q "{prompt}"  (same as respond; configurable)
//
// Not installed on this system — command template based on task spec and
// Docs/03-Model-Adapters.md. Marked HERMES_UNVERIFIED in handoff.
// Install: hermes setup --portal (see hermes-agent.nousresearch.com)

package adapter

// HermesAdapter implements Adapter for the Hermes Agent CLI.
type HermesAdapter struct {
	BaseAdapter
}
