// config_test.go — tests for the config package (S6).
//
// Covers LoadConfig (valid/missing/invalid), ValidateConfig (missing executor,
// bad read_only, missing {prompt}), GenerateExampleConfig, and InitConfig.
//
// All tests use temp directories + temp YAML files; no real config files are
// touched. The testify assertion library is used for readability.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeTempConfig writes the given YAML content to a temp file and returns its
// path. The file is cleaned up via t.Cleanup.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "models.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// validConfigYAML is a minimal but fully-valid config used as a baseline.
const validConfigYAML = `defaults:
  timeout: 120
  responders:
    - agy
    - grok
  executor: codex
  synthesizer: null

models:
  agy:
    command: 'agy -p "{prompt}"'
    execute_command: 'agy -p "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []

  codex:
    command: 'codex exec --sandbox read-only "{prompt}"'
    execute_command: 'codex exec --sandbox workspace-write "{prompt}"'
    read_only: "true"
    timeout: 180
    executor_capable: true
    extra_args: []

  grok:
    command: 'grok -p "{prompt}"'
    execute_command: 'grok -p "{prompt}"'
    read_only: "maybe"
    timeout: 180
    executor_capable: true
    extra_args: []
`

// ---------------------------------------------------------------------------
// LoadConfig tests
// ---------------------------------------------------------------------------

func TestLoadConfigValid(t *testing.T) {
	path := writeTempConfig(t, validConfigYAML)

	cfg, err := LoadConfig(path)
	require.NoError(t, err, "valid config should load without error")
	require.NotNil(t, cfg)

	// Defaults should be parsed correctly.
	assert.Equal(t, 120, cfg.Defaults.Timeout)
	assert.Equal(t, "codex", cfg.Defaults.Executor)
	assert.Equal(t, []string{"agy", "grok"}, cfg.Defaults.Responders)
	assert.Nil(t, cfg.Defaults.Synthesizer, "synthesizer null → nil pointer")

	// Models map should have 3 entries.
	assert.Len(t, cfg.Models, 3)

	// Name field (yaml:"-") should be injected from the map key.
	assert.Equal(t, "agy", cfg.Models["agy"].Name)
	assert.Equal(t, "codex", cfg.Models["codex"].Name)
	assert.Equal(t, "grok", cfg.Models["grok"].Name)

	// Spot-check a model's fields.
	codex := cfg.Models["codex"]
	assert.Equal(t, "true", codex.ReadOnly)
	assert.True(t, codex.ExecutorCapable)
	assert.Equal(t, 180, codex.Timeout)
	assert.Contains(t, codex.Command, "{prompt}")
}

func TestLoadConfigMissingFile(t *testing.T) {
	// A path that definitely does not exist.
	missingPath := filepath.Join(t.TempDir(), "nonexistent.yaml")

	cfg, err := LoadConfig(missingPath)
	require.Error(t, err, "missing file should return an error")
	assert.Nil(t, cfg)

	// Error message should hint at running config init.
	assert.Contains(t, err.Error(), "not found")
}

func TestLoadConfigMissingPromptPlaceholder(t *testing.T) {
	// A config where a model's command lacks {prompt}.
	badYAML := `defaults:
  timeout: 120
  responders:
    - agy
  executor: agy
  synthesizer: null

models:
  agy:
    command: 'agy -p no-placeholder-here'
    execute_command: 'agy -p no-placeholder-here'
    read_only: "maybe"
    timeout: 120
    executor_capable: true
    extra_args: []
`
	path := writeTempConfig(t, badYAML)

	cfg, err := LoadConfig(path)
	require.Error(t, err, "command without {prompt} should cause validation error")
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "{prompt}")
}

func TestLoadConfigMalformedYAML(t *testing.T) {
	badYAML := `defaults: [invalid yaml
  this is not valid: : :
`
	path := writeTempConfig(t, badYAML)

	cfg, err := LoadConfig(path)
	require.Error(t, err, "malformed YAML should return a parse error")
	assert.Nil(t, cfg)
}

// ---------------------------------------------------------------------------
// ValidateConfig tests
// ---------------------------------------------------------------------------

func TestValidateConfigValid(t *testing.T) {
	path := writeTempConfig(t, validConfigYAML)
	cfg, err := LoadConfig(path)
	require.NoError(t, err)

	issues := ValidateConfig(cfg)
	// The valid config has no ERROR-level issues (LoadConfig would have
	// failed otherwise). There may be WARNING-level issues (e.g. timeout=0
	// inheritance), but the example config sets timeouts explicitly.
	for _, issue := range issues {
		assert.False(t, strings.HasPrefix(issue, "ERROR:"),
			"valid config should have no ERROR issues, got: %s", issue)
	}
}

