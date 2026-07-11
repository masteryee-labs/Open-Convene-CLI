// Package mode defines the three execution modes for OpenConveneCLI and
// provides output formatting + configuration validation.
//
// Modes:
//   - research: Fan-out responders → synthesis (optional) → NO execution.
//     Output = synthesis or raw responses. Use for information gathering,
//     analysis, and Q&A where no file modifications are needed.
//   - code:     Fan-out responders → synthesis (optional) → executor.Execute.
//     The executor implements code based on the integrated guidance.
//     Use for coding tasks where file writes are expected.
//   - agent:    Fan-out responders → synthesis (optional) → executor.Execute.
//     The executor performs broader agentic actions (research, file ops,
//     multi-step workflows). Use for complex multi-step tasks.
//
// Mode behavior differences:
//
//	┌──────────┬───────────┬───────────┬──────────────────┐
//	│ Phase    │ research  │ code      │ agent            │
//	├──────────┼───────────┼───────────┼──────────────────┤
//	│ Fan-out  │ ✓         │ ✓         │ ✓                │
//	│ Synthesis│ ✓ (opt)   │ ✓ (opt)   │ ✓ (opt)          │
//	│ Execution│ ✗ (skip)  │ ✓ Execute │ ✓ Execute (broad)│
//	│ Output   │ synthesis │ execution │ execution        │
//	└──────────┴───────────┴───────────┴──────────────────┘
//
// MoA tradeoffs (warned by ValidateModeConfig when N ≥ 2):
//   - Latency ↑: fan-out + synthesis adds 5-15s vs single-model.
//   - Cost ↑: N responders + 1 synthesizer = N+1 API calls.
//   - Predictability ↓: synthesis output is harder to predict than a
//     single deterministic model.
//   - Best for: complex reasoning, long-form generation, multi-step logic,
//     reducing single-model blind spots.
//   - Not for: low-latency chat, simple classification, strict JSON schema.
package mode

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/masteryee-labs/open-convene-cli/internal/convene"
)

// ---------------------------------------------------------------------------
// Mode type
// ---------------------------------------------------------------------------

// Mode is the execution mode for a Convene run.
type Mode string

const (
	// ModeResearch: responders + optional synthesis, NO execution.
	// Output = synthesis (or raw responses if no synthesizer).
	ModeResearch Mode = "research"

	// ModeCode: responders + optional synthesis + executor (code implementation).
	// Output = execution result.
	ModeCode Mode = "code"

	// ModeAgent: responders + optional synthesis + executor (broad agentic actions).
	// Output = execution result.
	ModeAgent Mode = "agent"
)

// ---------------------------------------------------------------------------
// FormatOutput
// ---------------------------------------------------------------------------

// FormatOutput formats a ConveneResult for display to the user.
//
// The format differs by mode:
//   - research: shows synthesis (or raw responses) + responder details + metadata.
//   - code/agent: shows execution result + synthesis (if available) + metadata.
//
// Metadata (timing, success/failure counts, warnings, errors) is always included
// at the end for auditability — this is a core MoA requirement: users must be
// able to audit which model contributed what.
func FormatOutput(result convene.ConveneResult, mode Mode) string {
	var b strings.Builder

	// Header
	b.WriteString("=== Convene Result ===\n")
	fmt.Fprintf(&b, "Task: %s\n", truncate(result.Task, 200))
	fmt.Fprintf(&b, "Mode: %s\n\n", result.Mode)

	switch mode {
	case ModeResearch:
		formatResearchOutput(&b, result)
	case ModeCode, ModeAgent:
		formatExecutionOutput(&b, result)
	default:
		fmt.Fprintf(&b, "--- Unknown mode: %s ---\n", result.Mode)
	}

	// Metadata section (always included for auditability).
	formatMetadata(&b, result)

	return b.String()
}

