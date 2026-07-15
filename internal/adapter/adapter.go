// Package adapter provides the model CLI adapter layer for OpenConveneCLI.
//
// Each adapter wraps a specific coding-agent CLI (agy, codex, devin, grok,
// cursor, kimi, hermes, aider, opencode) behind a uniform Adapter interface.
// The ConveneEngine (internal/convene) calls Respond for read-only queries
// and Execute for agentic execution, without needing to know which CLI is
// underneath.
//
// Design notes:
//   - GetCommand returns a complete shell command string (including quotes),
//     not a split argument list. RunCommand executes it via shell (sh -c on
//     Unix, cmd /c on Windows) to handle each CLI's quoting conventions.
//   - RunCommand uses process groups (Setpgid on Unix, CREATE_NEW_PROCESS_GROUP
//     on Windows) so that timeout kills the entire process tree, preventing
//     orphaned grandchild processes from consuming CPU/API quota.
//   - SupportsReadOnly checks Config.ReadOnly == "true" directly (not via an
//     IsReadOnly method) so this package compiles independently of S5.
package adapter

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/config"
)

// DepthEnvVar is the environment variable used to guard against nested
// OpenConvene dispatch (omnilane-inspired no-nested-dispatch safety rail).
//
// When OpenConvene spawns an executor CLI (devin/codex/agy/...), it injects
// OPENCONVENE_DEPTH=<current+1> into the child environment. If that child
// (or any descendant) invokes openconvene again, the CLI checks this var at
// startup and refuses to run (exit 86), preventing runaway agent-calls-agent
// quota chains.
const DepthEnvVar = "OPENCONVENE_DEPTH"

// MaxDispatchDepth is the hard cap on nesting. Even depth 1 is refused (the
// top-level openconvene process has depth 0; any child that re-invokes
// openconvene sees depth >= 1).
const MaxDispatchDepth = 1

// AdapterResult is the return value of every adapter call.
//
// ConveneEngine extracts the plain-text response from Stdout.
// Stderr and ReturnCode are for diagnostics. Success is true only when
// ReturnCode == 0 AND Stdout is non-empty.
type AdapterResult struct {
	// Stdout is the CLI's standard output (plain-text response).
	Stdout string

	// Stderr is the CLI's standard error (diagnostics).
	Stderr string

	// ReturnCode is the CLI's exit code (0 = success).
	// Set to -1 when the process could not be started or was killed by signal.
	ReturnCode int

	// Success is true when ReturnCode == 0 and Stdout is non-empty.
	Success bool
}

// Adapter is the interface every model CLI adapter must implement.
//
// Respond is used for read-only queries (responders and synthesizers).
// Execute is used for agentic execution (executors).
// SupportsReadOnly indicates whether the CLI truly enforces read-only mode.
// GetCommand assembles the full shell command string for the given mode.
type Adapter interface {
	// Respond calls the CLI in read-only mode and returns the plain-text
	// response. Used for responders and synthesizers.
	Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error)

	// Execute calls the CLI in agentic mode, which may use tools (write files,
	// run commands). synthesisContext is the synthesizer's merged conclusion
	// (may be empty if no synthesizer was used).
	Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error)

	// SupportsReadOnly returns true only when the CLI enforces read-only mode
	// (Config.ReadOnly == "true"). "maybe" and "false" both return false.
	SupportsReadOnly() bool

	// GetCommand assembles the full shell command string (including quotes).
	// mode = "respond" uses Config.Command;
	// mode = "execute" uses Config.ExecuteCommand (falls back to Command if empty).
	// The {prompt} placeholder is replaced with the shell-escaped prompt.
	GetCommand(prompt string, mode string) string
}

// BaseAdapter provides shared logic for all adapters.
//
// Each concrete adapter embeds BaseAdapter to inherit GetCommand, Respond,
// Execute, and SupportsReadOnly. Adapters that need different behavior
// (e.g. Aider does not support Respond) override the relevant methods.
type BaseAdapter struct {
	// Name is the adapter identifier (e.g. "agy", "codex").
	// Populated by the factory from the config map key.
	Name string

	// Config is the strongly-typed model configuration.
	// Adapters access Config.Command, Config.ExecuteCommand, Config.ReadOnly,
	// Config.Timeout directly.
	Config config.ModelConfig
}

