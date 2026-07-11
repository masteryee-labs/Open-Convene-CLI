// cursor.go — Cursor CLI adapter.
//
// CursorAdapter wraps the Cursor CLI. The `agent` subcommand runs in
// non-interactive agent mode. The --force flag controls whether the agent
// can automatically modify files:
//   - Without --force: read-only (the agent will not auto-edit files)
//   - With --force:    the agent can automatically apply changes
//
// read_only capability: "true"
//   - Without --force, the agent operates in read-only mode. Safe for
//     responder use.
//   - SupportsReadOnly() returns true.
//
// Command template (from config):
//   respond:  cursor agent -p "{prompt}"
//   execute:  cursor agent -p --force "{prompt}"
//
// NOTE: The task spec's execute template is `cursor agent -p --force "{prompt}"`.
// The --force flag allows automatic file modifications. The exact flag syntax
// may need verification once Cursor CLI is installed.
//
// Not installed on this system — command template based on task spec and
// Docs/03-Model-Adapters.md. Marked CURSOR_UNVERIFIED in handoff.
// Install: curl https://cursor.com/install -fsS | bash

package adapter

// CursorAdapter implements Adapter for the Cursor CLI.
type CursorAdapter struct {
	BaseAdapter
}
