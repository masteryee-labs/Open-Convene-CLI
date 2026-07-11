// mode_test.go — tests for the mode package (S6).
//
// Covers ValidateModeConfig (errors + warnings for each check) and
// FormatOutput (research / code / agent formatting).

package mode

import (
	"strings"
	"testing"

	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/masteryee-labs/open-convene-cli/internal/convene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildModels creates a config.Models map with the given model names.
// All models are executor_capable with read_only="true" unless overridden.
func buildModels(overrides map[string]config.ModelConfig, names ...string) map[string]config.ModelConfig {
	models := make(map[string]config.ModelConfig, len(names))
	for _, name := range names {
		m := config.ModelConfig{
			Name:            name,
			Command:         `mock "{prompt}"`,
			ExecuteCommand:  `mock exec "{prompt}"`,
			ReadOnly:        "true",
			Timeout:         120,
			ExecutorCapable: true,
		}
		if override, ok := overrides[name]; ok {
			m = override
		}
		models[name] = m
	}
	return models
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// ValidateModeConfig tests
// ---------------------------------------------------------------------------

func TestValidateModeResearchNoExecutor(t *testing.T) {
	// research mode does not need an executor → no error for empty executor.
	models := buildModels(nil, "agy", "grok")
	errors, warnings := ValidateModeConfig(ModeResearch, []string{"agy", "grok"}, "", nil, models)

	assert.Empty(t, errors, "research mode with no executor should have no errors")
	// There will be a warning about MoA tradeoffs (N>=2), but no error.
	_ = warnings
}

func TestValidateModeCodeNoExecutor(t *testing.T) {
	// code mode without an executor → hard error.
	models := buildModels(nil, "agy", "grok")
	errors, warnings := ValidateModeConfig(ModeCode, []string{"agy", "grok"}, "", nil, models)

	assert.NotEmpty(t, errors, "code mode without executor should produce an error")
	foundExecError := false
	for _, e := range errors {
		if strings.Contains(e, "executor") {
			foundExecError = true
			break
		}
	}
	assert.True(t, foundExecError, "error should mention executor requirement")
	_ = warnings
}

func TestValidateModeAgentNoExecutor(t *testing.T) {
	// agent mode without an executor → hard error (same as code).
	models := buildModels(nil, "agy", "grok")
	errors, _ := ValidateModeConfig(ModeAgent, []string{"agy", "grok"}, "", nil, models)
	assert.NotEmpty(t, errors, "agent mode without executor should produce an error")
}

func TestValidateModeResponderNotReadOnly(t *testing.T) {
	// A responder with read_only=false in code mode → warning.
	models := buildModels(map[string]config.ModelConfig{
		"aider": {
			Name:            "aider",
			Command:         `aider "{prompt}"`,
			ExecuteCommand:  `aider --yes "{prompt}"`,
			ReadOnly:        "false",
			Timeout:         120,
			ExecutorCapable: true,
		},
	}, "agy", "aider")

	errors, warnings := ValidateModeConfig(ModeCode, []string{"aider"}, "agy", nil, models)

	assert.Empty(t, errors, "read_only=false responder should not cause errors")
	foundWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "read_only=false") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "should warn about read_only=false responder")
}

func TestValidateModeResearchWithExecutorWarning(t *testing.T) {
	// research mode + executor specified → warning (executor will be ignored).
	models := buildModels(nil, "agy", "codex")
	errors, warnings := ValidateModeConfig(ModeResearch, []string{"agy"}, "codex", nil, models)

	assert.Empty(t, errors, "research + executor should not error")
	foundIgnoredWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "ignored") {
			foundIgnoredWarning = true
			break
		}
	}
	assert.True(t, foundIgnoredWarning, "should warn that executor is ignored in research mode")
}

func TestValidateModeSingleResponderWarning(t *testing.T) {
	// N=1 responder → warning about MoA benefit lost.
	models := buildModels(nil, "agy", "codex")
	errors, warnings := ValidateModeConfig(ModeCode, []string{"agy"}, "codex", nil, models)

	assert.Empty(t, errors)
	foundMoAWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "MoA benefit") || strings.Contains(w, "1 responder") {
			foundMoAWarning = true
			break
		}
	}
	assert.True(t, foundMoAWarning, "should warn about MoA benefit loss with N=1")
}

func TestValidateModeZeroRespondersError(t *testing.T) {
	// 0 responders → error (at least one required).
	models := buildModels(nil, "codex")
	errors, _ := ValidateModeConfig(ModeCode, []string{}, "codex", nil, models)
	assert.NotEmpty(t, errors, "zero responders should be an error")
}

