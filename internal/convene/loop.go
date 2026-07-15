// loop.go — Agentic outer loop (P0).
//
// ConveneEngine.Run executes a single MoA pipeline pass. Before this file,
// the CLI ran one pass and stopped — the executor's own internal loop was a
// black box and OpenConvene had no way to re-dispatch when a task was only
// partially complete.
//
// ConveneLoop wraps Run in an outer loop that re-dispatches the task until
// completion. Completion is detected by two mechanisms (dual):
//
//  1. Explicit [[DONE]] marker — the executor emits a [[DONE]] block in its
//     output when it considers the task finished. The loop scans for this
//     marker and stops immediately.
//  2. Implicit judge — if no [[DONE]] marker is present, a judge model
//     (the synthesizer, or the executor if no synthesizer is configured) is
//     asked "is the task complete? if not, what is the next step?". If the
//     judge says complete, the loop stops. Otherwise the judge's "next step"
//     becomes the task for the next iteration.
//
// Stop conditions (any one):
//   - [[DONE]] marker found in execution output.
//   - judge declares the task complete.
//   - iteration count reaches MaxIterations (default 5).
//   - context canceled (Ctrl+C / timeout).
//
// Research mode does not loop (no execution phase, nothing to iterate on).
package convene

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// DefaultMaxIterations is the loop cap when MaxIterations is 0 or unset.
const DefaultMaxIterations = 5

// doneMarkerRe matches a [[DONE]] or [[DONE/]] block in executor output.
// The block may contain optional summary text between the markers.
var doneMarkerRe = regexp.MustCompile(`(?s)\[\[DONE\]\]\s*\n?(.*?)\n?\s*\[\[/DONE\]\]`)

// bareDoneRe matches a bare [[DONE]] without a closing tag (single-line).
var bareDoneRe = regexp.MustCompile(`(?m)^\[\[DONE\]\]\s*$`)

// judgePromptTemplate asks the judge model whether the task is complete.
const judgePromptTemplate = `You are a completion judge in a Mixture-of-Agents agentic loop. The user gave a task and the executor has run. Decide whether the task is fully complete.

INSTRUCTIONS:
- If the task is complete, reply with EXACTLY this line and nothing else:
  COMPLETE
- If the task is NOT complete, reply with the next concrete step the executor should take (a single actionable instruction, no preamble). This will be fed back as the next task.

=== ORIGINAL TASK ===
%s

=== EXECUTION OUTPUT (most recent iteration) ===
%s

=== YOUR JUDGMENT ===`

// LoopResult is the aggregate result of a full ConveneLoop run.
type LoopResult struct {
	// Iterations is the number of pipeline passes executed.
	Iterations int

	// FinalResult is the ConveneResult from the last iteration.
	FinalResult ConveneResult

	// AllResults is the result of every iteration (for auditability).
	AllResults []ConveneResult

	// StopReason is why the loop stopped:
	// "done-marker" | "judge-complete" | "max-iterations" | "canceled" | "error"
	StopReason string

	// TotalElapsed is the wall-clock time of the whole loop.
	TotalElapsed time.Duration
}

// HasDoneMarker reports whether text contains a [[DONE]] marker (block or bare).
func HasDoneMarker(text string) bool {
	return doneMarkerRe.MatchString(text) || bareDoneRe.MatchString(text)
}

