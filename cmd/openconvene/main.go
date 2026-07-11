// Command openconvene is the CLI entry point for OpenConveneCLI.
//
// OpenConveneCLI is a Mixture-of-Agents (MoA) collaboration tool that
// orchestrates multiple AI coding-CLI adapters (agy, grok, codex, devin,
// cursor, kimi, hermes, aider, opencode) through a 3-phase pipeline:
//
//	Phase 1 — Fan-out: N responder models run in parallel (read-only).
//	Phase 2 — Synthesis (optional): a synthesizer integrates responses.
//	Phase 3 — Execution (mode-dependent): an executor acts on the result.
//
// Three modes:
//   - ask (research): responders + optional synthesis, NO execution.
//   - code (default): responders + optional synthesis + executor (code implementation).
//   - agent:          responders + optional synthesis + executor (broad agentic actions).
//
// Usage (aligned with industry conventions — codex/grok/agy use positional args):
//
//	openconvene "fix the bug in foo.go"        # default = code mode
//	openconvene ask "what is CRDT?"            # ask mode (research, no execution)
//	openconvene agent "deploy the app"         # agent mode (broad agentic actions)
//
//	openconvene models                         # list configured models
//	openconvene detect                         # detect which CLIs are installed
//	openconvene init                           # generate a starter models.yaml
//	openconvene check                          # validate models.yaml
//
// Config resolution priority:
//  1. --config flag
//  2. OPENCONVENE_CLI_CONFIG environment variable
//  3. Default search paths (~/.config/openconvene/models.yaml, ./config/models.yaml)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/masteryee-labs/open-convene-cli/internal/convene"
	"github.com/masteryee-labs/open-convene-cli/internal/mode"
)

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	rootCmd := buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		// cobra prints the error to stderr automatically; just exit non-zero.
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Root command — default code mode, accepts positional task arg
// ---------------------------------------------------------------------------

func buildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "openconvene [task]",
		Short: "Mixture-of-Agents CLI that orchestrates multiple AI coding agents",
		Long: `OpenConveneCLI is a Mixture-of-Agents (MoA) collaboration tool that
orchestrates multiple AI coding-CLI adapters through a 3-phase pipeline:
fan-out responders, optional synthesis, and mode-dependent execution.

Supported adapters: agy, grok, codex, devin, cursor, kimi, hermes, aider, opencode.

Modes:
  ask   Responders + optional synthesis (no execution). For analysis & Q&A.
  code  Responders + optional synthesis + executor. For coding tasks. (default)
  agent Responders + optional synthesis + executor. For multi-step tasks.

Quick start:
  openconvene "fix the bug in foo.go"        # default = code mode
  openconvene ask "what is CRDT?"            # ask mode (research, no execution)
  openconvene agent "deploy the app"         # agent mode (broad agentic actions)

  openconvene detect                         # see which CLIs are installed
  openconvene init                           # generate a starter models.yaml
  openconvene models                         # list configured models
  openconvene check                          # validate models.yaml

If your task starts with a subcommand name (e.g. "ask"), use -- to disambiguate:
  openconvene -- ask`,
		RunE: runRoot,
		Args: cobra.ArbitraryArgs,
	}

	addConveneFlags(rootCmd)

	// Subcommands
	rootCmd.AddCommand(buildAskCmd())
	rootCmd.AddCommand(buildAgentCmd())
	rootCmd.AddCommand(buildModelsCmd())
	rootCmd.AddCommand(buildModelsInfoCmd())
	rootCmd.AddCommand(buildDetectCmd())
	rootCmd.AddCommand(buildInitCmd())
	rootCmd.AddCommand(buildCheckCmd())

	// Hidden aliases for backward compatibility
	rootCmd.AddCommand(buildRunCmdHidden())
	rootCmd.AddCommand(buildListModelsCmdHidden())
	rootCmd.AddCommand(buildConfigCmdHidden())

	return rootCmd
}

