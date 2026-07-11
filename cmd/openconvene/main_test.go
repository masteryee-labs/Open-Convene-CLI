// main_test.go — CLI integration tests for openconvene (S6 + CLI redesign).
//
// Strategy:
//   - --help tests truly run cobra's Execute() (no external services needed).
//   - Error-path tests (missing required flags) run cobra and check for errors.
//   - The "run" integration test uses a temp config with "echo" as the CLI
//     command — echo exists on all platforms, produces stdout, exits 0, so the
//     adapter's RunCommand succeeds without any real AI CLI. This tests the
//     full CLI pipeline (flag parsing → config load → mode validation →
//     engine.Run → FormatOutput) end-to-end.
//   - init and models tests use temp config files.
//
// We call buildRootCmd() (unexported, accessible since this is package main)
// and set args via SetArgs(), then call Execute(). main() is NOT called (it
// invokes os.Exit which would kill the test process).

package main

import (
	"fmt"
	"io"
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

// captureStdout temporarily replaces os.Stdout with a pipe, runs fn, and
// returns whatever was written to stdout during fn's execution.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to create pipe")

	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fnErr := fn()
	w.Close()

	out, err := io.ReadAll(r)
	require.NoError(t, err, "failed to read captured stdout")

	return string(out), fnErr
}

// captureStderr captures os.Stderr output during fn's execution.
func captureStderr(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err, "failed to create pipe")

	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	fnErr := fn()
	w.Close()

	out, err := io.ReadAll(r)
	require.NoError(t, err, "failed to read captured stderr")

	return string(out), fnErr
}

// writeEchoConfig writes a config file where the "command" is "echo" — a
// real shell command available on all platforms. This lets the full CLI
// pipeline run without any real AI CLI installed.
//
// IMPORTANT: the model names must be real adapter names (agy, grok, codex)
// because the engine's default factory (adapter.GetAdapter) only recognizes
// the 9 known adapter names. The echo commands in the config override the
// real CLI commands, so no real CLI is invoked.
func writeEchoConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "models.yaml")

	content := `defaults:
  timeout: 10
  responders:
    - agy
    - grok
  executor: codex
  synthesizer: null

models:
  agy:
    command: 'echo agy-response "{prompt}"'
    execute_command: 'echo agy-exec "{prompt}"'
    read_only: "true"
    timeout: 10
    executor_capable: true
    extra_args: []

  grok:
    command: 'echo grok-response "{prompt}"'
    execute_command: 'echo grok-exec "{prompt}"'
    read_only: "true"
    timeout: 10
    executor_capable: true
    extra_args: []

  codex:
    command: 'echo codex-response "{prompt}"'
    execute_command: 'echo codex-exec "{prompt}"'
    read_only: "true"
    timeout: 10
    executor_capable: true
    extra_args: []`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// execRootCmd builds the root command, sets the given args, and executes it.
func execRootCmd(args []string) error {
	cmd := buildRootCmd()
	cmd.SetArgs(args)
	return cmd.Execute()
}

// ---------------------------------------------------------------------------
// --help tests (truly run cobra)
// ---------------------------------------------------------------------------

func TestCLIHelp(t *testing.T) {
	// openconvene --help should not error.
	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{"--help"})
	})

	require.NoError(t, err, "openconvene --help should not error")
	assert.Contains(t, output, "openconvene")
	assert.Contains(t, output, "Usage")
}

func TestCLIAskHelp(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{"ask", "--help"})
	})

	require.NoError(t, err, "openconvene ask --help should not error")
	assert.Contains(t, output, "ask")
	assert.Contains(t, output, "research")
}

func TestCLIAgentHelp(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{"agent", "--help"})
	})

	require.NoError(t, err, "openconvene agent --help should not error")
	assert.Contains(t, output, "agent")
}

func TestCLIModelsHelp(t *testing.T) {
	err := execRootCmd([]string{"models", "--help"})
	assert.NoError(t, err, "models --help should not error")
}

func TestCLIDetectHelp(t *testing.T) {
	err := execRootCmd([]string{"detect", "--help"})
	assert.NoError(t, err, "detect --help should not error")
}

func TestCLIInitHelp(t *testing.T) {
	err := execRootCmd([]string{"init", "--help"})
	assert.NoError(t, err, "init --help should not error")
}

func TestCLICheckHelp(t *testing.T) {
	err := execRootCmd([]string{"check", "--help"})
	assert.NoError(t, err, "check --help should not error")
}

// ---------------------------------------------------------------------------
// Error-path tests
// ---------------------------------------------------------------------------

