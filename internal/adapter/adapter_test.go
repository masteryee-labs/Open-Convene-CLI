// adapter_test.go — tests for the adapter package (S6).
//
// Strategy: adapter tests do NOT call real AI CLIs (agy/codex/devin/grok etc.).
// Instead we test:
//   - GetCommand() command assembly (pure string logic, no subprocess).
//   - SupportsReadOnly() based on Config.ReadOnly values.
//   - GetAdapter() factory: correct type for each name, error for unknown.
//   - AiderAdapter.Respond() returns an error (overridden behavior).
//   - ReplacePrompt() placeholder substitution + shell escaping.
//   - RunCommand() with a real but trivial shell command (echo) + timeout.
//   - DetectAvailableAdapters() returns all 9 CLIs sorted by name.
//
// The echo command is used for RunCommand tests because it exists on all
// platforms, produces stdout, and exits 0 — satisfying Success=true criteria.

package adapter

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock adapter (for testing the Adapter interface contract)
// ---------------------------------------------------------------------------

// MockAdapter implements the Adapter interface for unit-testing code that
// depends on adapters without invoking real CLI processes.
type MockAdapter struct {
	Name         string
	ResponseText string
	ShouldFail   bool
	IsReadOnly   bool
}

func (m *MockAdapter) Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error) {
	if m.ShouldFail {
		return AdapterResult{Success: false, Stderr: "mock failure"}, nil
	}
	return AdapterResult{Stdout: m.ResponseText, Success: true, ReturnCode: 0}, nil
}

func (m *MockAdapter) Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error) {
	if m.ShouldFail {
		return AdapterResult{Success: false, Stderr: "mock failure"}, nil
	}
	return AdapterResult{Stdout: "executed: " + m.ResponseText, Success: true, ReturnCode: 0}, nil
}

func (m *MockAdapter) SupportsReadOnly() bool { return m.IsReadOnly }

func (m *MockAdapter) GetCommand(prompt string, mode string) string {
	return "mock " + prompt
}

// Compile-time check that MockAdapter satisfies the Adapter interface.
var _ Adapter = (*MockAdapter)(nil)

// ---------------------------------------------------------------------------
// GetCommand tests
// ---------------------------------------------------------------------------

func TestBaseAdapterGetCommandRespondMode(t *testing.T) {
	cfg := config.ModelConfig{
		Command:        `agy -p "{prompt}"`,
		ExecuteCommand: `agy --dangerously-skip-permissions "{prompt}"`,
	}
	b := &BaseAdapter{Name: "agy", Config: cfg}

	cmd := b.GetCommand("hello world", "respond")
	// Should use the Command template (respond mode).
	assert.Contains(t, cmd, "agy -p")
	assert.Contains(t, cmd, "hello world")
	// Should NOT use the execute command.
	assert.NotContains(t, cmd, "dangerously-skip-permissions")
}

func TestBaseAdapterGetCommandExecuteMode(t *testing.T) {
	cfg := config.ModelConfig{
		Command:        `agy -p "{prompt}"`,
		ExecuteCommand: `agy --dangerously-skip-permissions "{prompt}"`,
	}
	b := &BaseAdapter{Name: "agy", Config: cfg}

	cmd := b.GetCommand("hello world", "execute")
	// Should use the ExecuteCommand template (execute mode).
	assert.Contains(t, cmd, "dangerously-skip-permissions")
	assert.Contains(t, cmd, "hello world")
}

func TestBaseAdapterGetCommandExecuteFallbackToCommand(t *testing.T) {
	// When ExecuteCommand is empty, execute mode falls back to Command.
	cfg := config.ModelConfig{
		Command:        `grok -p "{prompt}"`,
		ExecuteCommand: "", // empty
	}
	b := &BaseAdapter{Name: "grok", Config: cfg}

	cmd := b.GetCommand("test prompt", "execute")
	assert.Contains(t, cmd, "grok -p")
	assert.Contains(t, cmd, "test prompt")
}

func TestBaseAdapterGetCommandReplacesPrompt(t *testing.T) {
	cfg := config.ModelConfig{
		Command: `echo "{prompt}"`,
	}
	b := &BaseAdapter{Name: "echo", Config: cfg}

	cmd := b.GetCommand("say hello", "respond")
	// {prompt} should be replaced (and shell-escaped).
	assert.NotContains(t, cmd, "{prompt}")
	assert.Contains(t, cmd, "say hello")
}

