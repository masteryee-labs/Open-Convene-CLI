// kimi.go — Kimi Code CLI adapter.
//
// KimiAdapter wraps the Kimi Code CLI. The -p flag runs a non-interactive
// prompt. Kimi Code auto-approves read-only operations and does not modify
// files in non-interactive mode.
//
// read_only capability: "true"
//   - Read-only operations are auto-approved; file modifications are not
//     performed in -p mode. Safe for responder use.
//   - SupportsReadOnly() returns true.
//
// Command template (from config):
//   respond:  kimi -p "{prompt}"
//   execute:  kimi -p "{prompt}"  (same as respond; configurable via ExecuteCommand)
//
// Not installed on this system — command template based on task spec and
// Docs/03-Model-Adapters.md. Marked KIMI_UNVERIFIED in handoff.
// Install: curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash

package adapter

// KimiAdapter implements Adapter for the Kimi Code CLI.
type KimiAdapter struct {
	BaseAdapter
}
