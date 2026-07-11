// engine_test.go — tests for ConveneEngine.Run (S6).
//
// Strategy: all adapter calls are mocked via SetAdapterFactory(). No real CLI
// processes are invoked. The mock factory returns mockAdapter instances that
// implement the adapter.Adapter interface with controllable responses.
//
// Because Go test files don't share types across packages, this file defines
// its own mockAdapter (lowercase, unexported) separate from the one in
// adapter_test.go.

package convene

import (
	"context"
	"testing"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock adapter (package convene — independent from adapter_test.go's mock)
// ---------------------------------------------------------------------------

// mockAdapter implements adapter.Adapter for engine unit tests.
type mockAdapter struct {
	name         string
	responseText string
	shouldFail   bool
	isReadOnly   bool
}

func (m *mockAdapter) Respond(ctx context.Context, prompt string, timeout int) (adapter.AdapterResult, error) {
	if m.shouldFail {
		return adapter.AdapterResult{Success: false, Stderr: "mock failure"}, nil
	}
	return adapter.AdapterResult{
		Stdout:     m.responseText,
		Success:    true,
		ReturnCode: 0,
	}, nil
}

func (m *mockAdapter) Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (adapter.AdapterResult, error) {
	if m.shouldFail {
		return adapter.AdapterResult{Success: false, Stderr: "mock exec failure"}, nil
	}
	return adapter.AdapterResult{
		Stdout:     "executed: " + m.responseText,
		Success:    true,
		ReturnCode: 0,
	}, nil
}

func (m *mockAdapter) SupportsReadOnly() bool { return m.isReadOnly }

func (m *mockAdapter) GetCommand(prompt string, mode string) string {
	return "mock " + m.name + " " + prompt
}

// Compile-time check.
var _ adapter.Adapter = (*mockAdapter)(nil)

// mockFactory returns an AdapterFactory that creates mockAdapters with the
// given response texts and failure list.
//
//   - responses: map[modelName]responseText — controls what each mock returns.
//   - failNames: list of model names whose mock should fail (Success=false).
func mockFactory(responses map[string]string, failNames []string) AdapterFactory {
	failSet := make(map[string]bool, len(failNames))
	for _, n := range failNames {
		failSet[n] = true
	}
	return func(name string, cfg config.ModelConfig) (adapter.Adapter, error) {
		resp := responses[name]
		if resp == "" {
			resp = "default-mock-response"
		}
		return &mockAdapter{
			name:         name,
			responseText: resp,
			shouldFail:   failSet[name],
			isReadOnly:   true, // responders are expected to be read-only
		}, nil
	}
}

// ---------------------------------------------------------------------------
// Test config builder
// ---------------------------------------------------------------------------

// buildTestConfig creates a ConveneConfig with the given model names.
// All models are executor_capable with read_only="true" and timeout=120.
func buildTestConfig(modelNames ...string) *config.ConveneConfig {
	cfg := &config.ConveneConfig{
		Defaults: config.DefaultsConfig{
			Timeout:    120,
			Responders: modelNames,
			Executor:   modelNames[0],
		},
		Models: make(map[string]config.ModelConfig),
	}
	for _, name := range modelNames {
		cfg.Models[name] = config.ModelConfig{
			Name:            name,
			Command:         `mock "{prompt}"`,
			ExecuteCommand:  `mock exec "{prompt}"`,
			ReadOnly:        "true",
			Timeout:         120,
			ExecutorCapable: true,
		}
	}
	return cfg
}

// ---------------------------------------------------------------------------
// ConveneEngine.Run tests
// ---------------------------------------------------------------------------

func TestConveneResearchMode(t *testing.T) {
	cfg := buildTestConfig("agy", "grok")
	cfg.Defaults.Responders = []string{"agy", "grok"}
	cfg.Defaults.Executor = "codex" // research ignores executor

	// Add codex to config so executor reference is valid.
	cfg.Models["codex"] = config.ModelConfig{
		Name: "codex", Command: `mock "{prompt}"`, ExecuteCommand: `mock "{prompt}"`,
		ReadOnly: "true", Timeout: 120, ExecutorCapable: true,
	}

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":  "agy research response",
		"grok": "grok research response",
	}, nil))

	result, err := engine.Run(context.Background(), "research task", "research",
		[]string{"agy", "grok"}, "codex", nil)

	require.NoError(t, err, "research mode should not error with working responders")
	assert.Equal(t, "research", result.Mode)
	assert.Equal(t, "research task", result.Task)

	// Research mode should NOT execute.
	assert.Nil(t, result.Execution, "research mode should not produce execution output")

	// Both responders should have responses.
	assert.Len(t, result.Responses, 2)
	assert.Equal(t, "agy research response", result.Responses["agy"])
	assert.Equal(t, "grok research response", result.Responses["grok"])

	// No synthesizer was configured → synthesis should be nil.
	assert.Nil(t, result.Synthesis)

	// Metadata should record success count.
	assert.Equal(t, 2, result.Metadata["success_count"])
	assert.Equal(t, 2, result.Metadata["responder_count"])
}

