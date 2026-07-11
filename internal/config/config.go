// Package config implements the configuration loading and validation logic for
// OpenConveneCLI.
//
// This file (produced by S5) contains:
//   - LoadConfig:           resolve + parse models.yaml into ConveneConfig
//   - ValidateConfig:       structural + referential integrity checks
//   - GenerateExampleConfig: produce a ready-to-use models.yaml string
//   - InitConfig:            write an example config to disk
//   - ModelConfig.IsReadOnly / IsMaybeReadOnly: read_only semantics helpers
//
// The struct definitions (ModelConfig / DefaultsConfig / ConveneConfig) live in
// models.go (produced by S1). Go allows methods on a type to be defined in a
// different file of the same package, so the helpers below are added here
// without touching models.go.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// ModelConfig helper methods (cross-file definitions for the S1 struct)
// ---------------------------------------------------------------------------

// IsReadOnly reports whether the model TRULY supports read-only mode.
//
// Only read_only == "true" counts. The "maybe" value describes a non-interactive
// mode that is still inherently agentic, so it is treated as NOT read-only
// (unreliable) and returns false.
func (m *ModelConfig) IsReadOnly() bool {
	return m.ReadOnly == "true"
}

// IsMaybeReadOnly reports whether the model has a soft, unreliable read-only
// mode (read_only == "maybe").
//
// "maybe" is a prompt-level soft constraint: the CLI can run non-interactively
// but nothing enforces that it will not modify files. Callers should NOT rely on
// this for safety guarantees.
func (m *ModelConfig) IsMaybeReadOnly() bool {
	return m.ReadOnly == "maybe"
}

// ---------------------------------------------------------------------------
// Config path resolution
// ---------------------------------------------------------------------------

// DefaultConfigPaths returns the default search locations for models.yaml, in
// priority order (the first existing file wins).
//
// Resolution order used by LoadConfig:
//  1. explicit path (--config flag)
//  2. OPENCONVENE_CLI_CONFIG environment variable
//  3. the paths returned here (first existing match)
func DefaultConfigPaths() []string {
	home, _ := os.UserHomeDir()
	return []string{
		filepath.Join(home, ".config", "openconvene", "models.yaml"),
		filepath.Join("config", "models.yaml"),
	}
}

// resolveConfigPath determines which config file to load.
//
// Priority: explicit path > OPENCONVENE_CLI_CONFIG env > default paths.
// When no explicit path or env var is given, the first existing default path
// wins. If none exist, an error is returned with an init hint.
func resolveConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if envPath := os.Getenv("OPENCONVENE_CLI_CONFIG"); envPath != "" {
		return envPath, nil
	}
	for _, p := range DefaultConfigPaths() {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	paths := DefaultConfigPaths()
	hint := "config/models.yaml"
	if len(paths) > 0 {
		hint = paths[0]
	}
	return "", fmt.Errorf("no models.yaml config found in default search paths. "+
		"To generate one, run: openconvene init (writes to %s)", hint)
}

// ---------------------------------------------------------------------------
// LoadConfig
// ---------------------------------------------------------------------------

// LoadConfig loads and validates models.yaml.
//
// path resolution (when path == ""):
//  1. OPENCONVENE_CLI_CONFIG environment variable
//  2. default search paths (see DefaultConfigPaths)
//
// After parsing, the YAML map key is injected into ModelConfig.Name (the field
// is tagged yaml:"-" so the parser skips it). Validation then runs via
// ValidateConfig; any ERROR-severity issue causes LoadConfig to return an error.
//
// A missing config file yields an error with an init hint.
func LoadConfig(path string) (*ConveneConfig, error) {
	resolved, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %q. "+
				"Run `openconvene init` to generate one", resolved)
		}
		return nil, fmt.Errorf("failed to read config %q: %w", resolved, err)
	}

	var cfg ConveneConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %q: %w", resolved, err)
	}

	// Inject the YAML map key into ModelConfig.Name (yaml:"-" is not auto-parsed).
	// Copy out, mutate, write back: map values are not addressable directly.
	for name, modelCfg := range cfg.Models {
		modelCfg.Name = name
		cfg.Models[name] = modelCfg
	}

	// Validate. ERROR-severity issues block loading; WARNING issues are advisory.
	issues := ValidateConfig(&cfg)
	var errs []string
	for _, issue := range issues {
		if strings.HasPrefix(issue, "ERROR:") {
			errs = append(errs, issue)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed for %q:\n  %s",
			resolved, strings.Join(errs, "\n  "))
	}

	return &cfg, nil
}

// ---------------------------------------------------------------------------
// ValidateConfig
// ---------------------------------------------------------------------------

