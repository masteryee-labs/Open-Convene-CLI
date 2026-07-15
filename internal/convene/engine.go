// engine.go — ConveneEngine: the Mixture-of-Agents (MoA) collaboration engine.
//
// ConveneEngine.Run executes a 3-phase pipeline:
//
//	Phase 1 — FAN-OUT (parallel responders):
//	  N responder models run concurrently via goroutines + errgroup.
//	  Total latency ≈ max(responder_i), NOT sum — this is the key MoA insight
//	  (arXiv:2406.04692). Each responder is read-only (Respond, not Execute):
//	  no tools, no file writes, no side-effects. This prevents N models from
//	  simultaneously modifying files during fan-out.
//	  Fault tolerance: a single responder failure does NOT abort the run.
//	  MoA's value is "at least one model gets it right" — losing one of N
//	  still leaves N-1 samples from the response distribution.
//
//	Phase 2 — SYNTHESIS (optional):
//	  If a synthesizer is configured (synthesizer != nil), it reads ALL
//	  responder responses and produces a reasoning-based integration — NOT
//	  majority voting, NOT averaging. The synthesizer identifies which model
//	  is correct on each sub-argument, flags hallucinations, and assembles a
//	  stronger answer. If the synthesizer fails, synthesis falls back to nil
//	  (executor reads raw responses instead).
//
//	Phase 3 — EXECUTION (mode-dependent):
//	  - research: skipped (output = synthesis or raw responses).
//	  - code/agent: the executor (the ONLY role with side-effects) runs AFTER
//	    synthesis, as a single execution unit. It uses the synthesis (or raw
//	    responses if no synthesis) as guidance and may write files / run
//	    commands.
//
// Dependency injection: AdapterFactory is a function type defaulting to
// adapter.GetAdapter. S6 tests can inject a mock factory via
// SetAdapterFactory() to test engine logic without real CLIs.
package convene

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// AdapterFactory is the function type for creating adapter instances.
//
// Default = adapter.GetAdapter. S6 tests inject a mock factory via
// SetAdapterFactory() to test engine logic without real CLI processes.
type AdapterFactory func(name string, cfg config.ModelConfig) (adapter.Adapter, error)

// ConveneEngine is the multi-model collaboration engine.
//
// It orchestrates the 3-phase MoA pipeline: parallel responders → synthesis →
// execution. The engine is stateless across runs (all state is in ConveneResult).
type ConveneEngine struct {
	// Config is the parsed ConveneConfig (models + defaults).
	Config *config.ConveneConfig

	// Language is the preferred output language for model responses.
	// Empty = no preference. When set, a language directive is prepended
	// to the task so all models (responders, synthesizer, executor) respond
	// in the specified language. CLI UI is NOT affected.
	Language string

	// adapterFactory creates Adapter instances. Defaults to adapter.GetAdapter.
	// S6 tests replace this with a mock factory via SetAdapterFactory().
	adapterFactory AdapterFactory
}

// ---------------------------------------------------------------------------
// Constructor + dependency injection
// ---------------------------------------------------------------------------

// NewConveneEngine creates a ConveneEngine with the given config and the
// default adapter factory (adapter.GetAdapter). It reads the Language setting
// from cfg.Defaults.Language.
func NewConveneEngine(cfg *config.ConveneConfig) *ConveneEngine {
	return &ConveneEngine{
		Config:         cfg,
		Language:       cfg.Defaults.Language,
		adapterFactory: adapter.GetAdapter,
	}
}

// SetAdapterFactory replaces the adapter factory. This is intended for S6
// testing: inject a mock factory that returns fake Adapters, allowing engine
// logic to be unit-tested without real CLI processes.
//
// Usage in tests:
//
//	engine := convene.NewConveneEngine(cfg)
//	engine.SetAdapterFactory(mockFactory)  // returns fake adapters
//	result, err := engine.Run(ctx, task, mode, responders, executor, synthesizer)
func (e *ConveneEngine) SetAdapterFactory(f AdapterFactory) {
	e.adapterFactory = f
}

// ---------------------------------------------------------------------------
// Internal result types
// ---------------------------------------------------------------------------

// responderResult holds the outcome of a single responder goroutine.
type responderResult struct {
	name    string
	result  adapter.AdapterResult
	err     error
	elapsed time.Duration
}

// preparedResponder holds a responder whose adapter has been successfully
// created and validated, ready for parallel Respond calls.
type preparedResponder struct {
	name    string
	adapter adapter.Adapter
	timeout int
}

// ---------------------------------------------------------------------------
// Run — the main pipeline
// ---------------------------------------------------------------------------