func TestValidateModeUnknownResponderError(t *testing.T) {
	models := buildModels(nil, "agy")
	errors, _ := ValidateModeConfig(ModeCode, []string{"ghost"}, "agy", nil, models)
	foundUnknown := false
	for _, e := range errors {
		if strings.Contains(e, "ghost") {
			foundUnknown = true
			break
		}
	}
	assert.True(t, foundUnknown, "should error on unknown responder")
}

func TestValidateModeUnknownExecutorError(t *testing.T) {
	models := buildModels(nil, "agy")
	errors, _ := ValidateModeConfig(ModeCode, []string{"agy"}, "ghost", nil, models)
	foundUnknown := false
	for _, e := range errors {
		if strings.Contains(e, "ghost") {
			foundUnknown = true
			break
		}
	}
	assert.True(t, foundUnknown, "should error on unknown executor")
}

func TestValidateModeUnknownSynthesizerError(t *testing.T) {
	models := buildModels(nil, "agy", "codex")
	errors, _ := ValidateModeConfig(ModeCode, []string{"agy"}, "codex", strPtr("phantom"), models)
	foundUnknown := false
	for _, e := range errors {
		if strings.Contains(e, "phantom") {
			foundUnknown = true
			break
		}
	}
	assert.True(t, foundUnknown, "should error on unknown synthesizer")
}

func TestValidateModeSynthesizerIsResponderWarning(t *testing.T) {
	models := buildModels(nil, "agy", "codex")
	errors, warnings := ValidateModeConfig(ModeCode, []string{"agy"}, "codex", strPtr("agy"), models)

	assert.Empty(t, errors)
	foundBias := false
	for _, w := range warnings {
		if strings.Contains(w, "bias") && strings.Contains(w, "agy") {
			foundBias = true
			break
		}
	}
	assert.True(t, foundBias, "should warn about synthesizer=responder bias")
}

func TestValidateModeSynthesizerNotReadOnlyWarning(t *testing.T) {
	// synthesizer with read_only="maybe" → warning about tool execution.
	models := buildModels(map[string]config.ModelConfig{
		"grok": {
			Name:            "grok",
			Command:         `grok -p "{prompt}"`,
			ExecuteCommand:  `grok -p "{prompt}"`,
			ReadOnly:        "maybe",
			Timeout:         120,
			ExecutorCapable: true,
		},
	}, "agy", "codex", "grok")

	errors, warnings := ValidateModeConfig(ModeCode, []string{"agy"}, "codex", strPtr("grok"), models)

	assert.Empty(t, errors)
	foundNotReadOnly := false
	for _, w := range warnings {
		if strings.Contains(w, "not truly read-only") {
			foundNotReadOnly = true
			break
		}
	}
	assert.True(t, foundNotReadOnly, "should warn about non-read-only synthesizer")
}

func TestValidateModeMultipleRespondersMoATradeoffWarning(t *testing.T) {
	// N>=2 → MoA tradeoff warning.
	models := buildModels(nil, "agy", "grok", "codex")
	errors, warnings := ValidateModeConfig(ModeCode, []string{"agy", "grok"}, "codex", nil, models)

	assert.Empty(t, errors)
	foundTradeoff := false
	for _, w := range warnings {
		if strings.Contains(w, "tradeoff") || strings.Contains(w, "MoA active") {
			foundTradeoff = true
			break
		}
	}
	assert.True(t, foundTradeoff, "should warn about MoA tradeoffs with N>=2")
}

// ---------------------------------------------------------------------------
// FormatOutput tests
// ---------------------------------------------------------------------------

func TestFormatOutputResearch(t *testing.T) {
	synth := "synthesized conclusion text"
	result := convene.ConveneResult{
		Task:      "research task",
		Mode:      "research",
		Responses: map[string]string{"agy": "agy response", "grok": "grok response"},
		Synthesis: &synth,
		Metadata: map[string]interface{}{
			"responder_count": 2,
			"success_count":   2,
			"agy_elapsed":     "1.0s",
			"grok_elapsed":    "1.5s",
			"total_elapsed":   "2.5s",
		},
	}

	output := FormatOutput(result, ModeResearch)

	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "research")
	assert.Contains(t, output, "research task")
	assert.Contains(t, output, "Synthesis")
	assert.Contains(t, output, synth)
	assert.Contains(t, output, "agy response")
	assert.Contains(t, output, "grok response")
}