// ValidateConfig checks the config for structural and referential integrity and
// returns a slice of human-readable issue strings.
//
// Each issue is prefixed with either:
//   - "ERROR:"   — blocks loading (LoadConfig returns an error)
//   - "WARNING:" — advisory only (does not block)
//
// An empty slice means the config is valid.
//
// Checks performed:
//   - at least one executor_capable=true model exists
//   - every model: read_only is one of "", "true", "false", "maybe"
//   - every model: timeout >= 0 (0 = inherit defaults; negative is invalid)
//   - command/execute_command {prompt} placeholder rules:
//     * execute_command empty  => command must be non-empty and contain {prompt}
//     * execute_command non-empty => execute_command must contain {prompt};
//       command (if non-empty) must contain {prompt}
//   - defaults.executor exists in models and is executor_capable=true
//   - defaults.responders is non-empty; each name exists in models and has a
//     non-empty command
//   - responder with read_only=false => WARNING (may modify files)
//   - defaults.synthesizer (if non-nil / non-empty) exists in models
func ValidateConfig(cfg *ConveneConfig) []string {
	if cfg == nil {
		return []string{"ERROR: config is nil"}
	}
	var issues []string

	// --- at least one executor_capable model (or dynamic model names in defaults) ---
	hasExecutor := false
	for _, m := range cfg.Models {
		if m.ExecutorCapable {
			hasExecutor = true
			break
		}
	}
	// If no executor_capable model in models section, check if defaults.executor
	// or defaults.responders reference dynamic model names (which are resolved
	// at runtime and are executor_capable by default).
	if !hasExecutor {
		if cfg.Defaults.Executor != "" {
			if _, ok := cfg.Models[cfg.Defaults.Executor]; !ok {
				// defaults.executor is a dynamic model name — all dynamic models
				// are executor_capable by default.
				hasExecutor = true
			}
		}
	}
	if !hasExecutor {
		issues = append(issues, "ERROR: no executor_capable model found; "+
			"at least one model must have executor_capable: true, "+
			"or defaults.executor must be a dynamic model name (CLI:模型名)")
	}

	// --- per-model checks ---
	for name, m := range cfg.Models {
		// read_only value
		switch m.ReadOnly {
		case "", "true", "false", "maybe":
			// valid
		default:
			issues = append(issues, fmt.Sprintf(
				"ERROR: model '%s': read_only must be 'true', 'false', or 'maybe', got '%s'",
				name, m.ReadOnly))
		}

		// timeout
		if m.Timeout < 0 {
			issues = append(issues, fmt.Sprintf(
				"ERROR: model '%s': timeout must be >= 0, got %d", name, m.Timeout))
		} else if m.Timeout == 0 {
			issues = append(issues, fmt.Sprintf(
				"WARNING: model '%s': timeout is 0; will inherit defaults.timeout", name))
		}

		// {prompt} or {prompt_file} placeholder rules
		// A command template must contain either {prompt} (inline, shell-escaped)
		// or {prompt_file} (written to a temp file, path substituted). The latter
		// is useful for CLIs that support --prompt-file (e.g. devin) and for
		// prompts with newlines or special characters that break shell quoting.
		hasPromptPlaceholder := func(s string) bool {
			return strings.Contains(s, "{prompt}") || strings.Contains(s, "{prompt_file}")
		}
		if m.ExecuteCommand == "" {
			// command is the fallback for both respond and execute modes
			if m.Command == "" {
				issues = append(issues, fmt.Sprintf(
					"ERROR: model '%s': command and execute_command are both empty; "+
						"at least one must contain the {prompt} or {prompt_file} placeholder", name))
			} else if !hasPromptPlaceholder(m.Command) {
				issues = append(issues, fmt.Sprintf(
					"ERROR: model '%s': command missing {prompt} or {prompt_file} placeholder", name))
			}
		} else {
			if !hasPromptPlaceholder(m.ExecuteCommand) {
				issues = append(issues, fmt.Sprintf(
					"ERROR: model '%s': execute_command missing {prompt} or {prompt_file} placeholder", name))
			}
			if m.Command != "" && !hasPromptPlaceholder(m.Command) {
				issues = append(issues, fmt.Sprintf(
					"ERROR: model '%s': command missing {prompt} or {prompt_file} placeholder", name))
			}
		}
	}

	// --- defaults.executor ---
	if cfg.Defaults.Executor == "" {
		issues = append(issues, "ERROR: defaults.executor is not set")
	} else {
		execModel, ok := cfg.Models[cfg.Defaults.Executor]
		if !ok {
			// May be a dynamic model name (CLI:模型名 format, e.g. "devin:glm-5.2").
			// Dynamic names are resolved at runtime by the engine; here we only
			// warn that it's not in the models section.
			issues = append(issues, fmt.Sprintf(
				"WARNING: defaults.executor '%s' is not in models section; "+
					"if it's a dynamic model name (CLI:模型名), it will be resolved at runtime",
				cfg.Defaults.Executor))
		} else if !execModel.ExecutorCapable {
			issues = append(issues, fmt.Sprintf(
				"ERROR: defaults.executor '%s' is not executor_capable "+
					"(executor_capable must be true)", cfg.Defaults.Executor))
		}
	}

	// --- defaults.responders ---
	if len(cfg.Defaults.Responders) == 0 {
		issues = append(issues, "ERROR: defaults.responders is empty; "+
			"at least one responder is required")
	} else {
		for _, r := range cfg.Defaults.Responders {
			rModel, ok := cfg.Models[r]
			if !ok {
				// May be a dynamic model name — warn but don't error.
				issues = append(issues, fmt.Sprintf(
					"WARNING: defaults.responders '%s' is not in models section; "+
						"if it's a dynamic model name (CLI:模型名), it will be resolved at runtime",
					r))
				continue
			}
			// a responder needs a usable respond command
			if rModel.Command == "" {
				issues = append(issues, fmt.Sprintf(
					"ERROR: responder '%s' has an empty command; cannot be used as a responder", r))
			}
			// read_only=false responder is risky (advisory)
			if rModel.ReadOnly == "false" {
				issues = append(issues, fmt.Sprintf(
					"WARNING: responder '%s' has read_only=false; "+
						"it may modify files in respond mode", r))
			}
		}
	}

	// --- defaults.synthesizer (nil / empty = executor doubles as synthesizer) ---
	if cfg.Defaults.Synthesizer != nil {
		synthName := *cfg.Defaults.Synthesizer
		if synthName != "" {
			if _, ok := cfg.Models[synthName]; !ok {
				// May be a dynamic model name — warn but don't error.
				issues = append(issues, fmt.Sprintf(
					"WARNING: defaults.synthesizer '%s' is not in models section; "+
						"if it's a dynamic model name (CLI:模型名), it will be resolved at runtime",
					synthName))
			}
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// GenerateExampleConfig + InitConfig
// ---------------------------------------------------------------------------

// GenerateExampleConfig returns a ready-to-use models.yaml example as a string.
//
// The command templates are derived from official CLI documentation research.
// read_only values marked "maybe" await empirical verification by S2.
// NOTE: read_only values are quoted so YAML parses them as strings (not bools)
// and they unmarshal cleanly into ModelConfig.ReadOnly (string).
func GenerateExampleConfig() string {
	return `# ============================================================
# OpenConveneCLI — Model Configuration
# Path: ~/.config/openconvene/models.yaml  (or ./config/models.yaml)
# Generate with: openconvene init
#
# Command templates are based on official CLI documentation research.
# read_only values marked "maybe" await empirical verification (S2).
# ============================================================

defaults:
  timeout: 120
  responders:
    - agy
    - grok
  executor: codex
  synthesizer: null              # null = executor doubles as synthesizer

models:
  # --- Antigravity (AGY) ---
  agy:
    command: 'agy -p "{prompt}"'
    execute_command: 'agy -p "{prompt}"'
    read_only: "maybe"           # Antigravity CLI; awaiting S2 verification
    timeout: 120
    executor_capable: true
    extra_args: []

  # --- Codex ---
  # respond vs execute use different sandbox flags
  codex:
    command: 'codex exec --sandbox read-only "{prompt}"'
    execute_command: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: "true"            # --sandbox read-only enforces read-only
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Devin ---
  devin:
    command: 'devin -p "{prompt}"'
    execute_command: 'devin --permission-mode bypass "{prompt}"'
    read_only: "maybe"           # -p is non-interactive but inherently agentic
    timeout: 300
    executor_capable: true
    extra_args: []

  # --- Grok ---
  grok:
    command: 'grok -p "{prompt}"'
    execute_command: 'grok -p "{prompt}"'
    read_only: "maybe"           # awaiting S2 verification
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Cursor ---
  cursor:
    command: 'cursor agent -p "{prompt}"'
    execute_command: 'cursor agent -p --force "{prompt}"'
    read_only: "true"            # read-only without --force
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Kimi Code ---
  kimi:
    command: 'kimi -p "{prompt}"'
    execute_command: 'kimi -p "{prompt}"'
    read_only: "true"            # read-only ops auto-approved
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Hermes ---
  hermes:
    command: 'hermes chat -q "{prompt}"'
    execute_command: 'hermes chat -q "{prompt}"'
    read_only: "maybe"           # awaiting S2 verification
    timeout: 180
    executor_capable: true
    extra_args: []

  # --- Aider ---
  # Not suitable as a responder (empty command); read_only=false by default.
  # User must set an API key env var (e.g. ANTHROPIC_API_KEY) and may adjust
  # the --model value (e.g. sonnet) to taste.
  aider:
    command: ''                  # aider is a code editor, not a responder
    execute_command: 'aider --yes --model sonnet "{prompt}"'
    read_only: "false"           # modifies files by default
    timeout: 300
    executor_capable: true
    extra_args: []

  # --- OpenCode ---
  opencode:
    command: 'opencode run "{prompt}"'
    execute_command: 'opencode run "{prompt}"'
    read_only: "maybe"           # awaiting S2 verification
    timeout: 180
    executor_capable: true
    extra_args: []
`
}

// InitConfig writes an example models.yaml to the given path.
//
// If path is empty it defaults to "config/models.yaml". The parent directory is
// created if needed. An existing file is NOT overwritten (the call fails with a
// clear message) to prevent clobbering user edits.
func InitConfig(path string) error {
	if path == "" {
		path = filepath.Join("config", "models.yaml")
	}

	// Refuse to overwrite an existing config.
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists at %q; "+
			"remove it first if you want to regenerate", path)
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create config directory %q: %w", dir, err)
		}
	}

	content := GenerateExampleConfig()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write config to %q: %w", path, err)
	}
	return nil
}