// addConveneFlags adds flags shared by root (code), ask, and agent commands.
func addConveneFlags(cmd *cobra.Command) {
	cmd.Flags().String("responders", "", "Comma-separated responder model names (overrides config defaults)")
	cmd.Flags().String("executor", "", "Executor model name (overrides config defaults; required for code/agent)")
	cmd.Flags().String("synthesizer", "", "Synthesizer model name (overrides config defaults; empty = executor doubles as synthesizer)")
	cmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	cmd.Flags().Int("timeout", 0, "Override per-call timeout in seconds (0 = use config default)")
	cmd.Flags().Bool("verbose", false, "Show raw responder responses and metadata")
	cmd.Flags().BoolP("print", "p", false, "Non-interactive mode (default: always non-interactive)")
	// --model / -m: alias for --executor (aligned with Devin/Codex/agy/Grok --model).
	cmd.Flags().StringP("model", "m", "", "Model name (alias for --executor; overrides config default)")
	// --json: output result as JSON (aligned with Grok --output-format json).
	cmd.Flags().Bool("json", false, "Output result as JSON (for scripting/automation)")
	// Hidden --task flag for backward compatibility with the old `run` command.
	// New usage prefers positional args: openconvene "task"
	cmd.Flags().String("task", "", "Task description (positional arg preferred; use '-' for stdin)")
	cmd.Flags().MarkHidden("task")
}

// runRoot handles `openconvene "task"` (default = code mode).
// If no task is provided, enters interactive REPL mode.
func runRoot(cmd *cobra.Command, args []string) error {
	task := resolveTask(cmd, args)
	if task == "" {
		return enterREPL(cmd, "code")
	}
	return runConvene(cmd, args, "code")
}

// ---------------------------------------------------------------------------
// ask command — research mode (no execution)
// ---------------------------------------------------------------------------

func buildAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask [task]",
		Short: "Ask a question — research mode (no execution, read-only)",
		Long: `Ask a question using multiple AI agents in parallel (research mode).

N responder models answer your question in parallel (read-only). An optional
synthesizer integrates their responses. No executor runs — no files are
modified, no commands are executed. Ideal for analysis, Q&A, and research.

The task is a positional argument:
  openconvene ask "what is CRDT?"
  openconvene ask "compare gRPC vs REST"

Use -- to disambiguate if needed:
  openconvene ask -- "explain this code"

Config resolution: --config flag > OPENCONVENE_CLI_CONFIG env > default paths.
Model overrides: CLI flags take priority over config defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			task := resolveTask(cmd, args)
			if task == "" {
				return enterREPL(cmd, "research")
			}
			return runConvene(cmd, args, "research")
		},
		Args: cobra.ArbitraryArgs,
	}

	addConveneFlags(cmd)
	return cmd
}

// ---------------------------------------------------------------------------
// agent command — agent mode (broad agentic actions with execution)
// ---------------------------------------------------------------------------

func buildAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [task]",
		Short: "Run an agentic task — agent mode (with execution)",
		Long: `Run a broad agentic task using multiple AI agents (agent mode).

N responder models answer your task in parallel (read-only). An optional
synthesizer integrates their responses. Then an executor acts on the result —
it may write files, run commands, and perform multi-step actions.

The task is a positional argument:
  openconvene agent "deploy the app to staging"
  openconvene agent "set up CI/CD pipeline"

If no task is provided, enters interactive REPL mode:
  openconvene agent

Use -- to disambiguate if needed:
  openconvene agent -- "agent something"

Config resolution: --config flag > OPENCONVENE_CLI_CONFIG env > default paths.
Model overrides: CLI flags take priority over config defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			task := resolveTask(cmd, args)
			if task == "" {
				return enterREPL(cmd, "agent")
			}
			return runConvene(cmd, args, "agent")
		},
		Args: cobra.ArbitraryArgs,
	}

	addConveneFlags(cmd)
	return cmd
}