func TestValidateConfigNoExecutor(t *testing.T) {
	// A config where no model has executor_capable=true.
	noExecutorYAML := `defaults:
  timeout: 120
  responders:
    - agy
  executor: agy
  synthesizer: null

models:
  agy:
    command: 'agy -p "{prompt}"'
    execute_command: 'agy -p "{prompt}"'
    read_only: "maybe"
    timeout: 120
    executor_capable: false
    extra_args: []
`
	path := writeTempConfig(t, noExecutorYAML)
	cfg, err := LoadConfig(path)
	// LoadConfig runs ValidateConfig internally; no executor_capable → ERROR.
	require.Error(t, err, "config with no executor_capable model should fail loading")
	assert.Nil(t, cfg)

	// Also test via ValidateConfig directly on a hand-built config.
	cfg2 := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:    120,
			Responders: []string{"agy"},
			Executor:   "agy",
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "maybe",
				Timeout:         120,
				ExecutorCapable: false,
			},
		},
	}
	issues := ValidateConfig(cfg2)
	foundExecutorError := false
	for _, issue := range issues {
		if strings.HasPrefix(issue, "ERROR:") && strings.Contains(issue, "executor_capable") {
			foundExecutorError = true
			break
		}
	}
	assert.True(t, foundExecutorError, "should report ERROR about no executor_capable model")
}

func TestValidateConfigBadReadOnly(t *testing.T) {
	cfg := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:    120,
			Responders: []string{"agy"},
			Executor:   "agy",
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "bogus", // invalid value
				Timeout:         120,
				ExecutorCapable: true,
			},
		},
	}
	issues := ValidateConfig(cfg)
	foundReadOnlyError := false
	for _, issue := range issues {
		if strings.HasPrefix(issue, "ERROR:") && strings.Contains(issue, "read_only") {
			foundReadOnlyError = true
			break
		}
	}
	assert.True(t, foundReadOnlyError, "should report ERROR for invalid read_only value")
}

func TestValidateConfigUnknownExecutor(t *testing.T) {
	cfg := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:    120,
			Responders: []string{"agy"},
			Executor:   "nonexistent",
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "maybe",
				Timeout:         120,
				ExecutorCapable: true,
			},
		},
	}
	issues := ValidateConfig(cfg)
	foundUnknownExecutor := false
	for _, issue := range issues {
		// Unknown executor is now a WARNING (may be a dynamic model name like "devin-glm-5.2")
		if strings.HasPrefix(issue, "WARNING:") && strings.Contains(issue, "not in models section") {
			foundUnknownExecutor = true
			break
		}
	}
	assert.True(t, foundUnknownExecutor, "should report WARNING for unknown executor reference (may be dynamic)")
}

func TestValidateConfigUnknownResponder(t *testing.T) {
	cfg := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:    120,
			Responders: []string{"ghost"}, // not in models
			Executor:   "agy",
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "maybe",
				Timeout:         120,
				ExecutorCapable: true,
			},
		},
	}
	issues := ValidateConfig(cfg)
	foundUnknownResponder := false
	for _, issue := range issues {
		// Unknown responder is now a WARNING (may be a dynamic model name)
		if strings.HasPrefix(issue, "WARNING:") && strings.Contains(issue, "not in models section") && strings.Contains(issue, "ghost") {
			foundUnknownResponder = true
			break
		}
	}
	assert.True(t, foundUnknownResponder, "should report WARNING for unknown responder reference (may be dynamic)")
}

func TestValidateConfigUnknownSynthesizer(t *testing.T) {
	synthName := "phantom"
	cfg := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:     120,
			Responders:  []string{"agy"},
			Executor:    "agy",
			Synthesizer: &synthName,
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "maybe",
				Timeout:         120,
				ExecutorCapable: true,
			},
		},
	}
	issues := ValidateConfig(cfg)
	foundUnknownSynth := false
	for _, issue := range issues {
		// Unknown synthesizer is now a WARNING (may be a dynamic model name)
		if strings.HasPrefix(issue, "WARNING:") && strings.Contains(issue, "not in models section") && strings.Contains(issue, "phantom") {
			foundUnknownSynth = true
			break
		}
	}
	assert.True(t, foundUnknownSynth, "should report WARNING for unknown synthesizer reference (may be dynamic)")
}