// Run executes the Convene pipeline for the given task.
//
// Parameters:
//   - ctx: context for cancellation (Ctrl+C) and deadline propagation.
//   - task: the original task description.
//   - mode: "research" | "code" | "agent".
//   - responders: model name list (e.g. ["agy", "grok"]).
//   - executor: model name for execution (ignored in research mode).
//   - synthesizer: model name pointer; nil = executor doubles as synthesizer.
//
// Returns:
//   - ConveneResult with all phase outputs and audit metadata.
//   - error only for fatal conditions (all responders failed, unknown mode).
//     Individual responder/synthesizer/executor failures are recorded in
//     ConveneResult.Metadata, NOT returned as errors (fault tolerance).
func (e *ConveneEngine) Run(
	ctx context.Context,
	task string,
	mode string,
	responders []string,
	executor string,
	synthesizer *string,
) (ConveneResult, error) {

	overallStart := time.Now()

	// If a language is set, prepend a language directive to the task so all
	// models (responders, synthesizer, executor) respond in that language.
	// This only affects model output — CLI UI stays in English.
	if e.Language != "" {
		task = fmt.Sprintf("[Please respond in %s.]\n\n%s", e.Language, task)
	}

	// Step 0: Initialize metadata map + nil synthesis/execution.
	metadata := make(map[string]interface{})

	// ─────────────────────────────────────────────────────────────────────
	// Phase 1: PARALLEL RESPONDERS (fan-out)
	// ─────────────────────────────────────────────────────────────────────
	//
	// MoA principle: N responders run concurrently. Total latency ≈ max(),
	// not sum(). Each responder is read-only (Respond, not Execute) — no
	// side-effects during fan-out. Fault tolerance: single failure ≠ abort.

	var warnings []string

	// Build + validate all responder adapters before fan-out.
	// This allows SupportsReadOnly() checks and per-model timeout resolution
	// to happen sequentially (cheap), while the expensive Respond() calls
	// run in parallel goroutines.
	prepared := make([]preparedResponder, 0, len(responders))
	for _, name := range responders {
		// Self-execute dedup (P2): if this responder is the same as the
		// executor and we're in code/agent mode, skip the redundant Respond
		// call — the executor will produce its own output in Phase 3, so
		// calling it twice wastes a quota call and biases the synthesis
		// toward the executor's own response.
		if (mode == "code" || mode == "agent") && name == executor {
			warnings = append(warnings, fmt.Sprintf(
				"self-execute: skipped redundant responder call for %s "+
					"(it is also the executor)", name))
			continue
		}

		// Resolve the model, walking the fallback chain if the primary is
		// unavailable (P1). resolveWithFallback returns the first model in
		// the chain that resolves to a usable adapter.
		a, resolvedName, modelCfg, fbNote := e.resolveWithFallback(name, "responder", warnings)
		if a == nil {
			warnings = fbNote
			continue // Fault tolerance: skip this responder, don't abort.
		}
		if resolvedName != name {
			warnings = append(warnings, fmt.Sprintf(
				"responder %s: fell back to %s", name, resolvedName))
		}

		// Validate read-only support. Responders MUST be read-only to prevent
		// N models from simultaneously modifying files during fan-out.
		if !a.SupportsReadOnly() {
			warnings = append(warnings, fmt.Sprintf(
				"responder %s is not truly read-only (read_only=%q), "+
					"may cause side-effects during fan-out",
				resolvedName, modelCfg.ReadOnly))
		}

		// Per-model timeout, fallback to defaults.
		t := modelCfg.Timeout
		if t <= 0 {
			t = e.Config.Defaults.Timeout
		}

		prepared = append(prepared, preparedResponder{
			name:    resolvedName,
			adapter: a,
			timeout: t,
		})
	}

	// Store warnings in metadata for auditability.
	if len(warnings) > 0 {
		metadata["responder_warnings"] = warnings
	}

	// Fan-out: each prepared responder gets its own goroutine via errgroup.
	//
	// We use errgroup for synchronization (g.Wait() blocks until all goroutines
	// complete). Each goroutine returns nil — errors are captured in the results
	// slice, NOT returned to errgroup. This preserves MoA fault tolerance:
	// a single responder failure does NOT cancel the context or abort other
	// goroutines. If we returned errors to errgroup, the first failure would
	// cancel the context and kill remaining responders — the opposite of what
	// MoA requires.
	//
	// Results are written to pre-allocated slice positions (no mutex needed —
	// each goroutine writes to a distinct index).
	results := make([]responderResult, len(prepared))
	g, gctx := errgroup.WithContext(ctx)
	for i, pr := range prepared {
		i, pr := i, pr // capture loop variables for goroutine closure
		g.Go(func() error {
			start := time.Now()
			r, err := pr.adapter.Respond(gctx, task, pr.timeout)
			results[i] = responderResult{
				name:    pr.name,
				result:  r,
				err:     err,
				elapsed: time.Since(start),
			}
			// Return nil always — fault tolerance: don't cancel other goroutines.
			return nil
		})
	}
	_ = g.Wait() // err is always nil (all goroutines return nil)

	// Collect results (fault tolerance: single failure ≠ abort).
	responses := make(map[string]string)
	successCount := 0
	for _, r := range results {
		// Record timing for every responder (audit trail).
		metadata[fmt.Sprintf("%s_elapsed", r.name)] = r.elapsed

		if r.err != nil {
			// Respond returned a Go error (timeout, start failure, etc.)
			metadata[fmt.Sprintf("%s_error", r.name)] = r.err.Error()
			metadata[fmt.Sprintf("%s_success", r.name)] = false
			continue
		}

		if r.result.Success {
			// Success: extract Stdout (plain-text response).
			responses[r.name] = r.result.Stdout
			metadata[fmt.Sprintf("%s_success", r.name)] = true
			successCount++
		} else {
			// Non-success: CLI ran but returned non-zero or empty stdout.
			metadata[fmt.Sprintf("%s_failed", r.name)] = r.result.Stderr
			metadata[fmt.Sprintf("%s_success", r.name)] = false
		}
	}

	metadata["responder_count"] = len(prepared)
	metadata["success_count"] = successCount

	// At least 1 responder must succeed. If all fail, MoA has no samples to
	// synthesize from — this is a fatal error.
	if successCount == 0 {
		metadata["total_elapsed"] = time.Since(overallStart)
		return ConveneResult{
			Task:      task,
			Mode:      mode,
			Responses: responses,
			Synthesis: nil,
			Execution: nil,
			Metadata:  metadata,
		}, &ConveneError{
			Phase:   "respond",
			Message: "all responders failed — no responses to synthesize",
		}
	}

	// ─────────────────────────────────────────────────────────────────────
	// Phase 2: SYNTHESIS (optional)
	// ─────────────────────────────────────────────────────────────────────
	//
	// MoA principle: the synthesizer performs reasoning-based integration,
	// NOT majority voting. It reads all N responses, identifies which model
	// is correct on each sub-argument, flags hallucinations, and assembles
	// a stronger answer. If synthesis fails, fall back to nil (executor
	// reads raw responses) — do NOT abort.

	var synthesis *string
	if synthesizer != nil {
		synthName := *synthesizer

		// P3: arbitrate panel (vote mode) — alternative to single-synthesizer
		// reasoning. When synthesis_mode == "vote", the question goes to a
		// panel of voters and a chair assembles the verdict.
		if e.Config.Defaults.SynthesisMode == "vote" {
			voters := e.Config.Defaults.VoteVoters
			rounds := e.Config.Defaults.VoteRounds
			if rounds < 1 {
				rounds = 1
			}
			verdict, arbMeta, arbErr := e.ArbitratePanel(ctx, task, responses,
				voters, synthName, rounds, e.Config.Defaults.Timeout)
			if arbErr != nil {
				metadata["synthesizer_error"] = fmt.Sprintf(
					"arbitrate panel failed: %v", arbErr)
				synthesis = nil // Fallback: no synthesis (don't abort).
			} else {
				synthesis = &verdict
				metadata["synthesizer_success"] = true
				metadata["synthesis_mode"] = "vote"
				// Merge arbitrate metadata.
				for k, v := range arbMeta {
					metadata[k] = v
				}
			}
		} else {
			// Default: single-synthesizer reasoning-based integration.
			synthAdapter, resolvedSynth, synthCfg, fbNote := e.resolveWithFallback(
				synthName, "synthesizer", warnings)
			warnings = fbNote
			if synthAdapter == nil {
				metadata["synthesizer_error"] = fmt.Sprintf(
					"synthesizer %s: adapter creation failed (no fallback)", synthName)
				synthesis = nil
			} else {
				if resolvedSynth != synthName {
					warnings = append(warnings, fmt.Sprintf(
						"synthesizer %s: fell back to %s", synthName, resolvedSynth))
					metadata["synthesizer_fallback"] = resolvedSynth
				}
				synthesisPrompt := BuildSynthesisPrompt(task, responses)

				// Per-model timeout, fallback to defaults.
				t := synthCfg.Timeout
				if t <= 0 {
					t = e.Config.Defaults.Timeout
				}

				synthStart := time.Now()
				synthResult, err := synthAdapter.Respond(ctx, synthesisPrompt, t)
				metadata["synthesizer_elapsed"] = time.Since(synthStart)

				if err == nil && synthResult.Success {
					s := synthResult.Stdout
					synthesis = &s
					metadata["synthesizer_success"] = true
				} else {
					// Synthesizer failed — fallback to nil (no synthesis).
					// Do NOT abort; executor will read raw responses.
					synthesis = nil
					metadata["synthesizer_success"] = false
					if err != nil {
						metadata["synthesizer_error"] = err.Error()
					} else {
						metadata["synthesizer_error"] = synthResult.Stderr
					}
					// Also record stderr for debugging (even when err != nil).
					if synthResult.Stderr != "" {
						metadata["synthesizer_stderr"] = synthResult.Stderr
					}
				}
			}
		}
	} else {
		// No synthesizer configured — executor doubles as synthesizer.
		synthesis = nil
	}

	// ─────────────────────────────────────────────────────────────────────
	// Phase 3: EXECUTION (mode-dependent)
	// ─────────────────────────────────────────────────────────────────────
	//
	// The executor is the ONLY role with side-effects (file writes, commands).
	// It runs AFTER synthesis, as a single execution unit, to prevent N models
	// from simultaneously modifying files.
	//   - research: skip (output = synthesis or raw responses).
	//   - code/agent: build executor adapter, call Execute with synthesis context.

	var execution *string
	switch mode {
	case "research":
		// Research mode does not execute — output is synthesis or responses.
		execution = nil

	case "code", "agent":
		// Resolve the executor, walking the fallback chain if the primary
		// is unavailable (P1).
		execAdapter, resolvedExec, execCfg, fbNote := e.resolveWithFallback(
			executor, "executor", warnings)
		warnings = fbNote
		if execAdapter == nil {
			// Use the last fallback warning as the error detail (it explains
			// why resolution failed: "not found", "adapter creation failed", etc.).
			detail := "adapter creation failed (no fallback)"
			if len(warnings) > 0 {
				detail = warnings[len(warnings)-1]
			}
			metadata["executor_error"] = fmt.Sprintf(
				"executor %s: %s", executor, detail)
			// execution stays nil.
		} else {
			if resolvedExec != executor {
				warnings = append(warnings, fmt.Sprintf(
					"executor %s: fell back to %s", executor, resolvedExec))
				metadata["executor_fallback"] = resolvedExec
			}
			// Build the executor prompt from task + synthesis (or raw responses).
			execPrompt := BuildExecPrompt(task, synthesis, responses, mode)

			// Per-model timeout, fallback to defaults.
			t := execCfg.Timeout
			if t <= 0 {
				t = e.Config.Defaults.Timeout
			}

			// synthesisContext: the synthesis string value (empty if nil).
			// The adapter's Execute receives this but the prompt already
			// contains the synthesis/responses — synthesisContext is
			// supplementary context passed through to the CLI.
			var synthCtx string
			if synthesis != nil {
				synthCtx = *synthesis
			}

			execStart := time.Now()
			execResult, err := execAdapter.Execute(ctx, execPrompt, t, synthCtx)
			metadata["executor_elapsed"] = time.Since(execStart)

			if err == nil && execResult.Success {
				// Use execOut (not e) to avoid shadowing receiver e *ConveneEngine.
				execOut := execResult.Stdout
				execution = &execOut
				metadata["executor_success"] = true
			} else {
				// Executor failed — record in metadata, execution stays nil.
				metadata["executor_success"] = false
				metadata["executor_failed"] = fmt.Sprintf(
					"err=%v stderr=%s", err, execResult.Stderr)
			}
		}

	default:
		metadata["total_elapsed"] = time.Since(overallStart)
		return ConveneResult{
			Task:      task,
			Mode:      mode,
			Responses: responses,
			Synthesis: synthesis,
			Execution: nil,
			Metadata:  metadata,
		}, &ConveneError{
			Phase:   "execute",
			Err:     fmt.Errorf("unknown mode: %s", mode),
			Message: "mode must be one of: research, code, agent",
		}
	}

	// ─────────────────────────────────────────────────────────────────────
	// Final: assemble ConveneResult
	// ─────────────────────────────────────────────────────────────────────
	metadata["total_elapsed"] = time.Since(overallStart)

	return ConveneResult{
		Task:      task,
		Mode:      mode,
		Responses: responses,
		Synthesis: synthesis,
		Execution: execution,
		Metadata:  metadata,
	}, nil
}