// ---------------------------------------------------------------------------
// SupportsReadOnly tests
// ---------------------------------------------------------------------------

func TestSupportsReadOnly(t *testing.T) {
	tests := []struct {
		readOnly string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"maybe", false},
		{"", false},
		{"bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.readOnly, func(t *testing.T) {
			b := &BaseAdapter{Config: config.ModelConfig{ReadOnly: tt.readOnly}}
			assert.Equal(t, tt.expected, b.SupportsReadOnly())
		})
	}
}

func TestAgyAdapterSupportsReadOnly(t *testing.T) {
	// agy has read_only="maybe" → SupportsReadOnly should be false.
	cfg := config.ModelConfig{ReadOnly: "maybe", Command: `agy -p "{prompt}"`}
	a := &AgyAdapter{BaseAdapter{Name: "agy", Config: cfg}}
	assert.False(t, a.SupportsReadOnly(), "agy read_only=maybe → SupportsReadOnly=false")
}

func TestCodexAdapterSupportsReadOnly(t *testing.T) {
	// codex has read_only="true" → SupportsReadOnly should be true.
	cfg := config.ModelConfig{ReadOnly: "true", Command: `codex exec --sandbox read-only "{prompt}"`}
	c := &CodexAdapter{BaseAdapter{Name: "codex", Config: cfg}}
	assert.True(t, c.SupportsReadOnly(), "codex read_only=true → SupportsReadOnly=true")
}

func TestDevinAdapterSupportsReadOnly(t *testing.T) {
	// devin has read_only="maybe" → SupportsReadOnly should be false.
	cfg := config.ModelConfig{ReadOnly: "maybe", Command: `devin -p "{prompt}"`}
	d := &DevinAdapter{BaseAdapter{Name: "devin", Config: cfg}}
	assert.False(t, d.SupportsReadOnly(), "devin read_only=maybe → SupportsReadOnly=false")
}

func TestAiderAdapterSupportsReadOnly(t *testing.T) {
	// aider has read_only="false" → SupportsReadOnly should be false.
	cfg := config.ModelConfig{ReadOnly: "false", ExecuteCommand: `aider --yes --model sonnet "{prompt}"`}
	a := &AiderAdapter{BaseAdapter{Name: "aider", Config: cfg}}
	assert.False(t, a.SupportsReadOnly(), "aider read_only=false → SupportsReadOnly=false")
}

// ---------------------------------------------------------------------------
// AiderAdapter.Respond test (overridden to return error)
// ---------------------------------------------------------------------------

func TestAiderAdapterRespondReturnsError(t *testing.T) {
	cfg := config.ModelConfig{
		ReadOnly:        "false",
		Command:         "",
		ExecuteCommand:  `aider --yes --model sonnet "{prompt}"`,
		ExecutorCapable: true,
	}
	a := &AiderAdapter{BaseAdapter{Name: "aider", Config: cfg}}

	result, err := a.Respond(context.Background(), "test prompt", 10)
	assert.Error(t, err, "aider Respond should return an error")
	assert.False(t, result.Success)
	assert.Contains(t, err.Error(), "respond mode")
}

func TestAiderAdapterGetCommand(t *testing.T) {
	cfg := config.ModelConfig{
		ExecuteCommand: `aider --yes --model sonnet "{prompt}"`,
	}
	a := &AiderAdapter{BaseAdapter{Name: "aider", Config: cfg}}

	// Execute mode should use the execute_command template.
	cmd := a.GetCommand("write a function", "execute")
	assert.Contains(t, cmd, "aider")
	assert.Contains(t, cmd, "write a function")
}

// ---------------------------------------------------------------------------
// GetAdapter factory tests
// ---------------------------------------------------------------------------

func TestGetAdapterAllKnownNames(t *testing.T) {
	names := []string{"agy", "codex", "devin", "grok", "cursor", "kimi", "hermes", "aider", "opencode"}
	cfg := config.ModelConfig{Command: `test "{prompt}"`, ReadOnly: "maybe", ExecutorCapable: true}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			a, err := GetAdapter(name, cfg)
			require.NoError(t, err, "GetAdapter should succeed for known name %s", name)
			require.NotNil(t, a)

			// The factory should populate the Name field from the map key.
			// We can check via GetCommand or SupportsReadOnly (they use Config).
			// The BaseAdapter.Name is set, but it's not directly accessible via
			// the interface. Instead, verify the adapter works.
			cmd := a.GetCommand("test", "respond")
			assert.Contains(t, cmd, "test")
		})
	}
}