func TestCLIRunInvalidMode(t *testing.T) {
	configPath := writeEchoConfig(t)

	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{
			"run",
			"--mode", "bogus",
			"--task", "test",
			"--config", configPath,
		})
	})

	assert.Error(t, err, "invalid mode should produce an error")
	assert.Contains(t, err.Error(), "mode")
}

func TestCLIRunNoConfig(t *testing.T) {
	// Test that autoGenerateConfig creates a valid config file.
	configPath := filepath.Join(t.TempDir(), "nonexistent.yaml")
	cfg, path, err := autoGenerateConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, configPath, path)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, cfg.Defaults.Responders)
	assert.NotEmpty(t, cfg.Defaults.Executor)

	// The config file should exist on disk.
	_, statErr := os.Stat(configPath)
	assert.NoError(t, statErr, "auto-generated config file should exist")
}

func TestCLIRunEmptyTask(t *testing.T) {
	configPath := writeEchoConfig(t)

	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{
			"run",
			"--mode", "research",
			"--task", "",
			"--config", configPath,
		})
	})

	assert.Error(t, err, "empty task should produce an error")
}

func TestCLIAskMissingTask(t *testing.T) {
	// ask with no task → enters REPL. With empty stdin (EOF), the REPL
	// exits immediately without error.
	configPath := writeEchoConfig(t)

	_, err := captureStdout(t, func() error {
		return execRootCmd([]string{"ask", "--config", configPath})
	})

	// REPL with EOF stdin should exit cleanly (no error).
	assert.NoError(t, err, "ask with no task should enter REPL and exit on EOF")
}

func TestCLIAgentMissingTask(t *testing.T) {
	// agent with no task → enters REPL. With empty stdin (EOF), exits.
	configPath := writeEchoConfig(t)

	_, err := captureStdout(t, func() error {
		return execRootCmd([]string{"agent", "--config", configPath})
	})

	assert.NoError(t, err, "agent with no task should enter REPL and exit on EOF")
}

// ---------------------------------------------------------------------------
// Full pipeline integration tests — new positional arg style (echo config)
// ---------------------------------------------------------------------------

func TestCLIDefaultCodeMode(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"--responders", "agy",
			"--executor", "codex",
			"--config", configPath,
			"write-test-code",
		})
	})

	require.NoError(t, err, "default code mode with echo config should succeed")
	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "code")
	assert.Contains(t, output, "codex")
}

func TestCLIAskMode(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--responders", "agy,grok",
			"--config", configPath,
			"test-task",
		})
	})

	require.NoError(t, err, "ask mode with echo config should succeed")
	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "research")
	assert.Contains(t, output, "agy-response")
	assert.Contains(t, output, "grok-response")
}

func TestCLIAgentMode(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"agent",
			"--responders", "agy,grok",
			"--executor", "codex",
			"--config", configPath,
			"deploy-test",
		})
	})

	require.NoError(t, err, "agent mode with echo config should succeed")
	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "Execution")
}

func TestCLIAskWithSynthesizer(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--responders", "agy",
			"--synthesizer", "grok",
			"--config", configPath,
			"test-synth",
		})
	})

	require.NoError(t, err, "ask with synthesizer should succeed")
	assert.Contains(t, output, "Synthesis")
}

func TestCLIVerbose(t *testing.T) {
	configPath := writeEchoConfig(t)

	// Verbose output goes to stderr; capture stdout for the formatted result.
	output, _ := captureStdout(t, func() error {
		_, err := captureStderr(t, func() error {
			return execRootCmd([]string{
				"ask",
				"--responders", "agy",
				"--config", configPath,
				"--verbose",
				"verbose-test",
			})
		})
		return err
	})

	assert.Contains(t, output, "=== Convene Result ===")
}

func TestCLIPrintFlag(t *testing.T) {
	configPath := writeEchoConfig(t)

	// -p flag should be accepted (no-op since CLI is already non-interactive).
	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"-p",
			"--responders", "agy",
			"--config", configPath,
			"print-test",
		})
	})

	require.NoError(t, err, "-p flag should be accepted")
	assert.Contains(t, output, "=== Convene Result ===")
}

// ---------------------------------------------------------------------------
// Backward-compatibility alias tests (hidden commands)
// ---------------------------------------------------------------------------

func TestCLIRunAliasResearch(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"run",
			"--mode", "research",
			"--responders", "agy,grok",
			"--task", "test-task",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "run alias research with echo config should succeed")
	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "research")
}