// GetCommand assembles the full shell command string for the given mode.
//
// mode = "respond" uses Config.Command (read-only template).
// mode = "execute" uses Config.ExecuteCommand (agentic template); if
// ExecuteCommand is empty, it falls back to Command.
//
// The {prompt} placeholder in the template is replaced with the shell-escaped
// prompt via ReplacePrompt.
//
// The {prompt_file} placeholder in the template is replaced with the path to
// a temporary file containing the prompt. This is useful for CLIs that support
// --prompt-file (e.g. devin) and for prompts that are too long or contain
// newlines/special characters that break shell quoting. The temp file is
// created by writePromptFile and cleaned up by the caller via cleanup.
func (b *BaseAdapter) GetCommand(prompt string, mode string) string {
	tmpl := b.Config.Command
	if mode == "execute" && b.Config.ExecuteCommand != "" {
		tmpl = b.Config.ExecuteCommand
	}

	// If the template uses {prompt_file}, write the prompt to a temp file
	// and substitute the file path.
	if strings.Contains(tmpl, "{prompt_file}") {
		path, err := writePromptFile(prompt)
		if err != nil {
			// Fall back to inline {prompt} substitution if file write fails.
			return ReplacePrompt(strings.ReplaceAll(tmpl, "{prompt_file}", "{prompt}"), prompt)
		}
		return strings.ReplaceAll(tmpl, "{prompt_file}", path)
	}

	return ReplacePrompt(tmpl, prompt)
}

// Respond calls the CLI in read-only mode using the configured Command template.
//
// This is the standard implementation shared by most adapters.
// Aider overrides this to return an error (it does not support respond mode).
func (b *BaseAdapter) Respond(ctx context.Context, prompt string, timeout int) (AdapterResult, error) {
	cmd := b.GetCommand(prompt, "respond")
	result, err := RunCommand(ctx, cmd, timeout)
	cleanupPromptFile(cmd)
	return result, err
}

// Execute calls the CLI in agentic mode using the configured ExecuteCommand
// template (or Command if ExecuteCommand is empty).
//
// synthesisContext is the synthesizer's merged conclusion. It is appended to
// the prompt by the caller (ConveneEngine) before calling Execute, so the
// adapter does not need to handle it separately — the prompt already contains
// the synthesis context.
func (b *BaseAdapter) Execute(ctx context.Context, prompt string, timeout int, synthesisContext string) (AdapterResult, error) {
	cmd := b.GetCommand(prompt, "execute")
	result, err := RunCommand(ctx, cmd, timeout)
	cleanupPromptFile(cmd)
	return result, err
}

// SupportsReadOnly returns true only when Config.ReadOnly == "true".
//
// "maybe" (has non-interactive mode but inherently agentic) and "false"
// (modifies files by default) both return false.
//
// This checks Config.ReadOnly directly rather than calling an IsReadOnly()
// method, so this package compiles independently of S5 (config.go).
func (b *BaseAdapter) SupportsReadOnly() bool {
	return b.Config.ReadOnly == "true"
}