// ---------------------------------------------------------------------------
// resolveTask — get task from positional arg or hidden --task flag
// ---------------------------------------------------------------------------

// resolveTask extracts the task from positional args (priority) or the
// hidden --task flag (fallback for backward compatibility with `run`).
// If the task is "-", it reads from stdin.
func resolveTask(cmd *cobra.Command, args []string) string {
	// Positional arg takes priority.
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	// Fall back to hidden --task flag.
	task, _ := cmd.Flags().GetString("task")
	return task
}

// enterREPL loads the config and starts the interactive REPL.
// Called when `openconvene`, `openconvene ask`, or `openconvene agent` is
// run without a task argument.
//
// If no config file is found, a default config is auto-generated using
// dynamic model names (CLI:模型名 format) so the user can start immediately
// without running `openconvene init` first.
func enterREPL(cmd *cobra.Command, initialMode string) error {
	// Resolve config path: --config > env > default.
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = os.Getenv("OPENCONVENE_CLI_CONFIG")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// No config found — auto-generate a default one with dynamic models.
		// This lets the user start using the REPL immediately.
		fmt.Fprintf(os.Stderr, "No models.yaml found. Auto-generating default config with dynamic models...\n\n")
		cfg, configPath, err = autoGenerateConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to auto-generate config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Config written to: %s\n\n", configPath)
	}

	// Apply CLI flag overrides for initial responders/executor/synthesizer.
	flagResponders, _ := cmd.Flags().GetString("responders")
	if flagResponders != "" {
		cfg.Defaults.Responders = parseCSV(flagResponders)
	}
	flagExecutor, _ := cmd.Flags().GetString("executor")
	if flagExecutor == "" {
		// --model / -m is an alias for --executor.
		flagExecutor, _ = cmd.Flags().GetString("model")
	}
	if flagExecutor != "" {
		cfg.Defaults.Executor = flagExecutor
	}
	flagSynth, _ := cmd.Flags().GetString("synthesizer")
	if flagSynth != "" {
		cfg.Defaults.Synthesizer = &flagSynth
	}
	flagTimeout, _ := cmd.Flags().GetInt("timeout")
	if flagTimeout > 0 {
		cfg.Defaults.Timeout = flagTimeout
	}

	return runREPL(initialMode, cfg, configPath)
}

// autoGenerateConfig creates a default models.yaml with dynamic model names
// and returns the loaded config + path. This is called when no config exists
// and the user tries to enter the REPL.
func autoGenerateConfig(explicitPath string) (*config.ConveneConfig, string, error) {
	// Determine where to write the config.
	path := explicitPath
	if path == "" {
		// Use the default search path: ~/.config/openconvene/models.yaml
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", err
		}
		path = filepath.Join(home, ".config", "openconvene", "models.yaml")
	}

	// Create the directory if needed.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, "", err
	}

	// Write a minimal config with dynamic model names.
	defaultContent := `# OpenConveneCLI — auto-generated default config
# Uses dynamic model names (CLI:模型名 format). No models section needed.
# Edit responders/executor/synthesizer to customize your MoA setup.

defaults:
  timeout: 120
  responders:
    - devin:glm-5.2
    - devin:swe-1.7
    - devin:kimi-k2.7
  executor: devin:glm-5.2
  synthesizer: devin:glm-5.2
`
	if err := os.WriteFile(path, []byte(defaultContent), 0644); err != nil {
		return nil, "", err
	}

	// Load and return the config we just wrote.
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, "", err
	}
	return cfg, path, nil
}