func TestCLIRunAliasCode(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"run",
			"--mode", "code",
			"--responders", "agy",
			"--executor", "codex",
			"--task", "write-test-code",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "run alias code with echo config should succeed")
	assert.Contains(t, output, "=== Convene Result ===")
	assert.Contains(t, output, "codex")
}

func TestCLIRunAliasMissingTask(t *testing.T) {
	// run --mode research (missing required --task) → cobra error.
	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{"run", "--mode", "research"})
	})

	assert.Error(t, err, "missing required --task flag should produce an error")
}

func TestCLIRunAliasMissingMode(t *testing.T) {
	// run --task "test" (missing required --mode) → cobra error.
	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{"run", "--task", "test"})
	})

	assert.Error(t, err, "missing required --mode flag should produce an error")
}

func TestCLIListModelsAlias(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"list-models",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "list-models alias should succeed with a valid config")
	assert.Contains(t, output, "MODEL")
	assert.Contains(t, output, "agy")
}

func TestCLIConfigInitAlias(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "models.yaml")

	err := execRootCmd([]string{"config", "init", "--path", outPath})
	require.NoError(t, err, "config init alias should succeed")

	info, err := os.Stat(outPath)
	require.NoError(t, err, "config file should exist after config init alias")
	assert.Greater(t, info.Size(), int64(0))
}

func TestCLIConfigValidateAlias(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"config", "validate",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "config validate alias on valid config should succeed")
	assert.Contains(t, output, "valid")
}

// ---------------------------------------------------------------------------
// init command test
// ---------------------------------------------------------------------------

func TestCLIInit(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "models.yaml")

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"init",
			"--path", outPath,
		})
	})

	require.NoError(t, err, "init should succeed")

	info, err := os.Stat(outPath)
	require.NoError(t, err, "config file should exist after init")
	assert.Greater(t, info.Size(), int64(0))

	assert.Contains(t, output, outPath)
}

func TestCLIInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "models.yaml")

	// First init succeeds.
	err := execRootCmd([]string{"init", "--path", outPath})
	require.NoError(t, err)

	// Second init should fail (file already exists).
	_, err = captureStderr(t, func() error {
		return execRootCmd([]string{"init", "--path", outPath})
	})
	assert.Error(t, err, "init should refuse to overwrite existing file")
}

// ---------------------------------------------------------------------------
// check command test
// ---------------------------------------------------------------------------

func TestCLICheck(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"check",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "check on valid config should succeed")
	assert.Contains(t, output, "valid")
}

func TestCLICheckMissingConfig(t *testing.T) {
	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{
			"check",
			"--config", filepath.Join(t.TempDir(), "nonexistent.yaml"),
		})
	})

	assert.Error(t, err, "check on missing config should error")
}

// ---------------------------------------------------------------------------
// models command test
// ---------------------------------------------------------------------------

func TestCLIModels(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"models",
			"--config", configPath,
		})
	})

	require.NoError(t, err, "models should succeed with a valid config")
	assert.Contains(t, output, "MODEL")
	assert.Contains(t, output, "READ_ONLY")
	assert.Contains(t, output, "agy")
	assert.Contains(t, output, "grok")
	assert.Contains(t, output, "codex")
	assert.Contains(t, output, "Total")
}

// ---------------------------------------------------------------------------
// detect command test
// ---------------------------------------------------------------------------

func TestCLIDetect(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{"detect"})
	})

	require.NoError(t, err, "detect should succeed without a config")
	assert.Contains(t, output, "CLI")
	assert.Contains(t, output, "INSTALLED")
	assert.Contains(t, output, "agy")
	assert.Contains(t, output, "codex")
	assert.Contains(t, output, "Installed:")
}

// ---------------------------------------------------------------------------
// No-args test
// ---------------------------------------------------------------------------

func TestCLINoArgsShowsHelp(t *testing.T) {
	// Running with no args → enters REPL (code mode). With empty stdin (EOF),
	// the REPL exits immediately. The welcome banner should be printed.
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{"--config", configPath})
	})

	// REPL with EOF stdin should exit cleanly.
	assert.NoError(t, err)
	// Should show the REPL welcome banner.
	assert.Contains(t, output, "OpenConveneCLI")
	assert.Contains(t, output, "REPL")
}

// ---------------------------------------------------------------------------
// Unknown command test
// ---------------------------------------------------------------------------

func TestCLIUnknownCommand(t *testing.T) {
	// An unknown flag should produce an error (cobra rejects unknown flags).
	_, err := captureStderr(t, func() error {
		return execRootCmd([]string{"--nonexistent-flag"})
	})

	assert.Error(t, err, "unknown flag should produce an error")
}