// RunCommand executes a full shell command string with timeout and returns
// an AdapterResult.
//
// The command is run via shell (sh -c on Unix, cmd /c on Windows) because
// the command string contains quoted prompts that cannot be safely split
// into arguments with strings.Fields.
//
// Process group handling:
//   - Unix: Setpgid creates a new process group. On timeout, the entire
//     group is killed with syscall.Kill(-pid, SIGKILL) to prevent orphaned
//     grandchild processes (the CLI is a grandchild of sh).
//   - Windows: CREATE_NEW_PROCESS_GROUP (0x00000200) creates a new process
//     group. On timeout, taskkill /T /F /PID kills the process tree.
//
// stdin is explicitly set to nil to prevent CLIs from blocking on stdin
// reads (Go defaults to /dev/null, but being explicit avoids edge cases).
//
// This function never panics. All errors are captured in AdapterResult or
// returned as error.
func RunCommand(ctx context.Context, cmdStr string, timeout int) (AdapterResult, error) {
	if cmdStr == "" {
		return AdapterResult{
			Stdout:     "",
			Stderr:     "empty command string",
			ReturnCode: -1,
			Success:    false,
		}, fmt.Errorf("empty command string")
	}

	// Set up timeout context.
	// If timeout <= 0, use a cancellable context without deadline (no timeout).
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Build the shell command with platform-specific process group isolation.
	// setProcessGroupAttr (defined in process_unix.go / process_windows.go)
	// configures the SysprocAttr for the target platform:
	//   - Unix:    Setpgid creates a new process group
	//   - Windows: CREATE_NEW_PROCESS_GROUP isolates the child group
	//
	// On Windows, cmd.exe's /c switch mangles quotes (Go exec escapes " as \",
	// which cmd.exe doesn't understand), causing arguments with spaces to be
	// split incorrectly. To work around this, we use two strategies:
	//   1. If the command contains shell features (|, >, <, &&, ||) or cmd
	//      builtins (echo, type, dir, set, etc.), use cmd /c (required for
	//      pipes and builtins).
	//   2. Otherwise, parse the command into args and use exec.Command directly,
	//      which lets Go handle quoting correctly via Windows API. This is
	//      essential for CLIs like agy that receive prompts as quoted arguments.
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		if needsShell(cmdStr) {
			cmd = exec.CommandContext(ctx, "cmd", "/c", cmdStr)
		} else {
			parts, err := parseArgs(cmdStr)
			if err != nil {
				return AdapterResult{
					Stdout:     "",
					Stderr:     fmt.Sprintf("failed to parse command args: %s", err),
					ReturnCode: -1,
					Success:    false,
				}, fmt.Errorf("failed to parse command args: %w", err)
			}
			if len(parts) == 0 {
				return AdapterResult{
					Stdout:     "",
					Stderr:     "empty command after parsing",
					ReturnCode: -1,
					Success:    false,
				}, fmt.Errorf("empty command after parsing")
			}
			cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		}
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
	}
	setProcessGroupAttr(cmd)

	// Inject OPENCONVENE_DEPTH=<current+1> into the child environment so that
	// any descendant that re-invokes openconvene is refused (no-nested-dispatch
	// guard). If the var is absent (top-level process), the child sees depth 1.
	cmd.Env = childEnvWithBumpedDepth()

	// Set stdin to the parent's stdin. This allows pipe-based command templates
	// (e.g. "type {prompt_file} | codex exec") to work — the shell (cmd/sh)
	// handles the pipe internally, but still needs access to stdin if a
	// command in the pipeline reads from it.
	// Note: most CLIs that don't read stdin will simply ignore it.
	cmd.Stdin = os.Stdin

	// Capture stdout and stderr.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the process.
	if err := cmd.Start(); err != nil {
		return AdapterResult{
			Stdout:     "",
			Stderr:     fmt.Sprintf("failed to start command: %s", err),
			ReturnCode: -1,
			Success:    false,
		}, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for the process to complete (or be killed by context cancellation).
	err := cmd.Wait()

	// If the context was cancelled or timed out, kill the entire process group
	// to prevent orphaned grandchild processes from continuing to run.
	//
	// On Unix, even after Go kills the direct child (sh) via context cancel,
	// the process group persists while grandchildren live. syscall.Kill(-pid,
	// SIGKILL) kills every process in the group.
	//
	// On Windows, taskkill /T /F /PID kills the process tree. This is
	// best-effort — if the parent already exited, children may not be found.
	if ctx.Err() != nil && cmd.Process != nil {
		killProcessGroup(cmd.Process.Pid)
	}

	// Determine return code.
	returnCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			returnCode = exitErr.ExitCode()
		} else {
			returnCode = -1
		}
	}

	// Handle timeout specifically.
	if ctx.Err() == context.DeadlineExceeded {
		return AdapterResult{
			Stdout:     stdout.String(),
			Stderr:     fmt.Sprintf("timeout after %ds", timeout),
			ReturnCode: returnCode,
			Success:    false,
		}, fmt.Errorf("timeout after %ds", timeout)
	}

	// Handle context cancellation (e.g. user pressed Ctrl+C).
	if ctx.Err() == context.Canceled {
		return AdapterResult{
			Stdout:     stdout.String(),
			Stderr:     "cancelled",
			ReturnCode: returnCode,
			Success:    false,
		}, ctx.Err()
	}

	// Normal completion.
	success := returnCode == 0 && stdout.Len() > 0
	return AdapterResult{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		ReturnCode: returnCode,
		Success:    success,
	}, err
}

// killProcessGroup is defined in process_unix.go and process_windows.go
// with platform-specific implementations.

// writePromptFile writes the prompt to a temporary file and returns its path.
// The file is prefixed with "openconvene-prompt-" for easy identification.
// The caller is responsible for cleaning up the file via cleanupPromptFile.
func writePromptFile(prompt string) (string, error) {
	tmpFile, err := os.CreateTemp("", "openconvene-prompt-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp prompt file: %w", err)
	}
	path := tmpFile.Name()
	if _, err := tmpFile.WriteString(prompt); err != nil {
		tmpFile.Close()
		os.Remove(path)
		return "", fmt.Errorf("failed to write prompt to temp file: %w", err)
	}
	tmpFile.Close()
	return path, nil
}

// cleanupPromptFile extracts the temp file path from a command string (if it
// used {prompt_file}) and removes the file. This is a best-effort cleanup —
// if extraction fails, the file may remain in the temp directory (OS will
// clean it up eventually).
func cleanupPromptFile(cmdStr string) {
	// Look for the temp file pattern in the command string.
	// The path was substituted directly into the command.
	idx := strings.Index(cmdStr, "openconvene-prompt-")
	if idx < 0 {
		return
	}
	// Extract the path — it's the substring from idx to the next space or quote.
	rest := cmdStr[idx:]
	end := len(rest)
	for i, ch := range rest {
		if ch == ' ' || ch == '"' || ch == '\'' {
			end = i
			break
		}
	}
	path := rest[:end]
	if path != "" {
		os.Remove(path)
	}
}