// parseCSV splits a comma-separated string into a trimmed slice.
func parseCSV(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// runConvene — shared implementation for root (code), ask (research), agent
// ---------------------------------------------------------------------------

func runConvene(cmd *cobra.Command, args []string, modeStr string) error {
	// --- 0. Resolve config path: --config > env > default ---
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = os.Getenv("OPENCONVENE_CLI_CONFIG")
	}

	// --- 1. Load config (auto-generate if missing) ---
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		// Auto-generate a default config so the user can start immediately.
		fmt.Fprintf(os.Stderr, "No models.yaml found. Auto-generating default config...\n\n")
		cfg, configPath, err = autoGenerateConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to auto-generate config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Config written to: %s\n\n", configPath)
	}

	// --- 2. Merge CLI params with config defaults (CLI takes priority) ---

	// 2a. Responders: flag (comma-separated) > config defaults.
	flagResponders, _ := cmd.Flags().GetString("responders")
	var responders []string
	if flagResponders != "" {
		for _, r := range strings.Split(flagResponders, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				responders = append(responders, r)
			}
		}
	} else {
		responders = cfg.Defaults.Responders
	}
	if len(responders) == 0 {
		return fmt.Errorf("no responders specified (use --responders or set defaults.responders in config)")
	}

	// 2b. Executor: --executor flag > --model flag (alias) > config defaults.
	flagExecutor, _ := cmd.Flags().GetString("executor")
	executor := flagExecutor
	if executor == "" {
		// --model / -m is an alias for --executor (aligned with other CLIs).
		flagModel, _ := cmd.Flags().GetString("model")
		executor = flagModel
	}
	if executor == "" {
		executor = cfg.Defaults.Executor
	}

	// 2c. Synthesizer: flag (non-empty → pointer) > config defaults (already *string, may be nil).
	flagSynth, _ := cmd.Flags().GetString("synthesizer")
	var synthesizer *string
	if flagSynth != "" {
		synthesizer = &flagSynth
	} else {
		synthesizer = cfg.Defaults.Synthesizer
	}

	// 2d. Timeout override: flag > 0 overrides config defaults.
	flagTimeout, _ := cmd.Flags().GetInt("timeout")
	if flagTimeout > 0 {
		cfg.Defaults.Timeout = flagTimeout
	}

	// --- 3. Resolve task (positional arg, --task flag, or stdin) ---
	task := resolveTask(cmd, args)
	if task == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read task from stdin: %w", err)
		}
		task = strings.TrimSpace(string(data))
	}
	if task == "" {
		return fmt.Errorf("task is empty")
	}

	// --- 4. Mode validation ---
	m := mode.Mode(modeStr)
	switch m {
	case mode.ModeResearch, mode.ModeCode, mode.ModeAgent:
		// valid
	default:
		return fmt.Errorf("invalid mode %q: must be one of research, code, agent", modeStr)
	}

	// Validate mode + model combination.
	validationErrors, validationWarnings := mode.ValidateModeConfig(
		m, responders, executor, synthesizer, cfg.Models)

	if len(validationErrors) > 0 {
		for _, e := range validationErrors {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
		}
		return fmt.Errorf("mode validation failed with %d error(s)", len(validationErrors))
	}

	for _, w := range validationWarnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

	// --- 5. Execute Convene flow ---
	ctx := context.Background()
	engine := convene.NewConveneEngine(cfg)

	result, err := engine.Run(ctx, task, modeStr, responders, executor, synthesizer)
	if err != nil {
		// Even on error, print verbose metadata if requested (helps debug failures).
		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose && len(result.Metadata) > 0 {
			fmt.Fprintln(os.Stderr, "\n=== VERBOSE OUTPUT (on failure) ===")
			fmt.Fprintln(os.Stderr, "--- Metadata ---")
			keys := make([]string, 0, len(result.Metadata))
			for k := range result.Metadata {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(os.Stderr, "[metadata] %s: %v\n", k, result.Metadata[k])
			}
		}
		return fmt.Errorf("convene run failed: %w", err)
	}

	// --- 6. Format and print output ---
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// JSON output mode (aligned with Grok --output-format json).
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		output := mode.FormatOutput(result, m)
		fmt.Print(output)
	}

	// --- 7. Verbose: raw responses + metadata to stderr ---
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		fmt.Fprintln(os.Stderr, "\n=== VERBOSE OUTPUT ===")

		if len(result.Responses) > 0 {
			fmt.Fprintln(os.Stderr, "--- Raw Responder Responses ---")
			names := make([]string, 0, len(result.Responses))
			for name := range result.Responses {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				fmt.Fprintf(os.Stderr, "\n--- %s ---\n%s\n", name, result.Responses[name])
			}
		}

		if len(result.Metadata) > 0 {
			fmt.Fprintln(os.Stderr, "\n--- Metadata ---")
			// Sort metadata keys for deterministic output.
			keys := make([]string, 0, len(result.Metadata))
			for k := range result.Metadata {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(os.Stderr, "[metadata] %s: %v\n", k, result.Metadata[k])
			}
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// models command — replaces list-models
// ---------------------------------------------------------------------------

func buildModelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List configured models with read-only capability and installation status",
		Long: `List all models defined in the config (models.yaml) along with their
read-only capability, executor capability, and installation status.

The installation status is detected via exec.LookPath (scans PATH).

Config resolution: --config flag > OPENCONVENE_CLI_CONFIG env > default paths.`,
		RunE: runListModels,
	}

	cmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	return cmd
}