// formatResearchOutput writes the research-mode output: synthesis or raw
// responses, followed by individual responder responses.
func formatResearchOutput(b *strings.Builder, result convene.ConveneResult) {
	if result.Synthesis != nil {
		b.WriteString("--- Synthesis ---\n")
		b.WriteString(*result.Synthesis)
		b.WriteString("\n\n")
	} else {
		b.WriteString("--- Synthesis ---\n")
		b.WriteString("(No synthesis available — showing raw responder responses)\n\n")
	}

	// Individual responder responses (for auditability).
	if len(result.Responses) > 0 {
		b.WriteString("--- Responder Responses ---\n")
		names := sortedKeys(result.Responses)
		for _, name := range names {
			elapsed := getDuration(result.Metadata, name+"_elapsed")
			fmt.Fprintf(b, "\n[%s] (%s):\n%s\n", name, elapsed, result.Responses[name])
		}
		b.WriteString("\n")
	}
}

// formatExecutionOutput writes the code/agent-mode output: execution result
// + synthesis (if available).
func formatExecutionOutput(b *strings.Builder, result convene.ConveneResult) {
	// Execution result (primary output for code/agent mode).
	b.WriteString("--- Execution ---\n")
	if result.Execution != nil {
		b.WriteString(*result.Execution)
	} else {
		b.WriteString("(Execution failed or not run — check metadata for details)")
	}
	b.WriteString("\n\n")

	// Synthesis (if available, shown as context).
	if result.Synthesis != nil {
		b.WriteString("--- Synthesis (integrated guidance) ---\n")
		b.WriteString(*result.Synthesis)
		b.WriteString("\n\n")
	}

	// Responder responses (abbreviated, for auditability).
	if len(result.Responses) > 0 {
		b.WriteString("--- Responder Responses ---\n")
		names := sortedKeys(result.Responses)
		for _, name := range names {
			elapsed := getDuration(result.Metadata, name+"_elapsed")
			fmt.Fprintf(b, "\n[%s] (%s):\n%s\n", name, elapsed,
				truncate(result.Responses[name], 500))
		}
		b.WriteString("\n")
	}
}

