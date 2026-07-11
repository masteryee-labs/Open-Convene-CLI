// grok.go — Grok CLI adapter.
//
// GrokAdapter wraps the Grok CLI. The -p / --single flag runs a single-turn
// prompt, prints the response to stdout, and exits.
//
// read_only capability: "maybe"
//   - The -p flag provides non-interactive single-turn mode, but Grok is
//     inherently agentic. Single-turn mode may still execute tools.
//   - SupportsReadOnly() returns false ("maybe" is not "true").
//
// Command template (from config):
//   respond:  grok -p "{prompt}"
//   execute:  grok -p "{prompt}"  (same as respond; configurable via ExecuteCommand)
//
// Verified: grok --help confirms:
//   - -p, --single <PROMPT> = "Single-turn prompt. Prints the response to
//     stdout and exits"
//   - --always-approve = auto-approve all tool executions (for execute mode)
//   - --permission-mode accepts: default, acceptEdits, auto, dontAsk,
//     bypassPermissions, plan
//   - --max-turns limits agent turns
//   - Install: curl -fsSL https://x.ai/cli/install.sh | bash

package adapter

// GrokAdapter implements Adapter for the Grok CLI.
type GrokAdapter struct {
	BaseAdapter
}