func runListModels(cmd *cobra.Command, args []string) error {
	// Resolve config path: --config > env > default.
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = os.Getenv("OPENCONVENE_CLI_CONFIG")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Models) == 0 {
		fmt.Println("No models configured.")
		return nil
	}

	// Sort model names for deterministic output.
	names := make([]string, 0, len(cfg.Models))
	for name := range cfg.Models {
		names = append(names, name)
	}
	sort.Strings(names)

	// Detect installed CLIs for the installation status column.
	detected := adapter.DetectAvailableAdapters()
	installedMap := make(map[string]bool, len(detected))
	for _, d := range detected {
		installedMap[d.Name] = d.Found
	}

	// Print table header.
	fmt.Printf("%-12s %-40s %-10s %-16s %-10s\n",
		"MODEL", "COMMAND", "READ_ONLY", "EXECUTOR_CAPABLE", "INSTALLED")
	fmt.Printf("%-12s %-40s %-10s %-16s %-10s\n",
		"------------", "----------------------------------------", "----------", "----------------", "----------")

	for _, name := range names {
		m := cfg.Models[name]

		// Truncate command for display if too long.
		cmdDisplay := m.Command
		if len(cmdDisplay) > 40 {
			cmdDisplay = cmdDisplay[:37] + "..."
		}
		if cmdDisplay == "" {
			cmdDisplay = "(none)"
		}

		readOnly := m.ReadOnly
		if readOnly == "" {
			readOnly = "(unset)"
		}

		execCapable := "false"
		if m.ExecutorCapable {
			execCapable = "true"
		}

		installed := "no"
		if installedMap[name] {
			installed = "yes"
		}

		fmt.Printf("%-12s %-40s %-10s %-16s %-10s\n",
			name, cmdDisplay, readOnly, execCapable, installed)
	}

	fmt.Printf("\nTotal: %d model(s)\n", len(cfg.Models))
	return nil
}

// ---------------------------------------------------------------------------
// detect command
// ---------------------------------------------------------------------------

func buildDetectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect which of the 9 supported CLIs are installed on this system",
		Long: `Detect which of the 9 supported coding-agent CLIs are installed on
this system by scanning PATH (exec.LookPath).

Supported CLIs: devin, grok, codex, agy, cursor, kimi, hermes, aider, opencode.

For each CLI, this shows:
  - Installation status (found / not found)
  - Full path (if found)
  - Read-only capability (true / false / maybe)
  - Whether it can act as a responder or executor
  - Installation command (for CLIs not yet installed)

This command does NOT require a config file — it scans PATH directly.
It does NOT auto-install any CLI; install commands are for reference only.`,
		RunE: runDetect,
	}
	return cmd
}

