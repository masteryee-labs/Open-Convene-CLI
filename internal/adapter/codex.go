// codex.go — Codex CLI adapter.
//
// CodexAdapter wraps the OpenAI Codex CLI. The `exec` subcommand runs
// non-interactively. The --sandbox flag controls file-system access:
//   - read-only:       the agent cannot write files (used for respond mode)
//   - workspace-write: the agent can write within the workspace (execute mode)
//   - danger-full-access: no sandbox (not used by default)
//
// read_only capability: "true"
//   - --sandbox read-only enforces read-only access. Safe for responder use.
//   - SupportsReadOnly() returns true.
//
// Command template (from config):
//   respond:  codex exec --sandbox read-only "{prompt}"
//   execute:  codex exec --sandbox workspace-write "{prompt}"
//
// This is the only adapter that MUST set ExecuteCommand differently from
// Command (different --sandbox flag values).
//
// Verified: codex --help and codex exec --help confirm:
//   - exec subcommand = "Run Codex non-interactively"
//   - --sandbox accepts read-only, workspace-write, danger-full-access
//   - PROMPT is a positional argument
//   - Requires OpenAI API key (auth via codex login)
//   - npm global install: npm install -g @openai/codex

package adapter

// CodexAdapter implements Adapter for the OpenAI Codex CLI.
type CodexAdapter struct {
	BaseAdapter
}
