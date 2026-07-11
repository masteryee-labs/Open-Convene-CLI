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
}

// ConveneConfig is the top-level configuration for OpenConveneCLI.
// It is parsed from models.yaml.
type ConveneConfig struct {
	// Models maps adapter names to their configurations.
	// The map key becomes ModelConfig.Name (populated by the factory).
	Models map[string]ModelConfig `yaml:"models"`

	// Defaults holds the default responder/executor/synthesizer/timeout values.
	Defaults DefaultsConfig `yaml:"defaults"`
}