func runDetect(cmd *cobra.Command, args []string) error {
	results := adapter.DetectAvailableAdapters()

	// Print table header.
	fmt.Printf("%-12s %-10s %-40s %-10s %-12s %-12s\n",
		"CLI", "INSTALLED", "PATH", "READ_ONLY", "CAN_RESPOND", "CAN_EXECUTE")
	fmt.Printf("%-12s %-10s %-40s %-10s %-12s %-12s\n",
		"------------", "----------", "----------------------------------------", "----------", "------------", "------------")

	var installedNames []string
	var missingNames []string

	for _, r := range results {
		installed := "no"
		if r.Found {
			installed = "yes"
		}

		path := r.Path
		if path == "" {
			path = "-"
		}
		if len(path) > 40 {
			path = "..." + path[len(path)-37:]
		}

		readOnly := r.ReadOnly
		if readOnly == "" {
			readOnly = "(unknown)"
		}

		canRespond := "no"
		if r.CanRespond {
			canRespond = "yes"
		}

		canExecute := "no"
		if r.CanExecute {
			canExecute = "yes"
		}

		fmt.Printf("%-12s %-10s %-40s %-10s %-12s %-12s\n",
			r.Name, installed, path, readOnly, canRespond, canExecute)

		if r.Found {
			installedNames = append(installedNames, r.Name)
		} else {
			missingNames = append(missingNames, r.Name)
		}
	}

	// Summary + recommendations.
	fmt.Println()
	fmt.Printf("Installed: %d / %d\n", len(installedNames), len(results))

	if len(installedNames) > 0 {
		fmt.Printf("Available responders: %s\n", strings.Join(filterResponders(results, true), ", "))
		fmt.Printf("Available executors:  %s\n", strings.Join(filterExecutors(results, true), ", "))
	}

	if len(missingNames) > 0 {
		fmt.Println("\nMissing (install to use):")
		for _, r := range results {
			if !r.Found {
				fmt.Printf("  %-12s  %s\n", r.Name, r.InstallCmd)
			}
		}
	}

	return nil
}

// filterResponders returns the names of detected CLIs that can respond,
// optionally filtered by installation status.
func filterResponders(results []adapter.DetectResult, installedOnly bool) []string {
	var names []string
	for _, r := range results {
		if r.CanRespond && (!installedOnly || r.Found) {
			names = append(names, r.Name)
		}
	}
	return names
}

// filterExecutors returns the names of detected CLIs that can execute,
// optionally filtered by installation status.
func filterExecutors(results []adapter.DetectResult, installedOnly bool) []string {
	var names []string
	for _, r := range results {
		if r.CanExecute && (!installedOnly || r.Found) {
			names = append(names, r.Name)
		}
	}
	return names
}

// ---------------------------------------------------------------------------
// init command — replaces config init
// ---------------------------------------------------------------------------

func buildInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter models.yaml with all 9 adapters pre-configured",
		Long: `Generate a ready-to-use models.yaml with all 9 supported adapters
(agy, codex, devin, grok, cursor, kimi, hermes, aider, opencode) pre-configured
with recommended command templates and default settings.

The file is written to the path specified by --path, or to the default
location (config/models.yaml) if not specified.

An existing file will NOT be overwritten — remove it first to regenerate.`,
		RunE: runInit,
	}

	cmd.Flags().String("path", "", "Output path for models.yaml (default: config/models.yaml)")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	path, _ := cmd.Flags().GetString("path")
	if path == "" {
		path = filepath.Join("config", "models.yaml")
	}

	if err := config.InitConfig(path); err != nil {
		return fmt.Errorf("config init failed: %w", err)
	}

	fmt.Printf("Config written to %s\n", path)
	fmt.Println("\nEdit the file to customize command templates, timeouts, and defaults.")
	fmt.Println("Then run 'openconvene check' to validate it.")
	return nil
}