func TestFormatOutputResearchNoSynthesis(t *testing.T) {
	result := convene.ConveneResult{
		Task:      "research task",
		Mode:      "research",
		Responses: map[string]string{"agy": "agy response"},
		Synthesis: nil,
		Metadata: map[string]interface{}{
			"responder_count": 1,
			"success_count":   1,
		},
	}

	output := FormatOutput(result, ModeResearch)

	// Should show a message about no synthesis available.
	assert.Contains(t, output, "No synthesis")
	assert.Contains(t, output, "agy response")
}

func TestFormatOutputCode(t *testing.T) {
	exec := "code execution result"
	synth := "synthesis guidance"
	result := convene.ConveneResult{
		Task:      "code task",
		Mode:      "code",
		Responses: map[string]string{"agy": "agy response"},
		Synthesis: &synth,
		Execution: &exec,
		Metadata: map[string]interface{}{
			"responder_count": 1,
			"success_count":   1,
			"executor_success": true,
		},
	}

	output := FormatOutput(result, ModeCode)

	assert.Contains(t, output, "Execution")
	assert.Contains(t, output, exec)
	assert.Contains(t, output, "Synthesis")
	assert.Contains(t, output, synth)
}

func TestFormatOutputAgent(t *testing.T) {
	exec := "agent execution result"
	result := convene.ConveneResult{
		Task:      "agent task",
		Mode:      "agent",
		Responses: map[string]string{"agy": "agy response"},
		Synthesis: nil,
		Execution: &exec,
		Metadata: map[string]interface{}{
			"responder_count":  1,
			"success_count":    1,
			"executor_success": true,
		},
	}

	output := FormatOutput(result, ModeAgent)

	assert.Contains(t, output, "Execution")
	assert.Contains(t, output, exec)
}

func TestFormatOutputCodeNoExecution(t *testing.T) {
	// Execution failed → should show failure message.
	result := convene.ConveneResult{
		Task:      "code task",
		Mode:      "code",
		Responses: map[string]string{"agy": "agy response"},
		Synthesis: nil,
		Execution: nil,
		Metadata: map[string]interface{}{
			"responder_count": 1,
			"success_count":   1,
			"executor_failed": "err=some error stderr=some stderr",
		},
	}

	output := FormatOutput(result, ModeCode)

	assert.Contains(t, output, "failed", "should indicate execution failure")
}

func TestFormatOutputIncludesMetadata(t *testing.T) {
	result := convene.ConveneResult{
		Task:      "task",
		Mode:      "research",
		Responses: map[string]string{"agy": "resp"},
		Synthesis: nil,
		Metadata: map[string]interface{}{
			"responder_count": 1,
			"success_count":   1,
			"total_elapsed":   "1.5s",
		},
	}

	output := FormatOutput(result, ModeResearch)

	assert.Contains(t, output, "Metadata")
	assert.Contains(t, output, "1/1 succeeded")
}

func TestFormatOutputUnknownMode(t *testing.T) {
	result := convene.ConveneResult{
		Task:     "task",
		Mode:     "bogus",
		Metadata: map[string]interface{}{},
	}

	output := FormatOutput(result, Mode("bogus"))
	assert.Contains(t, output, "Unknown mode")
}

// ---------------------------------------------------------------------------
// Mode type tests
// ---------------------------------------------------------------------------

func TestModeConstants(t *testing.T) {
	assert.Equal(t, Mode("research"), ModeResearch)
	assert.Equal(t, Mode("code"), ModeCode)
	assert.Equal(t, Mode("agent"), ModeAgent)
}

func TestFormatOutputTruncatesLongTask(t *testing.T) {
	longTask := strings.Repeat("A", 300) // 300 chars, longer than truncate limit (200)
	result := convene.ConveneResult{
		Task:     longTask,
		Mode:     "research",
		Metadata: map[string]interface{}{},
	}

	output := FormatOutput(result, ModeResearch)

	// The task should be truncated in the header.
	assert.Contains(t, output, "...")
	// Should not contain the full 300-char task in the header line.
	headerLine := strings.Split(output, "\n")[2] // "Task: ..."
	assert.Less(t, len(headerLine), 250, "task should be truncated in header")
}

func TestValidateModeConfigValidNoErrors(t *testing.T) {
	// A fully valid code-mode configuration.
	models := buildModels(nil, "agy", "grok", "codex")
	errors, _ := ValidateModeConfig(ModeCode, []string{"agy", "grok"}, "codex", nil, models)
	require.Empty(t, errors, "valid config should produce no errors")
}