// ---------------------------------------------------------------------------
// Timeout override test
// ---------------------------------------------------------------------------

func TestCLITimeoutOverride(t *testing.T) {
	configPath := writeEchoConfig(t)

	_, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--responders", "agy",
			"--config", configPath,
			"--timeout", "30",
			"timeout-test",
		})
	})

	assert.NoError(t, err, "ask with timeout override should succeed")
}

// ---------------------------------------------------------------------------
// stdin task test ("-")
// ---------------------------------------------------------------------------

func TestCLITaskFromStdin(t *testing.T) {
	configPath := writeEchoConfig(t)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		fmt.Fprint(w, "task-from-stdin")
		w.Close()
	}()

	_, err = captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--responders", "agy",
			"--config", configPath,
			"-",
		})
	})

	assert.NoError(t, err, "ask with task from stdin should succeed")
}

// ---------------------------------------------------------------------------
// filterResponders / filterExecutors helper tests
// ---------------------------------------------------------------------------

func TestFilterResponders(t *testing.T) {
	output := filterResponders(nil, true)
	assert.Empty(t, output)
}

func TestFilterExecutors(t *testing.T) {
	output := filterExecutors(nil, true)
	assert.Empty(t, output)
}

// ---------------------------------------------------------------------------
// captureStdout helper test
// ---------------------------------------------------------------------------

func TestCaptureStdoutHelper(t *testing.T) {
	output, err := captureStdout(t, func() error {
		fmt.Print("test-capture-output")
		return nil
	})

	require.NoError(t, err)
	assert.Contains(t, output, "test-capture-output")
}

// ---------------------------------------------------------------------------
// Output format test
// ---------------------------------------------------------------------------

func TestCLIAskOutputFormat(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--responders", "agy,grok",
			"--config", configPath,
			"format-check",
		})
	})

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.NotEmpty(t, lines)
	assert.Contains(t, lines[0], "=== Convene Result ===")
}

// ---------------------------------------------------------------------------
// REPL interactive mode tests
// ---------------------------------------------------------------------------

// runREPLWithInput runs the CLI with the given args and pipes the provided
// stdin text. Returns stdout output and error.
func runREPLWithInput(t *testing.T, args []string, stdinText string) (string, error) {
	t.Helper()

	// Replace stdin.
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	go func() {
		fmt.Fprint(w, stdinText)
		w.Close()
	}()

	return captureStdout(t, func() error {
		return execRootCmd(args)
	})
}

func TestREPLWelcomeBanner(t *testing.T) {
	configPath := writeEchoConfig(t)

	// Enter REPL with /exit immediately.
	output, err := runREPLWithInput(t, []string{"--config", configPath}, "/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "OpenConveneCLI")
	assert.Contains(t, output, "REPL")
	assert.Contains(t, output, "Mode:")
}

func TestREPLAskModeBanner(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"ask", "--config", configPath}, "/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "ask")
}

func TestREPLAgentModeBanner(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"agent", "--config", configPath}, "/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "agent")
}

func TestREPLHelpCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/help\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "/help")
	assert.Contains(t, output, "/mode")
	assert.Contains(t, output, "/usage")
	assert.Contains(t, output, "/responders")
}

func TestREPLModeSwitch(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"ask", "--config", configPath}, "/mode code\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Mode switched to: code")
}

func TestREPLRespondersCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/responders agy,grok\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Responders set to: agy, grok")
}

func TestREPLExecutorCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/executor codex\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Executor set to: codex")
}

func TestREPLSynthesizerCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/synthesizer grok\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Synthesizer set to: grok")
}

func TestREPLSynthesizerClear(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/synthesizer none\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Synthesizer cleared")
}

func TestREPLUsageNoRuns(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/usage\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "No runs yet")
}

func TestREPLConfigCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/config\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Current Configuration")
	assert.Contains(t, output, "Responders:")
	assert.Contains(t, output, "Executor:")
}

func TestREPLModelsCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/models\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "MODEL")
	assert.Contains(t, output, "agy")
	assert.Contains(t, output, "grok")
}

func TestREPLDetectCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/detect\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "CLI")
	assert.Contains(t, output, "INSTALLED")
}

func TestREPLUnknownCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/bogus\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Unknown command")
}

func TestREPLQuitAlias(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /quit should also exit.
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/quit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "REPL")
}