// ---------------------------------------------------------------------------
// check command — replaces config validate
// ---------------------------------------------------------------------------

func buildCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate models.yaml for structural and referential integrity",
		Long: `Validate an existing models.yaml for:
  - Correct YAML syntax
  - {prompt} placeholder presence in command templates
  - read_only values are valid (true / false / maybe)
  - timeouts are non-negative
  - defaults.executor exists and is executor_capable
  - defaults.responders reference existing models
  - defaults.synthesizer (if set) references an existing model

Issues are reported as ERROR (blocks loading) or WARNING (advisory).

Config resolution: --config flag > OPENCONVENE_CLI_CONFIG env > default paths.`,
		RunE: runCheck,
	}

	cmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	return cmd
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Resolve config path: --config > env > default.
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = os.Getenv("OPENCONVENE_CLI_CONFIG")
	}

	// LoadConfig runs ValidateConfig internally and returns an error if any
	// ERROR-severity issue is found.
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config validation FAILED:\n%v\n", err)
		return fmt.Errorf("config validation failed")
	}

	// LoadConfig succeeded (no ERROR-level issues). Run ValidateConfig again
	// to surface any WARNING-only issues that did not block loading.
	issues := config.ValidateConfig(cfg)
	var warnings []string
	for _, issue := range issues {
		if strings.HasPrefix(issue, "WARNING:") {
			warnings = append(warnings, issue)
		}
	}

	if len(warnings) > 0 {
		fmt.Printf("Config is valid, with %d warning(s):\n", len(warnings))
		for _, w := range warnings {
			fmt.Printf("  %s\n", w)
		}
	} else {
		fmt.Println("Config is valid")
	}

	// Print model count summary.
	fmt.Printf("\nModels configured: %d\n", len(cfg.Models))
	fmt.Printf("Default responders: %s\n", strings.Join(cfg.Defaults.Responders, ", "))
	fmt.Printf("Default executor: %s\n", cfg.Defaults.Executor)
	if cfg.Defaults.Synthesizer != nil && *cfg.Defaults.Synthesizer != "" {
		fmt.Printf("Default synthesizer: %s\n", *cfg.Defaults.Synthesizer)
	} else {
		fmt.Println("Default synthesizer: (none — executor doubles as synthesizer)")
	}
	fmt.Printf("Default timeout: %ds\n", cfg.Defaults.Timeout)

	return nil
}

// ---------------------------------------------------------------------------
// Hidden backward-compatibility aliases
// ---------------------------------------------------------------------------

// buildRunCmdHidden provides the old `openconvene run --task "..." --mode code`
// interface as a hidden command for backward compatibility.
func buildRunCmdHidden() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "run",
		Short:  "Execute a Convene flow (hidden alias — use positional args instead)",
		Long:   `Hidden backward-compatibility alias. Prefer: openconvene "task" or openconvene ask/agent "task".`,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			modeStr, _ := cmd.Flags().GetString("mode")
			// runConvene calls resolveTask which reads the --task flag.
			return runConvene(cmd, args, modeStr)
		},
	}

	cmd.Flags().String("task", "", "Task description (use '-' to read from stdin)")
	cmd.MarkFlagRequired("task")
	cmd.Flags().String("mode", "", "Execution mode: research|code|agent")
	cmd.MarkFlagRequired("mode")
	cmd.Flags().String("responders", "", "Comma-separated responder model names (overrides config defaults)")
	cmd.Flags().String("executor", "", "Executor model name (overrides config defaults)")
	cmd.Flags().String("synthesizer", "", "Synthesizer model name (overrides config defaults)")
	cmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	cmd.Flags().Int("timeout", 0, "Override per-call timeout in seconds (0 = use config default)")
	cmd.Flags().Bool("verbose", false, "Show raw responder responses and metadata")
	cmd.Flags().BoolP("print", "p", false, "Non-interactive mode (default: always non-interactive)")
	return cmd
}

