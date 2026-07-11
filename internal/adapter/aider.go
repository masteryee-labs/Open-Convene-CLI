// aider.go — Aider adapter.
//
// AiderAdapter wraps the Aider code editor CLI. Aider is inherently a code
// editor that modifies files by default. The --yes flag auto-approves all
// operations (no interactive prompts), and --model specifies the LLM to use.
//
// read_only capability: "false"
//   - Aider is a code editor; it modifies files by design. It should NOT
//     be used as a responder (unless the user explicitly accepts the risk).
//   - SupportsReadOnly() returns false.
//
// Command template (from config):
//   respond:  (empty — aider does not support respond mode)
//   execute:  aider --yes --model sonnet "{prompt}"
//
// The --model value (e.g. "sonnet", "gpt-4o") is configured by the user in
// config/models.yaml execute_command, NOT hardcoded in the adapter.
//
// IMPORTANT: Respond() is overridden to return an error because aider is
// fundamentally a code editor and cannot serve as a read-only responder.
//
// Verified: aider not installed on this system. Command template based on
// task spec and Docs/03-Model-Adapters.md. Marked AIDER_UNVERIFIED in handoff.
// Install: python -m pip install aider-install && aider-install

package adapter

import (
	"context"
	"fmt"
)

// AiderAdapter implements Adapter for the Aider code editor CLI.
//
// Unlike other adapters, AiderAdapter does NOT support Respond mode.
// Calling Respond returns an error. Aider should only be used as an executor.
type AiderAdapter struct {
	BaseAdapter
}

// Respond returns an error because aider does not support read-only respond
// mode. Aider is fundamentally a code editor that modifies files.
//
// ConveneEngine should not assign aider as a responder or synthesizer.
func (a *AiderAdapter) Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error) {
	return AdapterResult{
		Stdout:     "",
		Stderr:     "aider does not support respond mode (read_only=false, it is a code editor)",
		ReturnCode: -1,
		Success:    false,
	}, fmt.Errorf("aider does not support respond mode")
}

// Execute runs aider in agentic mode with --yes (auto-approve) and the
// configured --model. The command template comes from Config.ExecuteCommand
// (or Config.Command as fallback).
func (a *AiderAdapter) Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error) {
	cmd := a.GetCommand(prompt, "execute")
	return RunCommand(ctx, cmd, timeout)
}
