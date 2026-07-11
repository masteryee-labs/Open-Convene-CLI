// Package convene implements the Mixture-of-Agents (MoA) collaboration engine
// for OpenConveneCLI.
//
// This file defines ConveneResult (the complete output of a Convene run) and
// ConveneError (a phase-tagged error type). The engine logic lives in
// engine.go; prompt templates live in prompts.go.
//
// MoA theory (arXiv:2406.04692):
//   - N responders run in parallel; total latency ≈ max(responder_i), not sum.
//   - The synthesizer performs reasoning-based integration, NOT majority voting.
//   - Stochasticity is a feature: parallel sampling from the response
//     distribution lets the synthesizer pick the optimal combination.
//   - Fault tolerance: a single responder failure does not abort the run;
//     MoA's value is "at least one model gets it right."
package convene

import "fmt"

// ConveneResult is the complete result of a Convene execution.
//
// Field semantics (Go pointer semantics for optional values):
//   - Responses: always non-nil (may be empty if all responders failed).
//   - Synthesis: *string — nil means no synthesizer was configured, the
//     synthesizer adapter could not be created, or the synthesizer call failed
//     (fallback to executor-as-synthesizer).
//   - Execution: *string — nil means research mode (no execution) or the
//     executor call failed.
//   - Metadata: always non-nil; records per-responder timing, success/failure,
//     warnings, and errors for auditability.
type ConveneResult struct {
	// Task is the original task description provided by the user.
	Task string

	// Mode is the execution mode: "research" | "code" | "agent".
	Mode string

	// Responses maps responder model names to their plain-text responses.
	// Key = model name (e.g. "agy"), value = AdapterResult.Stdout.
	// Only successful responses are included; failures are recorded in Metadata.
	Responses map[string]string

	// Synthesis is the synthesizer's integrated result.
	// nil = no synthesizer configured, adapter creation failed, or the call
	// failed. When nil, the executor (if any) reads the raw Responses directly
	// via BuildExecPrompt.
	Synthesis *string

	// Execution is the executor's execution result.
	// nil = research mode (no execution) or the executor call failed.
	Execution *string

	// Metadata records timing, errors, warnings, and per-responder audit info.
	// Keys include:
	//   "responder_warnings"   — []string of warnings
	//   "<model>_success"      — bool
	//   "<model>_elapsed"      — time.Duration
	//   "<model>_error"        — string (error from Respond)
	//   "<model>_failed"       — string (Stderr from a non-success result)
	//   "synthesizer_success"  — bool
	//   "synthesizer_error"    — string
	//   "executor_success"     — bool
	//   "executor_error"       — string
	//   "executor_failed"      — string
	//   "responder_count"      — int
	//   "success_count"        — int
	//   "total_elapsed"        — time.Duration
	Metadata map[string]interface{}
}

// ConveneError is a custom error type that tags errors with the phase in which
// they occurred.
//
// Phase values: "respond" | "synthesize" | "execute".
// Either Err or Message (or both) may be set. When Err is non-nil, Error()
// includes both the underlying error and the extra message.
type ConveneError struct {
	// Phase is the Convene pipeline phase where the error occurred.
	// "respond" | "synthesize" | "execute"
	Phase string

	// Err is the underlying error (may be nil, paired with Message).
	Err error

	// Message is an additional human-readable message (may be empty; Err is
	// the primary error when both are set).
	Message string
}

// Error implements the error interface.
//
// When Err is non-nil, the output includes both the underlying error and the
// extra message. When Err is nil, only the phase and message are included.
func (e *ConveneError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("convene error in %s: %v: %s", e.Phase, e.Err, e.Message)
	}
	return fmt.Sprintf("convene error in %s: %s", e.Phase, e.Message)
}
