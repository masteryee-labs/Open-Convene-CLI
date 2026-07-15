// Package config defines the configuration data structures for OpenConveneCLI.
//
// This file contains ONLY the struct definitions with yaml tags.
// Methods (IsReadOnly, Validate, etc.) are implemented by S5 in config.go.
//
// S1 produces this file as the project skeleton so that subsequent sessions
// (S2, S3, S5) can import these types directly.
package config

// ModelConfig is the configuration for a single model CLI adapter.
//
// The Name field is NOT parsed from YAML; it is populated by the factory
// using the map key from ConveneConfig.Models (e.g. "agy", "codex").
type ModelConfig struct {
	// Name is the adapter identifier, populated from the YAML map key.
	// yaml:"-" ensures the parser skips this field; the factory fills it in.
	Name string `yaml:"-"`

	// Command is the respond (read-only) mode command template.
	// It MUST contain the {prompt} placeholder, e.g.:
	//   "agy -p \"{prompt}\""
	// The factory replaces {prompt} with the actual prompt at call time.
	Command string `yaml:"command"`

	// ExecuteCommand is the execute (agentic) mode command template.
	// Optional: if empty, the adapter falls back to Command.
	// Some CLIs need a different invocation for execution, e.g.:
	//   "codex exec --sandbox workspace-write \"{prompt}\""
	ExecuteCommand string `yaml:"execute_command"`

	// ReadOnly indicates whether the CLI truly supports read-only mode.
	// Values: "true" | "false" | "maybe"
	//   true  — the CLI enforces read-only (e.g. codex --sandbox read-only)
	//   false — the CLI modifies files by default (e.g. aider)
	//   maybe — the CLI has a non-interactive mode but is inherently agentic
	ReadOnly string `yaml:"read_only"`

	// Timeout is the default per-call timeout in seconds.
	// Overridable by the CLI --timeout flag or Defaults.Timeout.
	Timeout int `yaml:"timeout"`

	// ExecutorCapable indicates whether this model can serve as an executor
	// (i.e. run in agentic mode to write code / modify files).
	// Models with read_only="false" are typically executor-capable (e.g. aider).
	ExecutorCapable bool `yaml:"executor_capable"`

	// ExtraArgs holds additional CLI arguments appended to the command
	// (e.g. ["--model", "gpt-4o"] for aider).
	ExtraArgs []string `yaml:"extra_args"`

	// Fallback is the ordered list of model names to try when this model's
	// CLI is not installed or its adapter cannot be created. The engine walks
	// the chain and uses the first candidate that resolves successfully.
	// Each entry may itself have its own Fallback chain (recursion is bounded
	// by a visited-set to prevent cycles). Empty = no fallback (current behavior).
	// Example: fallback: [codex, devin:glm-5.2]
	Fallback []string `yaml:"fallback"`
}

// DefaultsConfig holds the default values applied when the CLI does not
// explicitly specify responders, executor, or synthesizer.
//
// NOTE: This is a strongly-typed struct, NOT map[string]interface{}.
// Access fields directly: cfg.Defaults.Responders, cfg.Defaults.Executor, etc.
type DefaultsConfig struct {
	// Timeout is the default per-call timeout in seconds.
	Timeout int `yaml:"timeout"`

	// Responders is the default list of responder model names
	// (e.g. ["agy", "grok"]).
	Responders []string `yaml:"responders"`

	// Executor is the default executor model name (e.g. "codex").
	Executor string `yaml:"executor"`

	// Synthesizer is the default synthesizer model name.
	// nil (null in YAML) means the executor doubles as synthesizer.
	Synthesizer *string `yaml:"synthesizer"`

	// Language is the preferred output language for model responses.
	// Empty = no preference (model defaults to English or follows the task language).
	// Examples: "zh-TW", "繁體中文", "English", "日本語".
	// This only affects model output language — CLI UI (slash commands, help
	// text, error messages) remains in English.
	Language string `yaml:"language"`

	// LaneRouting enables task-classification lane routing (omnilane-inspired).
	// When true (default), the engine classifies the task into a lane
	// (hardest-coding, bulk-mechanical, triage, taste-final, long-context,
	// live-search) and selects responders/executor accordingly.
	// Set to false to use the static responders/executor lists above.
	LaneRouting *bool `yaml:"lane_routing"`

	// SynthesisMode selects the synthesis strategy.
	// "reasoning" (default) = single synthesizer performs reasoning-based
	//   integration (current MoA behavior).
	// "vote" = arbitrate panel: the question goes to 1-4 voter models,
	//   opinions come back side by side, and a chair model assembles the
	//   verdict (omnilane-inspired arbitrate lane).
	SynthesisMode string `yaml:"synthesis_mode"`

	// VoteVoters is the list of voter model names used when SynthesisMode ==
	// "vote". Each voter answers the task independently. If empty, the
	// responders list is reused as voters.
	VoteVoters []string `yaml:"vote_voters"`

	// VoteRounds is the number of debate rounds for the arbitrate panel.
	// 1 (default) = single round, each voter answers independently.
	// 2 = voters see all round-1 opinions and rebut only disagreements.
	VoteRounds int `yaml:"vote_rounds"`

	// MaxIterations is the upper bound for the agentic outer loop
	// (P0). The loop re-dispatches the task until the executor emits a
	// [[DONE]] marker, a judge model declares the task complete, or this
	// limit is reached. 0 = use the built-in default (5). 1 = single-shot
	// (disables the loop, preserving the pre-loop behavior for research mode
	// or explicit single-run use).
	MaxIterations int `yaml:"max_iterations"`
}

// ConveneConfig is the top-level configuration for OpenConveneCLI.
// It is parsed from models.yaml.
type ConveneConfig struct {
	// Models maps adapter names to their configurations.
	// The map key becomes ModelConfig.Name (populated by the factory).
	Models map[string]ModelConfig `yaml:"models"`

	// Defaults holds the default responder/executor/synthesizer/timeout values.
	Defaults DefaultsConfig `yaml:"defaults"`

	// Lanes holds optional per-lane model overrides. When LaneRouting is
	// enabled and a lane key is present here, its responders/executor take
	// priority over Defaults. Keys are lane names (hardest-coding,
	// bulk-mechanical, triage, taste-final, long-context, live-search).
	// Empty = use built-in default lane assignments.
	Lanes map[string]LaneConfig `yaml:"lanes"`
}

// LaneConfig holds the model selection for a single routing lane.
// A lane is a task category (omnilane-inspired) that maps to a preferred
// set of responders and an executor.
type LaneConfig struct {
	// Responders is the list of responder model names for this lane.
	// If empty, the engine falls back to Defaults.Responders.
	Responders []string `yaml:"responders"`

	// Executor is the executor model name for this lane.
	// If empty, the engine falls back to Defaults.Executor.
	Executor string `yaml:"executor"`

	// Synthesizer is the synthesizer model name for this lane (optional).
	// nil = use Defaults.Synthesizer.
	Synthesizer *string `yaml:"synthesizer"`

	// Description is a human-readable hint shown in /lane output.
	Description string `yaml:"description"`
}