func TestREPLRunPromptThenUsage(t *testing.T) {
	configPath := writeEchoConfig(t)

	// Run a prompt, then check usage, then exit.
	output, err := runREPLWithInput(t,
		[]string{"ask", "--config", configPath},
		"test task\n/usage\n/exit\n")

	require.NoError(t, err)
	// The prompt should have run and produced output.
	assert.Contains(t, output, "=== Convene Result ===")
	// Usage should show 1 run.
	assert.Contains(t, output, "Total runs:")
	assert.Contains(t, output, "1")
}

func TestREPLSessionSummaryOnExit(t *testing.T) {
	configPath := writeEchoConfig(t)

	// Run a prompt, then exit — should see session summary.
	output, err := runREPLWithInput(t,
		[]string{"ask", "--config", configPath},
		"test task\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Session Summary")
	assert.Contains(t, output, "Total runs:")
}

func TestREPLQuitShortcut(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /q should also exit.
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/q\n")

	require.NoError(t, err)
	assert.Contains(t, output, "REPL")
}

// ---------------------------------------------------------------------------
// REPL slash command alignment tests (Devin/Codex/agy/Grok conventions)
// ---------------------------------------------------------------------------

func TestREPLExecutorShowCurrent(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /executor with no args → show current executor.
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/executor\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Current executor:")
}

func TestREPLStatusCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/status\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Session Status")
	assert.Contains(t, output, "Mode:")
	assert.Contains(t, output, "Model (exec):")
	assert.Contains(t, output, "Responders:")
	assert.Contains(t, output, "Runs:")
}

func TestREPLNewCommand(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /new should clear session (like Devin/Codex /new).
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/new\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "session cleared")
}

func TestREPLCompactStub(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/compact\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "stub")
	assert.Contains(t, output, "compact")
}

func TestREPLResumeStub(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/resume\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "stub")
	assert.Contains(t, output, "resume")
}

func TestREPLContinueAlias(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /continue is an alias for /resume.
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/continue\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "stub")
}

func TestREPLUpdateStub(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/update\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "stub")
	assert.Contains(t, output, "update")
}

func TestREPLSettingsAlias(t *testing.T) {
	configPath := writeEchoConfig(t)

	// /settings is an alias for /config (like agy).
	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/settings\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "Current Configuration")
}

func TestREPLHelpListsNewCommands(t *testing.T) {
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--config", configPath}, "/help\n/exit\n")

	require.NoError(t, err)
	// Commands should appear in help (note: /model was removed, /executor replaces it).
	assert.Contains(t, output, "/status")
	assert.Contains(t, output, "/executor")
	assert.Contains(t, output, "/new")
	assert.Contains(t, output, "/compact")
	assert.Contains(t, output, "/settings")
	assert.Contains(t, output, "/resume")
	assert.Contains(t, output, "/update")
}

// ---------------------------------------------------------------------------
// CLI flag alignment tests (--model, --json)
// ---------------------------------------------------------------------------

func TestCLIModelFlagAlias(t *testing.T) {
	// --model / -m is an alias for --executor.
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--model", "codex",
			"--responders", "agy,grok",
			"--config", configPath,
			"test-model-flag",
		})
	})

	require.NoError(t, err)
	// The echo config should produce output (model flag = executor).
	assert.Contains(t, output, "=== Convene Result ===")
}

func TestCLIModelShortFlag(t *testing.T) {
	// -m is the short form of --model.
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"-m", "codex",
			"--responders", "agy,grok",
			"--config", configPath,
			"test-model-short",
		})
	})

	require.NoError(t, err)
	assert.Contains(t, output, "=== Convene Result ===")
}

func TestCLIJSONFlag(t *testing.T) {
	// --json outputs the result as JSON.
	configPath := writeEchoConfig(t)

	output, err := captureStdout(t, func() error {
		return execRootCmd([]string{
			"ask",
			"--json",
			"--responders", "agy,grok",
			"--config", configPath,
			"test-json",
		})
	})

	require.NoError(t, err)
	// JSON output should start with { (JSON object).
	trimmed := strings.TrimSpace(output)
	assert.True(t, strings.HasPrefix(trimmed, "{"),
		"JSON output should start with '{', got: %s", trimmed[:min(20, len(trimmed))])
}

func TestCLIModelFlagEntersREPL(t *testing.T) {
	// --model should also work when entering REPL (no task).
	configPath := writeEchoConfig(t)

	output, err := runREPLWithInput(t,
		[]string{"--model", "devin", "--config", configPath}, "/status\n/exit\n")

	require.NoError(t, err)
	assert.Contains(t, output, "devin")
}