func TestGetAdapterCustomNameReturnsGenericAdapter(t *testing.T) {
	// Custom model names (not in the 9 built-in CLIs) should return a generic
	// BaseAdapter, enabling "single-CLI multi-model" configurations.
	cfg := config.ModelConfig{
		Command:         `devin --model glm-5.2 -p "{prompt}"`,
		ExecuteCommand:  `devin --model glm-5.2 --permission-mode dangerous "{prompt}"`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
	}
	a, err := GetAdapter("glm-5.2", cfg)
	require.NoError(t, err, "GetAdapter should succeed for custom model name")
	require.NotNil(t, a)

	// The generic adapter should use the command template from config.
	cmd := a.GetCommand("hello", "respond")
	assert.Contains(t, cmd, "devin --model glm-5.2")
	assert.Contains(t, cmd, "hello")

	// Execute mode should use ExecuteCommand.
	execCmd := a.GetCommand("hello", "execute")
	assert.Contains(t, execCmd, "permission-mode dangerous")
}

// ---------------------------------------------------------------------------
// ReplacePrompt tests
// ---------------------------------------------------------------------------

func TestReplacePromptBasic(t *testing.T) {
	result := ReplacePrompt(`echo "{prompt}"`, "hello world")
	assert.NotContains(t, result, "{prompt}")
	assert.Contains(t, result, "hello world")
}

func TestReplacePromptNoPlaceholder(t *testing.T) {
	// If the template has no {prompt}, it should be returned as-is (escaped).
	template := `echo static-text`
	result := ReplacePrompt(template, "ignored prompt")
	assert.Equal(t, template, result)
}

func TestReplacePromptShellEscape(t *testing.T) {
	// The prompt should be shell-escaped to prevent injection.
	// On Unix: $ and ` are escaped. On Windows: % is escaped.
	// We just verify the prompt content survives and {prompt} is gone.
	result := ReplacePrompt(`echo "{prompt}"`, `hello"; rm -rf /`)
	assert.NotContains(t, result, "{prompt}")
	// The double quote in the prompt should be escaped (backslash-quote).
	if runtime.GOOS != "windows" {
		assert.Contains(t, result, `\"`)
	}
}

// ---------------------------------------------------------------------------
// RunCommand tests (using real shell commands — echo / timeout)
// ---------------------------------------------------------------------------

func TestRunCommandEchoSuccess(t *testing.T) {
	// echo is available on all platforms (cmd /c echo on Windows, sh -c echo on Unix).
	result, err := RunCommand(context.Background(), "echo test-response-from-echo", 10)
	require.NoError(t, err, "echo should not return a Go error")
	assert.True(t, result.Success, "echo produces stdout + exit 0 → Success=true")
	assert.Equal(t, 0, result.ReturnCode)
	assert.Contains(t, result.Stdout, "test-response-from-echo")
}

func TestRunCommandEmptyString(t *testing.T) {
	result, err := RunCommand(context.Background(), "", 10)
	assert.Error(t, err, "empty command string should return error")
	assert.False(t, result.Success)
}

func TestRunCommandNonExistentCommand(t *testing.T) {
	// A command that does not exist should fail to start or return non-zero.
	result, _ := RunCommand(context.Background(), "this-command-does-not-exist-xyz123", 10)
	// The process may start (sh -c) and then the inner command fails,
	// or it may fail to start. Either way, Success should be false.
	assert.False(t, result.Success, "non-existent command should not succeed")
}

func TestRunCommandTimeout(t *testing.T) {
	// A command that runs longer than the timeout should time out.
	// Use a short ping count on Windows (process group kill is best-effort,
	// see S2 handoff §8.3). Even if the kill doesn't propagate to the
	// grandchild, the command completes in ~3s, well under the 5s assertion.
	var longCmd string
	if runtime.GOOS == "windows" {
		longCmd = "ping -n 4 127.0.0.1" // ~3s
	} else {
		longCmd = "sleep 10"
	}

	start := time.Now()
	result, err := RunCommand(context.Background(), longCmd, 1)
	elapsed := time.Since(start)

	assert.False(t, result.Success, "timed-out command should not succeed")
	assert.Error(t, err, "timeout should return a Go error")
	assert.Contains(t, err.Error(), "timeout")
	// Should complete well under the full command duration.
	// On Windows, if the process group kill doesn't propagate, ping -n 4
	// still completes in ~3s. On Unix, sleep 10 is killed in ~1s.
	assert.Less(t, elapsed, 6*time.Second, "should not wait the full command duration")
}

