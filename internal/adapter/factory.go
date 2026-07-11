// factory.go — Adapter factory function.
//
// GetAdapter maps a model name (from config map key) to the corresponding
// Adapter implementation. It populates BaseAdapter.Name from the map key
// (since ModelConfig.Name has yaml:"-" and is not parsed from YAML).
//
// Usage:
//   adapter, err := adapter.GetAdapter("codex", cfg.Models["codex"])
//   if err != nil { ... }
//   result, err := adapter.Respond(ctx, prompt, timeout)
//
// ConveneEngine uses AdapterFactory (a function type) for dependency injection,
// defaulting to GetAdapter. Tests can replace it with a mock factory (S6).

package adapter

import (
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// GetAdapter returns an Adapter instance for the given model name.
//
// For the 9 built-in CLI names (agy, codex, devin, grok, cursor, kimi, hermes,
// aider, opencode), it returns the corresponding typed adapter.
//
// For any other name (e.g. "glm-5.2", "swe-1.7", "kimi-k2.7"), it returns a
// generic BaseAdapter. This enables "single-CLI multi-model" configurations
// where multiple config entries use the same CLI (e.g. devin) with different
// --model flags. The generic adapter relies entirely on the command templates
// in ModelConfig (Command and ExecuteCommand) for execution.
//
// cfg is the ModelConfig for that model. cfg.Name is set to name by the
// factory (yaml:"-" prevents YAML parsing of the Name field; the map key
// is the source of truth).
func GetAdapter(name string, cfg config.ModelConfig) (Adapter, error) {
	// Populate Name from the map key (yaml:"-" means it's not parsed from YAML).
	cfg.Name = name

	// Build the shared BaseAdapter with the name and config.
	base := BaseAdapter{Name: name, Config: cfg}

	switch name {
	case "agy":
		return &AgyAdapter{base}, nil
	case "codex":
		return &CodexAdapter{base}, nil
	case "devin":
		return &DevinAdapter{base}, nil
	case "grok":
		return &GrokAdapter{base}, nil
	case "cursor":
		return &CursorAdapter{base}, nil
	case "kimi":
		return &KimiAdapter{base}, nil
	case "hermes":
		return &HermesAdapter{base}, nil
	case "aider":
		return &AiderAdapter{base}, nil
	case "opencode":
		return &OpenCodeAdapter{base}, nil
	default:
		// Generic adapter: uses command templates from config.
		// This supports custom model names (e.g. "glm-5.2") that map to
		// the same CLI with different --model flags.
		return &base, nil
	}
}
