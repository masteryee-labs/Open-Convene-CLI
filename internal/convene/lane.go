// lane.go — Task-classification lane routing (omnilane-inspired).
//
// Before running the MoA pipeline, the engine can classify the task into one
// of six lanes. Each lane maps to a preferred set of responders and an
// executor, so the right models are used for the right kind of work:
//
//	hardest-coding   — deep root-cause debug, correctness-critical edits
//	bulk-mechanical  — refactors, migrations, tests, review sweeps
//	triage           — high-volume scans, first-pass filtering
//	taste-final      — user-facing prose, prompt/doc polish
//	long-context     — large-context synthesis (read-only analysis)
//	live-search      — realtime web/social search and context
//
// Classification is done by a lightweight responder call that returns the
// lane name. The classifier uses the first responder in the configured list
// (or the executor if no responders are configured). If classification fails
// or lane routing is disabled, the engine falls back to the static
// responders/executor from config defaults.
//
// Lane routing is ON by default. Disable with --no-lane or
// defaults.lane_routing: false in models.yaml.
package convene

import (
	"context"
	"fmt"
	"strings"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// ---------------------------------------------------------------------------
// Lane definitions
// ---------------------------------------------------------------------------

// Lane is a task category used for model routing.
type Lane string

const (
	LaneHardestCoding  Lane = "hardest-coding"
	LaneBulkMechanical Lane = "bulk-mechanical"
	LaneTriage         Lane = "triage"
	LaneTasteFinal     Lane = "taste-final"
	LaneLongContext    Lane = "long-context"
	LaneLiveSearch     Lane = "live-search"
)

// AllLanes returns the full ordered list of built-in lanes.
func AllLanes() []Lane {
	return []Lane{
		LaneHardestCoding,
		LaneBulkMechanical,
		LaneTriage,
		LaneTasteFinal,
		LaneLongContext,
		LaneLiveSearch,
	}
}

// LaneDescription returns a human-readable description of a lane.
func LaneDescription(l Lane) string {
	switch l {
	case LaneHardestCoding:
		return "Hardest implementation, deep root-cause debug, correctness-critical edits"
	case LaneBulkMechanical:
		return "Refactors, migrations, tests, review sweeps — mechanical endurance"
	case LaneTriage:
		return "High-volume scans, first-pass filtering"
	case LaneTasteFinal:
		return "User-facing prose, prompt/doc polish, style arbitration"
	case LaneLongContext:
		return "Large-context synthesis — analysis only, never agentic loops"
	case LaneLiveSearch:
		return "Realtime web/social search and context"
	default:
		return "unknown lane"
	}
}

// classifyPromptTemplate instructs the classifier model to pick a lane.
const classifyPromptTemplate = `You are a task router in a Mixture-of-Agents pipeline. Classify the following task into exactly one of these lanes by replying with ONLY the lane name on a single line (no explanation, no punctuation):

- hardest-coding: deep root-cause debugging, correctness-critical edits, the hardest implementation work
- bulk-mechanical: refactors, migrations, writing tests, review sweeps, mechanical high-volume edits
- triage: high-volume scans, first-pass filtering, sorting/categorizing many items
- taste-final: user-facing prose, documentation polish, prompt engineering, style decisions
- long-context: synthesis over very large inputs (read-only analysis, no execution)
- live-search: tasks needing realtime web or social-media search

=== TASK ===
%s

=== LANE (reply with the lane name only) ===`

// ---------------------------------------------------------------------------
// Lane selection
// ---------------------------------------------------------------------------

// LaneSelection is the resolved model selection for a classified lane.
type LaneSelection struct {
	Lane         Lane
	Responders   []string
	Executor     string
	Synthesizer  *string
	ClassifiedBy string // "classifier" | "fallback" | "disabled"
}

// ResolveLane picks the responders/executor/synthesizer for a given lane,
// merging config.Lanes overrides with config.Defaults fallbacks.
//
// Resolution order for each role:
//  1. cfg.Lanes[lane].<role> (if non-empty)
//  2. cfg.Defaults.<role>
func ResolveLane(l Lane, cfg *config.ConveneConfig) LaneSelection {
	sel := LaneSelection{Lane: l, ClassifiedBy: "fallback"}

	// Start from defaults.
	sel.Responders = append([]string(nil), cfg.Defaults.Responders...)
	sel.Executor = cfg.Defaults.Executor
	sel.Synthesizer = cfg.Defaults.Synthesizer

	// Apply per-lane overrides if present.
	if laneCfg, ok := cfg.Lanes[string(l)]; ok {
		if len(laneCfg.Responders) > 0 {
			sel.Responders = append([]string(nil), laneCfg.Responders...)
		}
		if laneCfg.Executor != "" {
			sel.Executor = laneCfg.Executor
		}
		if laneCfg.Synthesizer != nil {
			sel.Synthesizer = laneCfg.Synthesizer
		}
	}

	return sel
}

// ClassifyLane asks a classifier model to categorize the task into a lane.
//
// The classifier is the first responder in the list (or the executor if no
// responders are configured). It receives a short prompt and must reply with
// only the lane name. If the call fails or the reply does not match a known
// lane, LaneHardestCoding is returned as a safe default (hardest-coding is
// the most capable superset).
//
// This function performs a single read-only Respond call. It does NOT execute.
func (e *ConveneEngine) ClassifyLane(ctx context.Context, task string, responders []string, executor string, timeout int) (Lane, error) {
	// Pick the classifier: first responder, else executor.
	classifierName := ""
	var classifierCfg config.ModelConfig
	if len(responders) > 0 {
		classifierName = responders[0]
	} else if executor != "" {
		classifierName = executor
	} else {
		return LaneHardestCoding, fmt.Errorf("no model available to classify lane")
	}

	// Resolve config (config map or dynamic).
	classifierCfg, exists := e.Config.Models[classifierName]
	if !exists {
		dynCfg, _, dynOk := adapter.ResolveDynamicModel(classifierName)
		if !dynOk {
			return LaneHardestCoding, fmt.Errorf("classifier %s not found", classifierName)
		}
		classifierCfg = dynCfg
	}

	a, err := e.adapterFactory(classifierName, classifierCfg)
	if err != nil {
		return LaneHardestCoding, fmt.Errorf("classifier adapter creation failed: %w", err)
	}

	prompt := fmt.Sprintf(classifyPromptTemplate, task)
	t := timeout
	if t <= 0 {
		t = classifierCfg.Timeout
	}
	if t <= 0 {
		t = e.Config.Defaults.Timeout
	}

	result, err := a.Respond(ctx, prompt, t)
	if err != nil || !result.Success {
		return LaneHardestCoding, fmt.Errorf("classifier call failed: %v", err)
	}

	// Parse the lane name from the response (tolerate surrounding whitespace
	// and a trailing explanation by taking the first token that matches).
	lane := parseLane(result.Stdout)
	if lane == "" {
		return LaneHardestCoding, fmt.Errorf("classifier returned unknown lane: %q", strings.TrimSpace(result.Stdout))
	}
	return lane, nil
}

// parseLane extracts a known lane name from the classifier output. It scans
// the text for any known lane identifier (case-insensitive) and returns the
// canonical Lane. Returns "" if none match.
func parseLane(text string) Lane {
	lower := strings.ToLower(text)
	for _, l := range AllLanes() {
		ls := string(l)
		if strings.Contains(lower, ls) {
			return l
		}
		// Also match the bare keyword without the category suffix.
		short := strings.Split(ls, "-")[0]
		if short != ls && strings.Contains(lower, short) {
			// Only accept short match if it's a distinct word to avoid
			// false positives (e.g. "live" inside "delivery").
			if hasWord(lower, short) {
				return l
			}
		}
	}
	return ""
}

// hasWord reports whether s contains the given word as a standalone token
// (bounded by non-letter characters).
func hasWord(s, word string) bool {
	idx := strings.Index(s, word)
	for idx >= 0 {
		before := idx == 0 || !isLetter(s[idx-1])
		after := idx+len(word) == len(s) || !isLetter(s[idx+len(word)])
		if before && after {
			return true
		}
		next := idx + 1
		if next >= len(s) {
			break
		}
		idx = strings.Index(s[next:], word)
		if idx < 0 {
			break
		}
		idx += next
	}
	return false
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