// buildModelsInfoCmd creates the `openconvene models-info` command.
//
// This command queries each installed CLI for its available models and
// authentication status. For CLIs with a `models` subcommand (agy, grok),
// it executes the command and parses the output. For CLIs without one
// (devin, codex), it shows known model hints from documentation.
func buildModelsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "models-info",
		Short: "Show available models for each installed CLI",
		Long: `Query each installed CLI for its available models.

For CLIs with a 'models' subcommand (agy, grok), the command is executed
and the output is parsed. For CLIs without one (devin, codex), known model
hints from documentation are shown.

This helps you decide which models to configure in models.yaml, especially
for single-CLI multi-model MoA setups (e.g. multiple devin --model entries).`,
		RunE: runModelsInfo,
	}
}

func runModelsInfo(cmd *cobra.Command, args []string) error {
	infos := adapter.QueryCLIModels()

	fmt.Println("=== CLI Models Info ===")
	fmt.Println()

	installedCount := 0
	for _, info := range infos {
		status := "NOT INSTALLED"
		if info.Installed {
			status = "INSTALLED"
			installedCount++
		}

		fmt.Printf("--- %s [%s] ---\n", info.Name, status)
		if info.Error != "" && !info.Installed {
			fmt.Printf("  Error: %s\n\n", info.Error)
			continue
		}

		if info.DefaultModel != "" {
			fmt.Printf("  Default model: %s\n", info.DefaultModel)
		}

		if len(info.Models) > 0 {
			fmt.Printf("  Available models (%d):\n", len(info.Models))
			for _, m := range info.Models {
				fmt.Printf("    - %s\n", m)
			}
		} else if info.Error != "" {
			fmt.Printf("  Error querying models: %s\n", info.Error)
		}

		if info.RawOutput != "" && info.RawOutput != "(no models subcommand — using known hints)" {
			// Show raw output for debugging (truncated to 200 chars).
			raw := info.RawOutput
			if len(raw) > 200 {
				raw = raw[:200] + "..."
			}
			fmt.Printf("  Raw output: %s\n", strings.TrimSpace(raw))
		}
		fmt.Println()
	}

	fmt.Printf("Installed: %d / %d\n", installedCount, len(infos))
	fmt.Println("\nTip: Use these model names in your models.yaml command templates.")
	fmt.Println("     For CLIs with --model flag (devin, codex, grok, agy),")
	fmt.Println("     create separate config entries per model for MoA.")

	return nil
}

// buildListModelsCmdHidden provides the old `openconvene list-models` command
// as a hidden alias for `openconvene models`.
func buildListModelsCmdHidden() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "list-models",
		Short:  "List configured models (hidden alias — use 'models' instead)",
		Hidden: true,
		RunE:   runListModels,
	}

	cmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	return cmd
}

// buildConfigCmdHidden provides the old `openconvene config init/validate`
// commands as hidden aliases for `openconvene init/check`.
func buildConfigCmdHidden() *cobra.Command {
	configCmd := &cobra.Command{
		Use:    "config",
		Short:  "Config management (hidden alias — use init/check instead)",
		Hidden: true,
	}

	// config init → alias for init
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter models.yaml (hidden alias — use 'openconvene init' instead)",
		RunE:  runInit,
	}
	initCmd.Flags().String("path", "", "Output path for models.yaml (default: config/models.yaml)")
	configCmd.AddCommand(initCmd)

	// config validate → alias for check
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate models.yaml (hidden alias — use 'openconvene check' instead)",
		RunE:  runCheck,
	}
	validateCmd.Flags().String("config", "", "Path to models.yaml (default: search standard locations)")
	configCmd.AddCommand(validateCmd)

	return configCmd
}