// StripDoneMarker removes the [[DONE]] marker block from text, returning the
// cleaned text. If no marker is present, the original text is returned.
func StripDoneMarker(text string) string {
	cleaned := doneMarkerRe.ReplaceAllString(text, "")
	cleaned = bareDoneRe.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

// ConveneLoop runs the agentic outer loop around ConveneEngine.Run.
//
// It re-dispatches the task until completion (dual mechanism: [[DONE]] marker
// or judge verdict) or the iteration cap is reached. Research mode bypasses
// the loop (single pass, no execution to iterate on).
//
// Parameters:
//   - ctx: context for cancellation.
//   - task: the original task.
//   - mode: "research" | "code" | "agent".
//   - responders, executor, synthesizer: model selection (post-lane-routing).
//   - maxIterations: loop cap (0 = DefaultMaxIterations, 1 = single-shot).
func (e *ConveneEngine) ConveneLoop(
	ctx context.Context,
	task string,
	mode string,
	responders []string,
	executor string,
	synthesizer *string,
	maxIterations int,
) (LoopResult, error) {
	overallStart := time.Now()
	result := LoopResult{StopReason: "max-iterations"}

	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}

	// Research mode never loops — no execution phase to iterate on.
	// Single-shot (maxIterations == 1) also bypasses the judge/marker logic.
	singleShot := mode == "research" || maxIterations == 1

	currentTask := task

	for iter := 1; iter <= maxIterations; iter++ {
		// Check context before each expensive pass.
		if err := ctx.Err(); err != nil {
			result.StopReason = "canceled"
			result.Iterations = iter - 1
			result.TotalElapsed = time.Since(overallStart)
			return result, err
		}

		runResult, err := e.Run(ctx, currentTask, mode, responders, executor, synthesizer)
		if err != nil {
			// Record the partial result if Run returned one alongside the error.
			result.Iterations = iter
			result.FinalResult = runResult
			result.AllResults = append(result.AllResults, runResult)
			result.StopReason = "error"
			result.TotalElapsed = time.Since(overallStart)
			return result, err
		}

		result.AllResults = append(result.AllResults, runResult)
		result.FinalResult = runResult
		result.Iterations = iter

		if singleShot {
			result.StopReason = "single-shot"
			result.TotalElapsed = time.Since(overallStart)
			return result, nil
		}

		// --- Dual completion check ---

		// Mechanism 1: explicit [[DONE]] marker in execution output.
		execOutput := ""
		if runResult.Execution != nil {
			execOutput = *runResult.Execution
		}
		if HasDoneMarker(execOutput) {
			// Strip the marker from the stored execution output.
			cleaned := StripDoneMarker(execOutput)
			result.FinalResult.Execution = &cleaned
			result.StopReason = "done-marker"
			result.TotalElapsed = time.Since(overallStart)
			return result, nil
		}

		// Mechanism 2: implicit judge.
		// On the last allowed iteration, skip the judge (no point — we'd stop
		// anyway) and report max-iterations.
		if iter == maxIterations {
			result.StopReason = "max-iterations"
			result.TotalElapsed = time.Since(overallStart)
			return result, nil
		}

		judgment, err := e.judgeCompletion(ctx, task, execOutput, synthesizer, executor)
		if err != nil {
			// Judge failed — record and continue to next iteration with the
			// original task (re-try). This is fault-tolerant: a failed judge
			// does not abort the loop.
			if result.FinalResult.Metadata == nil {
				result.FinalResult.Metadata = make(map[string]interface{})
			}
			result.FinalResult.Metadata["loop_judge_error"] = err.Error()
			currentTask = task
			continue
		}

		judgment = strings.TrimSpace(judgment)
		if isJudgeComplete(judgment) {
			result.StopReason = "judge-complete"
			result.TotalElapsed = time.Since(overallStart)
			return result, nil
		}

		// Judge says incomplete — feed the next-step as the new task.
		if judgment != "" {
			currentTask = judgment
		} else {
			// Empty judgment — fall back to original task.
			currentTask = task
		}
	}

	result.TotalElapsed = time.Since(overallStart)
	return result, nil
}

// judgeCompletion asks the judge model whether the task is complete.
// Returns the judge's raw output (which may be "COMPLETE" or a next-step).
func (e *ConveneEngine) judgeCompletion(
	ctx context.Context,
	originalTask string,
	execOutput string,
	synthesizer *string,
	executor string,
) (string, error) {
	// The judge is the synthesizer if configured, else the executor.
	judgeName := ""
	if synthesizer != nil && *synthesizer != "" {
		judgeName = *synthesizer
	} else if executor != "" {
		judgeName = executor
	} else {
		return "", fmt.Errorf("no model available to judge completion")
	}

	judgeCfg, exists := e.Config.Models[judgeName]
	if !exists {
		dynCfg, _, dynOk := adapter.ResolveDynamicModel(judgeName)
		if !dynOk {
			return "", fmt.Errorf("judge %s not found", judgeName)
		}
		judgeCfg = dynCfg
	}

	a, err := e.adapterFactory(judgeName, judgeCfg)
	if err != nil {
		return "", fmt.Errorf("judge adapter creation failed: %w", err)
	}

	prompt := fmt.Sprintf(judgePromptTemplate, originalTask, execOutput)
	t := judgeCfg.Timeout
	if t <= 0 {
		t = e.Config.Defaults.Timeout
	}

	res, err := a.Respond(ctx, prompt, t)
	if err != nil || !res.Success {
		return "", fmt.Errorf("judge call failed: %v", err)
	}
	return res.Stdout, nil
}

// isJudgeComplete reports whether the judge's output declares the task
// complete. The judge is instructed to reply "COMPLETE" on its own line;
// we tolerate surrounding whitespace and a trailing period.
func isJudgeComplete(judgment string) bool {
	upper := strings.ToUpper(strings.TrimSpace(judgment))
	// Exact match or first-line match.
	firstLine := strings.SplitN(upper, "\n", 2)[0]
	firstLine = strings.TrimRight(firstLine, ".!? ")
	return firstLine == "COMPLETE" || firstLine == "DONE"
}

// ResolveMaxIterations picks the effective max-iterations value from the
// config default and an optional CLI override.
func ResolveMaxIterations(cfgDefault, cliOverride int) int {
	if cliOverride > 0 {
		return cliOverride
	}
	if cfgDefault > 0 {
		return cfgDefault
	}
	return DefaultMaxIterations
}

// LaneRoutingEnabled reports whether lane routing is on, given the config
// and an optional CLI disable flag.
func LaneRoutingEnabled(cfg *config.ConveneConfig, cliDisabled bool) bool {
	if cliDisabled {
		return false
	}
	if cfg.Defaults.LaneRouting == nil {
		return true // default ON
	}
	return *cfg.Defaults.LaneRouting
}