func TestConveneCodeMode(t *testing.T) {
	cfg := buildTestConfig("agy", "grok", "codex")

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "agy response",
		"grok":  "grok response",
		"codex": "codex execution result",
	}, nil))

	result, err := engine.Run(context.Background(), "write a function", "code",
		[]string{"agy", "grok"}, "codex", nil)

	require.NoError(t, err)
	assert.Equal(t, "code", result.Mode)

	// Code mode should produce an execution result.
	require.NotNil(t, result.Execution, "code mode should produce execution output")
	assert.Contains(t, *result.Execution, "codex execution result")

	// Responses should be collected from both responders.
	assert.Len(t, result.Responses, 2)

	// No synthesizer → synthesis is nil.
	assert.Nil(t, result.Synthesis)

	// Executor success should be recorded.
	assert.Equal(t, true, result.Metadata["executor_success"])
}

func TestConveneAgentMode(t *testing.T) {
	cfg := buildTestConfig("agy", "grok", "codex")

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "agy response",
		"grok":  "grok response",
		"codex": "codex agent execution",
	}, nil))

	result, err := engine.Run(context.Background(), "deploy the app", "agent",
		[]string{"agy", "grok"}, "codex", nil)

	require.NoError(t, err)
	assert.Equal(t, "agent", result.Mode)

	// Agent mode should produce execution.
	require.NotNil(t, result.Execution)
	assert.Contains(t, *result.Execution, "codex agent execution")

	// Responses collected.
	assert.Len(t, result.Responses, 2)
}

func TestConveneResponderPartialFailure(t *testing.T) {
	cfg := buildTestConfig("agy", "grok", "codex")

	engine := NewConveneEngine(cfg)
	// agy fails, grok succeeds.
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "should not be used",
		"grok":  "grok working response",
		"codex": "codex result",
	}, []string{"agy"}))

	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy", "grok"}, "codex", nil)

	require.NoError(t, err, "partial failure should not abort the run")

	// Only grok's response should be in Responses (agy failed).
	assert.Len(t, result.Responses, 1)
	assert.Contains(t, result.Responses, "grok")
	assert.NotContains(t, result.Responses, "agy")

	// Success count should be 1 (only grok).
	assert.Equal(t, 1, result.Metadata["success_count"])

	// agy failure should be recorded in metadata.
	assert.Equal(t, false, result.Metadata["agy_success"])
}

func TestConveneAllRespondersFail(t *testing.T) {
	cfg := buildTestConfig("agy", "grok")

	engine := NewConveneEngine(cfg)
	// Both responders fail.
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":  "x",
		"grok": "x",
	}, []string{"agy", "grok"}))

	result, err := engine.Run(context.Background(), "task", "research",
		[]string{"agy", "grok"}, "", nil)

	// All responders failing is a fatal error.
	require.Error(t, err, "all responders failing should return an error")
	assert.Contains(t, err.Error(), "all responders failed")

	// Result should still be populated (with empty responses).
	assert.Equal(t, 0, result.Metadata["success_count"])
	assert.Empty(t, result.Responses)
	assert.Nil(t, result.Synthesis)
	assert.Nil(t, result.Execution)
}

func TestConveneNoSynthesizer(t *testing.T) {
	cfg := buildTestConfig("agy", "codex")
	cfg.Defaults.Responders = []string{"agy"}
	cfg.Defaults.Executor = "codex"

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "agy raw response",
		"codex": "codex executed",
	}, nil))

	// synthesizer = nil → executor doubles as synthesizer.
	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy"}, "codex", nil)

	require.NoError(t, err)
	// No synthesizer → Synthesis is nil.
	assert.Nil(t, result.Synthesis, "no synthesizer → Synthesis should be nil")

	// Executor should still execute (using raw responses as guidance).
	require.NotNil(t, result.Execution)
}

func TestConveneWithSynthesizer(t *testing.T) {
	cfg := buildTestConfig("agy", "grok", "codex")
	synthName := "grok" // grok doubles as synthesizer

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "agy response",
		"grok":  "grok synthesized conclusion",
		"codex": "codex execution",
	}, nil))

	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy"}, "codex", &synthName)

	require.NoError(t, err)

	// With a working synthesizer, Synthesis should be non-nil.
	require.NotNil(t, result.Synthesis, "synthesizer should produce synthesis")
	assert.Contains(t, *result.Synthesis, "grok synthesized conclusion")

	// Synthesizer success should be recorded.
	assert.Equal(t, true, result.Metadata["synthesizer_success"])

	// Execution should still happen.
	require.NotNil(t, result.Execution)
}