func TestRunCommandExitCodeNonZero(t *testing.T) {
	// A command that exits with non-zero (e.g. false on Unix, or exit 1 on Windows).
	var failCmd string
	if runtime.GOOS == "windows" {
		failCmd = "cmd /c exit 1"
	} else {
		failCmd = "false"
	}

	result, _ := RunCommand(context.Background(), failCmd, 10)
	// false/exit 1 exits non-zero → Success=false.
	assert.False(t, result.Success, "non-zero exit should not be Success")
	assert.NotEqual(t, 0, result.ReturnCode, "non-zero exit should have non-zero ReturnCode")
}

// ---------------------------------------------------------------------------
// DetectAvailableAdapters tests
// ---------------------------------------------------------------------------

func TestDetectAvailableAdaptersReturnsAllCLIs(t *testing.T) {
	results := DetectAvailableAdapters()
	assert.Len(t, results, 9, "should detect all 9 known CLIs")

	// Results should be sorted by Name.
	for i := 1; i < len(results); i++ {
		assert.True(t, results[i-1].Name <= results[i].Name,
			"results should be sorted by Name: %s <= %s", results[i-1].Name, results[i].Name)
	}

	// Verify all expected names are present.
	expectedNames := map[string]bool{
		"agy": false, "codex": false, "devin": false, "grok": false,
		"cursor": false, "kimi": false, "hermes": false,
		"aider": false, "opencode": false,
	}
	for _, r := range results {
		if _, ok := expectedNames[r.Name]; ok {
			expectedNames[r.Name] = true
		}
	}
	for name, found := range expectedNames {
		assert.True(t, found, "detect should include %s", name)
	}
}

func TestDetectAvailableAdaptersAiderCannotRespond(t *testing.T) {
	results := DetectAvailableAdapters()
	for _, r := range results {
		if r.Name == "aider" {
			assert.False(t, r.CanRespond, "aider should not be able to respond")
			assert.True(t, r.CanExecute, "aider should be able to execute")
			assert.Equal(t, "false", r.ReadOnly)
			return
		}
	}
	t.Fatal("aider not found in detect results")
}

func TestDetectAvailableAdaptersHasInstallCmds(t *testing.T) {
	results := DetectAvailableAdapters()
	for _, r := range results {
		assert.NotEmpty(t, r.InstallCmd, "%s should have an install command", r.Name)
	}
}

// ---------------------------------------------------------------------------
// MockAdapter contract test
// ---------------------------------------------------------------------------

func TestMockAdapterRespond(t *testing.T) {
	m := &MockAdapter{ResponseText: "mock response", IsReadOnly: true}
	result, err := m.Respond(context.Background(), "test", 10)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "mock response", result.Stdout)
}

func TestMockAdapterExecute(t *testing.T) {
	m := &MockAdapter{ResponseText: "mock response", IsReadOnly: true}
	result, err := m.Execute(context.Background(), "test", 10, "synth context")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Stdout, "executed")
}

func TestMockAdapterShouldFail(t *testing.T) {
	m := &MockAdapter{ShouldFail: true}
	result, err := m.Respond(context.Background(), "test", 10)
	require.NoError(t, err, "mock failure returns nil error, not Go error")
	assert.False(t, result.Success)
	assert.Contains(t, result.Stderr, "mock failure")
}

// ---------------------------------------------------------------------------
// GetCommand with special characters
// ---------------------------------------------------------------------------

func TestGetCommandWithSpecialCharsInPrompt(t *testing.T) {
	cfg := config.ModelConfig{Command: `echo "{prompt}"`}
	b := &BaseAdapter{Name: "test", Config: cfg}

	// Prompt with special shell characters should be escaped.
	cmd := b.GetCommand(`hello $HOME`, "respond")
	assert.NotContains(t, cmd, "{prompt}")
	// The $ should be escaped on Unix, or the text should at least survive.
	if runtime.GOOS != "windows" {
		assert.Contains(t, cmd, `\$HOME`)
	}
}