// needsShell returns true if the command string contains shell features that
// require a shell (cmd.exe on Windows, sh on Unix) to execute.
//
// Shell features detected: pipes (|), redirection (>, <), command chaining
// (&&, ||), and common Windows cmd.exe builtins (type, dir, set, copy, del).
func needsShell(cmdStr string) bool {
	// Check for pipe or redirection operators.
	if strings.ContainsAny(cmdStr, "|<>") {
		return true
	}
	// Check for command chaining.
	if strings.Contains(cmdStr, "&&") || strings.Contains(cmdStr, "||") {
		return true
	}
	// Check for common Windows cmd.exe builtins.
	// These must be the first token (command name).
	trimmed := strings.TrimSpace(cmdStr)
	for _, builtin := range []string{"echo ", "type ", "dir ", "set ", "copy ", "del ", "ren ", "md ", "rd ", "cd ", "call ", "start ", "for ", "if ", "rem "} {
		if strings.HasPrefix(strings.ToLower(trimmed), builtin) {
			return true
		}
	}
	return false
}

// parseArgs splits a command string into arguments, respecting double quotes.
//
// This is a simple parser for Windows command lines. It handles:
//   - Double-quoted strings (quotes are removed from the output)
//   - Backslash-escaped double quotes inside double-quoted strings (\" → ")
//   - Spaces as argument separators (outside quotes)
//
// Example: `agy --print "hello world"` → ["agy", "--print", "hello world"]
func parseArgs(cmdStr string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuotes := false
	i := 0

	for i < len(cmdStr) {
		ch := cmdStr[i]
		switch ch {
		case '"':
			if inQuotes && i+1 < len(cmdStr) && cmdStr[i+1] == '"' {
				// Escaped quote inside quoted string: "" → "
				current.WriteByte('"')
				i += 2
				continue
			}
			inQuotes = !inQuotes
			i++
		case '\\':
			if inQuotes && i+1 < len(cmdStr) && cmdStr[i+1] == '"' {
				// Escaped quote: \" → "
				current.WriteByte('"')
				i += 2
				continue
			}
			current.WriteByte(ch)
			i++
		case ' ', '\t':
			if inQuotes {
				current.WriteByte(ch)
				i++
			} else {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
				// Skip consecutive whitespace.
				for i < len(cmdStr) && (cmdStr[i] == ' ' || cmdStr[i] == '\t') {
					i++
				}
			}
		default:
			current.WriteByte(ch)
			i++
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unterminated quoted string in command")
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args, nil
}

// ReplacePrompt replaces the {prompt} placeholder in a command template with
// the shell-escaped prompt.
//
// The prompt is shell-escaped before insertion to prevent command injection.
// Escaping rules differ by platform (see shellEscape).
func ReplacePrompt(template, prompt string) string {
	return strings.ReplaceAll(template, "{prompt}", shellEscape(prompt))
}

// shellEscape escapes a string for safe inclusion inside double quotes in a
// shell command.
//
// On Unix (sh): escapes backslash, double quote, dollar sign, and backtick
// to prevent command substitution and variable expansion.
//
// On Windows (cmd.exe): escapes double quote and percent sign. cmd.exe
// inside double quotes still expands %VAR%, so % is escaped as %%.
//
// Known limitation: Windows cmd.exe percent-escaping (%%) is designed for
// batch files, not interactive commands. In practice, prompts from
// OpenConveneCLI rarely contain % characters. This is a known cross-platform
// edge case noted for future hardening.
func shellEscape(s string) string {
	if runtime.GOOS == "windows" {
		// Windows cmd.exe: inside double quotes, escape " and %.
		s = strings.ReplaceAll(s, `"`, `\"`)
		s = strings.ReplaceAll(s, `%`, `%%`)
		return s
	}
	// Unix sh: inside double quotes, escape \, ", $, and `.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `$`, `\$`)
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// childEnvWithBumpedDepth returns the current process environment with
// OPENCONVENE_DEPTH incremented by 1. This is injected into every spawned
// CLI subprocess so that descendants cannot re-invoke openconvene (the
// no-nested-dispatch guard in main.go refuses to run when depth > 0).
func childEnvWithBumpedDepth() []string {
	current := 0
	if v := os.Getenv(DepthEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			current = n
		}
	}
	childDepth := current + 1

	env := os.Environ()
	found := false
	prefixed := DepthEnvVar + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefixed) {
			env[i] = prefixed + strconv.Itoa(childDepth)
			found = true
			break
		}
	}
	if !found {
		env = append(env, prefixed+strconv.Itoa(childDepth))
	}
	return env
}

// CurrentDispatchDepth returns the current OPENCONVENE_DEPTH value (0 at the
// top-level openconvene process, >= 1 inside a spawned CLI's child that
// re-invokes openconvene).
func CurrentDispatchDepth() int {
	if v := os.Getenv(DepthEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 0
}