func TestValidateConfigResponderReadOnlyFalseWarning(t *testing.T) {
	cfg := &ConveneConfig{
		Defaults: DefaultsConfig{
			Timeout:    120,
			Responders: []string{"aider"},
			Executor:   "agy",
		},
		Models: map[string]ModelConfig{
			"agy": {
				Name:            "agy",
				Command:         `agy -p "{prompt}"`,
				ExecuteCommand:  `agy -p "{prompt}"`,
				ReadOnly:        "maybe",
				Timeout:         120,
				ExecutorCapable: true,
			},
			"aider": {
				Name:            "aider",
				Command:         `aider "{prompt}"`,
				ExecuteCommand:  `aider --yes --model sonnet "{prompt}"`,
				ReadOnly:        "false",
				Timeout:         120,
				ExecutorCapable: true,
			},
		},
	}
	issues := ValidateConfig(cfg)
	foundWarning := false
	for _, issue := range issues {
		if strings.HasPrefix(issue, "WARNING:") && strings.Contains(issue, "read_only=false") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "should WARN about responder with read_only=false")
}

func TestValidateConfigNil(t *testing.T) {
	issues := ValidateConfig(nil)
	assert.NotEmpty(t, issues, "nil config should produce at least one issue")
	assert.Contains(t, issues[0], "nil")
}

// ---------------------------------------------------------------------------
// ModelConfig helper methods
// ---------------------------------------------------------------------------

func TestModelConfigIsReadOnly(t *testing.T) {
	tests := []struct {
		readOnly string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"maybe", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.readOnly, func(t *testing.T) {
			m := &ModelConfig{ReadOnly: tt.readOnly}
			assert.Equal(t, tt.expected, m.IsReadOnly())
		})
	}
}

func TestModelConfigIsMaybeReadOnly(t *testing.T) {
	tests := []struct {
		readOnly string
		expected bool
	}{
		{"maybe", true},
		{"true", false},
		{"false", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.readOnly, func(t *testing.T) {
			m := &ModelConfig{ReadOnly: tt.readOnly}
			assert.Equal(t, tt.expected, m.IsMaybeReadOnly())
		})
	}
}

// ---------------------------------------------------------------------------
// GenerateExampleConfig + InitConfig
// ---------------------------------------------------------------------------

func TestGenerateExampleConfig(t *testing.T) {
	content := GenerateExampleConfig()
	assert.NotEmpty(t, content)
	// Should contain all 9 adapter names.
	for _, name := range []string{"agy", "codex", "devin", "grok", "cursor", "kimi", "hermes", "aider", "opencode"} {
		assert.Contains(t, content, name, "example config should include %s", name)
	}
	// Should contain {prompt} placeholders.
	assert.Contains(t, content, "{prompt}")

	// The generated example should be loadable by LoadConfig (no ERROR issues).
	path := writeTempConfig(t, content)
	cfg, err := LoadConfig(path)
	require.NoError(t, err, "generated example config should load successfully")
	assert.NotNil(t, cfg)
	assert.GreaterOrEqual(t, len(cfg.Models), 9, "example should configure all 9 models")
}

func TestInitConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "models.yaml")

	// InitConfig should create parent directories and write the file.
	err := InitConfig(path)
	require.NoError(t, err, "InitConfig should succeed for a new path")

	// File should exist and be non-empty.
	info, err := os.Stat(path)
	require.NoError(t, err, "config file should exist after InitConfig")
	assert.Greater(t, info.Size(), int64(0), "config file should not be empty")

	// Second call should refuse to overwrite.
	err = InitConfig(path)
	assert.Error(t, err, "InitConfig should refuse to overwrite an existing file")
}

func TestInitConfigDefaultPath(t *testing.T) {
	// InitConfig("") defaults to "config/models.yaml" relative to CWD.
	// Use a manual temp dir (not t.TempDir) to avoid Windows cleanup races
	// when the CWD was inside the temp dir.
	origDir, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, err := os.MkdirTemp("", "occli-test-init-*")
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(origDir)
		_ = os.RemoveAll(tmpDir)
	}()

	require.NoError(t, os.Chdir(tmpDir))

	err = InitConfig("")
	require.NoError(t, err)

	// The default path is config/models.yaml.
	defaultPath := filepath.Join("config", "models.yaml")
	_, err = os.Stat(defaultPath)
	assert.NoError(t, err, "default config path should exist after InitConfig('')")
}