// formatMetadata writes the metadata section: success counts, timing, warnings.
func formatMetadata(b *strings.Builder, result convene.ConveneResult) {
	b.WriteString("--- Metadata ---\n")

	// Responder success/failure counts.
	responderCount := getInt(result.Metadata, "responder_count")
	successCount := getInt(result.Metadata, "success_count")
	fmt.Fprintf(b, "Responders: %d/%d succeeded\n", successCount, responderCount)

	// Per-responder status.
	if responderCount > 0 {
		names := sortedKeys(result.Responses)
		for _, name := range names {
			elapsed := getDuration(result.Metadata, name+"_elapsed")
			fmt.Fprintf(b, "  %s: succeeded (%s)\n", name, elapsed)
		}
		// Show failed responders from metadata.
		for k, v := range result.Metadata {
			if strings.HasSuffix(k, "_error") || strings.HasSuffix(k, "_failed") {
				name := strings.TrimSuffix(strings.TrimSuffix(k, "_error"), "_failed")
				fmt.Fprintf(b, "  %s: failed — %v\n", name, v)
			}
		}
	}

	// Synthesizer status.
	if result.Synthesis != nil {
		elapsed := getDuration(result.Metadata, "synthesizer_elapsed")
		fmt.Fprintf(b, "Synthesizer: succeeded (%s)\n", elapsed)
	} else if synthErr, ok := result.Metadata["synthesizer_error"]; ok {
		fmt.Fprintf(b, "Synthesizer: failed — %v\n", synthErr)
	} else {
		b.WriteString("Synthesizer: not configured (executor doubles as synthesizer)\n")
	}

	// Executor status (code/agent mode).
	if result.Mode != "research" {
		if result.Execution != nil {
			elapsed := getDuration(result.Metadata, "executor_elapsed")
			fmt.Fprintf(b, "Executor: succeeded (%s)\n", elapsed)
		} else if execErr, ok := result.Metadata["executor_error"]; ok {
			fmt.Fprintf(b, "Executor: failed — %v\n", execErr)
		} else if execFailed, ok := result.Metadata["executor_failed"]; ok {
			fmt.Fprintf(b, "Executor: failed — %v\n", execFailed)
		}
	}

	// Total elapsed.
	totalElapsed := getDuration(result.Metadata, "total_elapsed")
	fmt.Fprintf(b, "Total elapsed: %s\n", totalElapsed)

	// Warnings.
	if warnings, ok := result.Metadata["responder_warnings"]; ok {
		if w, ok := warnings.([]string); ok && len(w) > 0 {
			b.WriteString("Warnings:\n")
			for _, warning := range w {
				fmt.Fprintf(b, "  - %s\n", warning)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// ValidateModeConfig
// ---------------------------------------------------------------------------

// ValidateModeConfig validates the mode + model combination for a Convene run.
//
// Returns:
//   - errors: hard errors that should abort the run (S4 prints these and
//     returns an error).
//   - warnings: soft warnings that should be displayed but do not abort
//     (S4 prints these to stderr and continues).
//
// Checks performed:
//
//  1. research mode + executor specified → warning "executor will be ignored"
//  2. code/agent mode + empty executor → ERROR "mode requires executor"
//  3. code/agent mode + responder with read_only=false → warning about
//     side-effects during fan-out
//  4. N=1 responders → warning "MoA benefit lost" (only 1 sample from the
//     response distribution)
//  5. synthesizer is also a responder → warning about synthesis bias
//  6. executor is also a responder → warning about execution bias
//  7. synthesizer not truly read-only → warning about tool execution during
//     synthesis
//  8. N ≥ 2 → MoA tradeoff warnings (latency, cost, predictability)
//  9. referenced models not in config → ERROR
func ValidateModeConfig(
	mode Mode,
	responders []string,
	executor string,
	synthesizer *string,
	models map[string]config.ModelConfig,
) (errors []string, warnings []string) {

	// --- Check 1: research mode + executor specified → warning ---
	if mode == ModeResearch && executor != "" {
		warnings = append(warnings,
			"research mode does not execute; executor will be ignored")
	}

	// --- Check 2: code/agent mode + empty executor → ERROR ---
	if (mode == ModeCode || mode == ModeAgent) && executor == "" {
		errors = append(errors,
			fmt.Sprintf("mode %q requires an executor model", mode))
	}

	// --- Check 9 (early): referenced models must exist in config or be dynamic ---
	// Dynamic model names (CLI:模型名 format, e.g. "devin:glm-5.2") are resolved
	// at runtime by the engine using built-in CLI templates. Here we only check
	// that the name is either in config models or looks like a dynamic name.
	for _, r := range responders {
		if _, ok := models[r]; !ok {
			if !looksLikeDynamicModel(r) {
				errors = append(errors,
					fmt.Sprintf("responder %q is not defined in config models and is not a dynamic model name (CLI:模型名)", r))
			}
		}
	}
	if executor != "" {
		if _, ok := models[executor]; !ok {
			if !looksLikeDynamicModel(executor) {
				errors = append(errors,
					fmt.Sprintf("executor %q is not defined in config models and is not a dynamic model name (CLI:模型名)", executor))
			}
		}
	}
	if synthesizer != nil && *synthesizer != "" {
		if _, ok := models[*synthesizer]; !ok {
			if !looksLikeDynamicModel(*synthesizer) {
				errors = append(errors,
					fmt.Sprintf("synthesizer %q is not defined in config models and is not a dynamic model name (CLI:模型名)", *synthesizer))
			}
		}
	}

	// --- Check 3: code/agent + responder with read_only=false → warning ---
	if mode == ModeCode || mode == ModeAgent {
		for _, r := range responders {
			if modelCfg, ok := models[r]; ok {
				if modelCfg.ReadOnly == "false" {
					warnings = append(warnings, fmt.Sprintf(
						"responder %q has read_only=false; it may modify files "+
							"during fan-out (side-effect risk in %s mode)", r, mode))
				}
			}
		}
	}

	// --- Check 4: N=1 → MoA benefit lost ---
	if len(responders) == 1 {
		warnings = append(warnings,
			"only 1 responder configured; MoA benefit is lost — "+
				"parallel multi-model sampling needs N≥2 to draw multiple samples "+
				"from the response distribution")
	} else if len(responders) == 0 {
		errors = append(errors,
			"at least one responder is required")
	}

	// --- Check 5: synthesizer is also a responder → bias warning ---
	if synthesizer != nil && *synthesizer != "" {
		for _, r := range responders {
			if r == *synthesizer {
				warnings = append(warnings, fmt.Sprintf(
					"synthesizer %q is also a responder; this may bias synthesis "+
						"toward its own response", *synthesizer))
				break
			}
		}
	}

	// --- Check 6: executor is also a responder → bias warning ---
	if executor != "" {
		for _, r := range responders {
			if r == executor {
				warnings = append(warnings, fmt.Sprintf(
					"executor %q is also a responder; this may bias execution "+
						"toward its own response", executor))
				break
			}
		}
	}

	// --- Check 7: synthesizer not truly read-only → warning ---
	if synthesizer != nil && *synthesizer != "" {
		if modelCfg, ok := models[*synthesizer]; ok {
			if modelCfg.ReadOnly != "true" {
				warnings = append(warnings, fmt.Sprintf(
					"synthesizer %q is not truly read-only (read_only=%q); "+
						"it may execute tools during synthesis, causing side-effects "+
						"before the executor runs",
					*synthesizer, modelCfg.ReadOnly))
			}
		}
	}

	// --- Check 8: N ≥ 2 → MoA tradeoff warnings ---
	if len(responders) >= 2 {
		warnings = append(warnings,
			fmt.Sprintf("MoA active with %d responders — tradeoffs: "+
				"latency +5-15s vs single-model, cost = %d API calls (N responders + "+
				"1 synthesizer + 1 executor), output predictability decreases. "+
				"Best for: complex reasoning, long-form generation, multi-step logic. "+
				"Not for: low-latency chat, simple classification, strict JSON schema.",
				len(responders), len(responders)+2))
	}

	return errors, warnings
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// sortedKeys returns the keys of a map sorted alphabetically (for deterministic
// output ordering).
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// getInt safely extracts an int from a map[string]interface{}.
func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		}
	}
	return 0
}

// getDuration safely extracts a time.Duration from a map[string]interface{}
// and formats it as a string (e.g. "1.23s").
func getDuration(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case time.Duration:
			return val.String()
		case string:
			return val
		}
	}
	return "unknown"
}

// knownCLIPrefixes is the list of 9 supported CLI names used for dynamic model
// name detection. This mirrors adapter.knownCLINames but is duplicated here to
// avoid an import cycle (mode → adapter → config → mode).
//
// We use ":" as the separator (not "-") because model names frequently contain
// hyphens (e.g. "glm-5.2"), making "-" ambiguous. ":" never appears in CLI or
// model names.
var knownCLIPrefixes = []string{
	"devin:", "grok:", "codex:", "agy:", "cursor:",
	"kimi:", "hermes:", "aider:", "opencode:",
}

// looksLikeDynamicModel returns true if the name starts with a known CLI name
// followed by ":" (e.g. "devin:glm-5.2", "agy:Gemini 3.5 Flash (High)").
//
// This is a lightweight check used by mode validation to accept dynamic model
// names without importing the adapter package (which would cause a cycle).
// The full resolution happens at runtime in the engine via adapter.ResolveDynamicModel.
func looksLikeDynamicModel(name string) bool {
	for _, prefix := range knownCLIPrefixes {
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
