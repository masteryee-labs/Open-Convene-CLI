// devin.go — Devin CLI adapter.
//
// DevinAdapter wraps the Devin CLI. The -p / --print flag runs in
// non-interactive mode: processes the prompt and exits, printing the response
// to stdout.
//
// read_only capability: "maybe"
//   - The -p flag provides non-interactive print mode, but Devin is
//     inherently an agentic AI. Print mode may still trigger read-only tools.
//   - SupportsReadOnly() returns false ("maybe" is not "true").
//
// Command template (from config):
//   respond:  devin -p "{prompt}"
//   execute:  devin --permission-mode dangerous "{prompt}"
//     (NOTE: the task spec suggested "bypass" but actual valid permission modes
//      are: auto, accept-edits, smart, dangerous. "dangerous" auto-approves all
//      tools, which is the closest to "bypass". Users should configure the
//      exact mode in config/models.yaml execute_command.)
//
// Verified: devin --help confirms:
//   - -p, --print [<PROMPT>] = "Print response and exit" (non-interactive)
//   - --permission-mode accepts: auto, accept-edits, smart, dangerous
//   - "dangerous" = auto-approve all tools
//   - Requires Devin account login (devin auth / devin setup)
//   - Install: curl -fsSL https://cli.devin.ai/install.sh | bash
//
// Known issue: task spec referenced "--permission-mode bypass" which is NOT
// a valid Devin permission mode. The config should use "dangerous" for
// auto-approving all tools. See handoff/S2.md for details.

package adapter

// DevinAdapter implements Adapter for the Devin CLI.
type DevinAdapter struct {
	BaseAdapter
}
