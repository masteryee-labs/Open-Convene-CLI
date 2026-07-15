// fallback.go — Fallback chain resolution (P1, omnilane-inspired).
//
// When a model's CLI is not installed or its adapter cannot be created, the
// engine walks the model's Fallback chain (config.ModelConfig.Fallback) and
// uses the first candidate that resolves successfully. This lets a single
// subscription work: if "codex" is not installed but "devin:glm-5.2" is, a
// model configured with `fallback: [devin:glm-5.2]` degrades gracefully.
//
// Cycle safety: a visited-set bounds recursion. If every candidate in the
// chain fails (including the primary), resolveWithFallback returns a nil
// adapter and the caller skips that role (responder) or records an error
// (synthesizer/executor), preserving the existing fault-tolerance semantics.
package convene

import (
	"fmt"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// resolveWithFallback resolves a model name to a usable adapter, walking the
// Fallback chain if the primary fails.
//
// It returns:
//   - the adapter (nil if all candidates failed)
//   - the resolved model name (may differ from the input if a fallback was used)
//   - the ModelConfig of the resolved model
//   - the updated warnings slice (append-only; the input slice is not mutated)
//
// The role parameter ("responder" | "synthesizer" | "executor") is used only
// for warning messages.
func (e *ConveneEngine) resolveWithFallback(
	name string,
	role string,
	warnings []string,
) (adapter.Adapter, string, config.ModelConfig, []string) {
	visited := make(map[string]bool)
	return e.resolveChain(name, role, visited, warnings)
}

// resolveChain is the recursive worker for resolveWithFallback. The visited
// set prevents infinite loops if two models list each other as fallbacks.
func (e *ConveneEngine) resolveChain(
	name string,
	role string,
	visited map[string]bool,
	warnings []string,
) (adapter.Adapter, string, config.ModelConfig, []string) {
	if visited[name] {
		warnings = append(warnings, fmt.Sprintf(
			"%s %s: fallback cycle detected, skipping", role, name))
		return nil, name, config.ModelConfig{}, warnings
	}
	visited[name] = true

	// Resolve config: config map first, then dynamic model name.
	modelCfg, exists := e.Config.Models[name]
	if !exists {
		dynCfg, _, dynOk := adapter.ResolveDynamicModel(name)
		if !dynOk {
			// Primary not resolvable — try its fallback chain if it has one
			// (only possible if it's in the config map with a Fallback list).
			// Since it's not in the map, there's no Fallback to walk.
			warnings = append(warnings, fmt.Sprintf(
				"%s %s: not found in config models and not a dynamic model name",
				role, name))
			return nil, name, config.ModelConfig{}, warnings
		}
		modelCfg = dynCfg
	}

	a, err := e.adapterFactory(name, modelCfg)
	if err == nil {
		return a, name, modelCfg, warnings
	}

	// Primary adapter creation failed — walk the fallback chain.
	if len(modelCfg.Fallback) == 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%s %s: adapter creation failed: %v (no fallback configured)",
			role, name, err))
		return nil, name, config.ModelConfig{}, warnings
	}

	for _, fbName := range modelCfg.Fallback {
		fa, fResolved, fCfg, warnings := e.resolveChain(fbName, role, visited, warnings)
		if fa != nil {
			return fa, fResolved, fCfg, warnings
		}
	}

	warnings = append(warnings, fmt.Sprintf(
		"%s %s: adapter creation failed and all %d fallback(s) exhausted",
		role, name, len(modelCfg.Fallback)))
	return nil, name, config.ModelConfig{}, warnings
}
