// agy.go — Antigravity CLI adapter.
//
// AgyAdapter wraps the Antigravity CLI (agy). The -p / --print flag runs a
// single prompt non-interactively and prints the response to stdout.
//
// read_only capability: "maybe"
//   - The -p flag provides a non-interactive print mode, but agy is
//     inherently an agentic AI. In print mode it may still execute tools,
//     so it is NOT guaranteed read-only.
//   - SupportsReadOnly() returns false ("maybe" is not "true").
//
// Command template (from config):
//   respond:  agy -p "{prompt}"
//   execute:  agy -p "{prompt}"  (same as respond; configurable via ExecuteCommand)
//
// Verified: agy --help confirms -p / --print = "Run a single prompt
// non-interactively and print the response".
// Additional flags observed: --dangerously-skip-permissions (auto-approve all
// tool requests), --sandbox (terminal restrictions), --mode (accept-edits/plan).
// These can be added via config extra_args if needed.

package adapter

// AgyAdapter implements Adapter for the Antigravity CLI (agy).
type AgyAdapter struct {
	BaseAdapter
}