func TestConveneSynthesizerFailsFallsBackToNil(t *testing.T) {
	cfg := buildTestConfig("agy", "grok", "codex")
	synthName := "grok"

	engine := NewConveneEngine(cfg)
	// grok (synthesizer) fails; agy (responder) succeeds; codex (executor) succeeds.
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "agy response",
		"grok":  "should fail",
		"codex": "codex execution",
	}, []string{"grok"}))

	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy"}, "codex", &synthName)

	require.NoError(t, err, "synthesizer failure should not abort (fallback to nil)")

	// Synthesis should be nil (synthesizer failed).
	assert.Nil(t, result.Synthesis, "failed synthesizer → Synthesis=nil")

	// Executor should still execute using raw responses.
	require.NotNil(t, result.Execution, "executor should still run after synth failure")
}

func TestConveneUnknownMode(t *testing.T) {
	cfg := buildTestConfig("agy")

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{"agy": "resp"}, nil))

	result, err := engine.Run(context.Background(), "task", "invalid-mode",
		[]string{"agy"}, "", nil)

	require.Error(t, err, "unknown mode should return an error")
	assert.Contains(t, err.Error(), "unknown mode")
	assert.Equal(t, "invalid-mode", result.Mode)
}

func TestConveneResponderNotInConfig(t *testing.T) {
	cfg := buildTestConfig("agy")

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{"agy": "resp"}, nil))

	// "ghost" is not in config models.
	result, err := engine.Run(context.Background(), "task", "research",
		[]string{"agy", "ghost"}, "", nil)

	require.NoError(t, err, "unknown responder should be skipped, not fatal")
	// Only agy should be in responses.
	assert.Len(t, result.Responses, 1)
	// A warning should be recorded.
	warnings, ok := result.Metadata["responder_warnings"].([]string)
	require.True(t, ok, "responder_warnings should be recorded")
	assert.NotEmpty(t, warnings)
}

func TestConveneMetadataHasTiming(t *testing.T) {
	cfg := buildTestConfig("agy", "codex")
	cfg.Defaults.Responders = []string{"agy"}
	cfg.Defaults.Executor = "codex"

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{
		"agy":   "resp",
		"codex": "exec",
	}, nil))

	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy"}, "codex", nil)

	require.NoError(t, err)

	// agy_elapsed should exist in metadata.
	_, hasAgyElapsed := result.Metadata["agy_elapsed"]
	assert.True(t, hasAgyElapsed, "metadata should record agy_elapsed")

	// total_elapsed should exist.
	_, hasTotalElapsed := result.Metadata["total_elapsed"]
	assert.True(t, hasTotalElapsed, "metadata should record total_elapsed")
}

func TestConveneExecutorNotInConfig(t *testing.T) {
	cfg := buildTestConfig("agy")
	cfg.Defaults.Responders = []string{"agy"}
	cfg.Defaults.Executor = ""

	engine := NewConveneEngine(cfg)
	engine.SetAdapterFactory(mockFactory(map[string]string{"agy": "resp"}, nil))

	// code mode with executor "ghost" not in config.
	result, err := engine.Run(context.Background(), "task", "code",
		[]string{"agy"}, "ghost", nil)

	// Should not error (executor failure is recorded in metadata, not fatal).
	require.NoError(t, err)
	// Execution should be nil (executor not found).
	assert.Nil(t, result.Execution)
	// Executor error should be recorded.
	execErr, hasErr := result.Metadata["executor_error"]
	assert.True(t, hasErr)
	assert.Contains(t, execErr.(string), "not found")
}

func TestConveneSetAdapterFactoryReplacesDefault(t *testing.T) {
	// Verify that SetAdapterFactory actually replaces the factory.
	cfg := buildTestConfig("agy")
	engine := NewConveneEngine(cfg)

	customFactory := func(name string, cfg config.ModelConfig) (adapter.Adapter, error) {
		return &mockAdapter{name: name, responseText: "custom"}, nil
	}
	engine.SetAdapterFactory(customFactory)

	result, err := engine.Run(context.Background(), "task", "research",
		[]string{"agy"}, "", nil)

	require.NoError(t, err)
	assert.Equal(t, "custom", result.Responses["agy"],
		"SetAdapterFactory should replace the default factory")
}

func TestConveneNewConveneEngineHasDefaultFactory(t *testing.T) {
	// NewConveneEngine should set a non-nil default factory.
	cfg := buildTestConfig("agy")
	engine := NewConveneEngine(cfg)
	assert.NotNil(t, engine.adapterFactory, "default factory should be non-nil")
}
