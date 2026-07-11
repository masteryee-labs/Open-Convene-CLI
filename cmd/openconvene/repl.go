// repl.go — Interactive REPL for openconvene.
//
// When the user runs `openconvene`, `openconvene ask`, or `openconvene agent`
// without a task argument, the CLI enters an interactive REPL (Read-Eval-Print
// Loop) similar to codex, grok, agy, and devin.
//
// In the REPL, the user can:
//   - Type a prompt directly → it runs through the Convene pipeline in the
//     current mode.
//   - Use slash commands to inspect state, switch modes, change models, and
//     view usage statistics.
//
// Slash commands (aligned with Devin, Codex, agy, and Grok CLIs):
//
//	/help, /h, /?           Show available commands
//	/status                 Show session status (mode, models, run count)
//	/models, /m             List all configured models
//	/mode [ask|code|agent]  Show or switch current mode
//	/responders [a,b,c]     Show or set responders
//	/executor [name]        Show or set executor
//	/synthesizer [name]     Show or set synthesizer (or "none")
//	/usage, /u              Show session usage statistics (per-CLI call counts)
//	/config, /c, /settings  Show current configuration summary
//	/detect, /d             Detect installed CLIs
//	/clear, /new            Clear screen and reset session
//	/compact                (stub) Summarize conversation to free tokens
//	/resume, /continue      (stub) Resume a previous session
//	/update                 Show update instructions (copy-paste install command)
//	/exit, /quit, /q        Exit the REPL
//
// Usage tracking:
//
// The REPL accumulates per-session statistics: how many times each CLI was
// called as responder/synthesizer/executor, total elapsed time, and
// success/failure counts. This is displayed via /usage.

package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/masteryee-labs/open-convene-cli/internal/adapter"
	"github.com/masteryee-labs/open-convene-cli/internal/config"
	"github.com/masteryee-labs/open-convene-cli/internal/convene"
	"github.com/masteryee-labs/open-convene-cli/internal/mode"
	"github.com/masteryee-labs/open-convene-cli/internal/version"
	"github.com/reeflective/readline"
	"github.com/reeflective/readline/inputrc"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Session state
// ---------------------------------------------------------------------------

// replSession holds the mutable state of an interactive REPL session.
type replSession struct {
	cfg *config.ConveneConfig

	// Current mode: "research" (ask), "code", or "agent".
	currentMode string

	// Current model overrides (empty = use config defaults).
	responders    []string
	executor       string
	synthesizer    string
	configPath     string
	defaultTimeout int

	// Language for model responses (empty = no preference).
	language string

	// Usage tracking.
	usage *sessionUsage
}

// sessionUsage tracks per-CLI call statistics across the REPL session.
type sessionUsage struct {
	// calls maps CLI name → total call count (as responder, synthesizer, or executor).
	calls map[string]int

	// successes maps CLI name → successful call count.
	successes map[string]int

	// failures maps CLI name → failed call count.
	failures map[string]int

	// totalElapsed maps CLI name → accumulated elapsed time.
	totalElapsed map[string]time.Duration

	// totalRuns is the total number of Convene runs in this session.
	totalRuns int

	// sessionStart is when the REPL session began.
	sessionStart time.Time
}

func newSessionUsage() *sessionUsage {
	return &sessionUsage{
		calls:        make(map[string]int),
		successes:    make(map[string]int),
		failures:     make(map[string]int),
		totalElapsed: make(map[string]time.Duration),
		sessionStart: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// REPL entry point
// ---------------------------------------------------------------------------

// runREPL starts the interactive REPL loop.
//
// initialMode is the mode to start in ("research", "code", or "agent").
// cfg is the loaded configuration. configPath is the path to the config file
// (for display purposes).
//
// The REPL reads lines from stdin. Lines starting with "/" are slash commands.
// All other lines are treated as task prompts and run through the Convene
// pipeline.
func runREPL(initialMode string, cfg *config.ConveneConfig, configPath string) error {
	session := &replSession{
		cfg:            cfg,
		currentMode:    initialMode,
		responders:     cfg.Defaults.Responders,
		executor:       cfg.Defaults.Executor,
		configPath:     configPath,
		defaultTimeout: cfg.Defaults.Timeout,
		language:       cfg.Defaults.Language,
		usage:          newSessionUsage(),
	}
	if cfg.Defaults.Synthesizer != nil {
		session.synthesizer = *cfg.Defaults.Synthesizer
	}

	// Print welcome banner.
	printWelcome(session)

	// Check if stdin AND stdout are both terminals. If either is not
	// (e.g. piped input/output in tests), use the basic fallback reader.
	// reeflective/readline requires both to be interactive terminals;
	// it blocks indefinitely if stdin is a terminal but stdout is piped.
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return runREPLBasic(session)
	}

	// Set up reeflective/readline shell with menu-complete (fish-style) Tab behavior.
	// Tab is rebound from "complete" (bash-style) to "menu-complete" (fish-style):
	// pressing Tab shows a completion menu, and up/down arrows navigate candidates.
	rl := readline.NewShell()

	// Bind Tab to menu-complete in emacs mode (the default).
	rl.Config.Binds["emacs"][inputrc.Unescape(`\C-i`)] = inputrc.Bind{Action: "menu-complete"}
	// Shift-Tab cycles backward through completions.
	rl.Config.Binds["emacs"][inputrc.Unescape(`\e[Z`)] = inputrc.Bind{Action: "menu-complete-backward"}

	// Set up the prompt (dynamic — re-evaluated each loop iteration).
	// Devin-style: ❭ symbol with no trailing text.
	rl.Prompt.Primary(func() string {
		return session.prompt()
	})

	// Right prompt: show current executor model name (like Devin shows "GLM-5.2 High").
	rl.Prompt.Right(func() string {
		return session.rightPrompt()
	})

	// Persistent hint: shown below the input line at all times.
	rl.Hint.Persist(session.hintText())

	// Set up file-based history.
	histFile := historyFilePath()
	if hist, err := readline.NewHistoryFromFile(histFile); err == nil {
		rl.History.Add("file", hist)
	}

	// Set up the completer (session-aware, two-phase slash command + argument completion).
	// The completer adds navigation hints to the completion menu.
	completer := &slashCompleter{session: session}
	rl.Completer = func(line []rune, cursor int) readline.Completions {
		return completer.completeReeflective(line, cursor)
	}

	for {
		line, err := rl.Readline()
		if err != nil {
			break // EOF (Ctrl+D) or interrupt (Ctrl+C)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			// Slash command.
			if shouldExit := handleSlashCommand(session, line); shouldExit {
				break
			}
			continue
		}

		// Regular prompt → run Convene.
		runPromptInREPL(session, line)
	}

	// Print session summary on exit.
	fmt.Println()
	printSessionSummary(session)
	return nil
}

// isTerminal returns true if the given file is a terminal (not a pipe or file).
func isTerminal(f *os.File) bool {
	// On Windows, check if the file descriptor is a character device.
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	// Character devices (terminals) have ModeCharDevice set.
	// Pipes and regular files do not.
	return fi.Mode()&os.ModeCharDevice != 0
}

// runREPLBasic is a fallback when readline is unavailable or stdin is not a
// terminal (e.g. piped input in tests). It uses bufio.Scanner for simple
// line-by-line reading without history or tab completion.
func runREPLBasic(session *replSession) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		fmt.Print("❭ ")
		if !scanner.Scan() {
			break // EOF (Ctrl+D)
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if shouldExit := handleSlashCommand(session, line); shouldExit {
				break
			}
			continue
		}
		runPromptInREPL(session, line)
	}

	fmt.Println()
	printSessionSummary(session)
	return nil
}

// historyFilePath returns the path to the readline history file.
func historyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".openconvene_history"
	}
	return filepath.Join(home, ".openconvene_history")
}

// ---------------------------------------------------------------------------
// Slash command tab completion
// ---------------------------------------------------------------------------

// slashCompleter is a session-aware completer for slash commands.
// It supports two-phase completion:
//   - Phase 1: completing the command name (e.g. /ex → /executor, /exit)
//   - Phase 2: completing arguments after the command (e.g. /executor d → devin, devin:glm-5.2)
//
// The completer is session-aware: it reads model names from the session's config
// to provide relevant argument completions.
type slashCompleter struct {
	session *replSession
}

// allSlashCommands returns every slash command and alias recognized by the REPL.
func allSlashCommands() []string {
	return []string{
		"/help", "/h", "/?",
		"/models", "/m",
		"/detect", "/d",
		"/mode",
		"/responders",
		"/executor",
		"/synthesizer",
		"/language", "/lang",
		"/choose",
		"/usage", "/u",
		"/status",
		"/config", "/c", "/settings",
		"/compact",
		"/clear", "/new",
		"/resume", "/continue",
		"/update",
		"/exit", "/quit", "/q",
	}
}

// commandsWithArgs maps each command (and aliases) to whether it accepts arguments
// that can be tab-completed. Commands not in this map either take no arguments
// or take free-form text (no meaningful completion).
var commandsWithArgs = map[string]bool{
	"/mode":        true,
	"/responders":  true,
	"/executor":    true,
	"/synthesizer": true,
	"/language":    true,
	"/lang":        true,
}

// modelNamesForCompletion returns all model names available for completion.
// This includes keys from the config's models map plus any dynamic model names
// currently in use (responders, executor, synthesizer).
func (c *slashCompleter) modelNamesForCompletion() []string {
	seen := make(map[string]bool)
	var names []string

	// Add models from config.
	if c.session != nil && c.session.cfg != nil {
		for name := range c.session.cfg.Models {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		// Add current responders/executor/synthesizer (may be dynamic names).
		for _, r := range c.session.responders {
			if r != "" && !seen[r] {
				seen[r] = true
				names = append(names, r)
			}
		}
		if c.session.executor != "" && !seen[c.session.executor] {
			seen[c.session.executor] = true
			names = append(names, c.session.executor)
		}
		if c.session.synthesizer != "" && !seen[c.session.synthesizer] {
			seen[c.session.synthesizer] = true
			names = append(names, c.session.synthesizer)
		}
	}

	// Add some common dynamic model prefixes so users get hints.
	// These are commonly used CLI:model format names.
	commonDynamic := []string{
		"devin:glm-5.2", "devin:swe-1.7", "devin:kimi-k2.7",
		"agy:gemini-3.5-flash", "agy:gemini-3.5-pro",
		"grok:grok-4.5", "codex:gpt-5",
	}
	for _, d := range commonDynamic {
		if !seen[d] {
			seen[d] = true
			names = append(names, d)
		}
	}

	sort.Strings(names)
	return names
}

// commonLanguages returns commonly used language values for /language completion.
func commonLanguages() []string {
	return []string{
		"zh-TW", "zh-CN", "繁體中文", "简体中文",
		"English", "日本語", "한국어", "Français", "Deutsch", "Español",
		"none",
	}
}

// modeValues returns valid mode values for /mode completion.
func modeValues() []string {
	return []string{"ask", "code", "agent"}
}

// completeReeflective is the completion function for reeflective/readline.
// It returns readline.Completions (a struct with candidates) instead of [][]rune.
//
// Behavior:
//   - Only completes when the line starts with "/".
//   - No space in input: complete command names (e.g. /ex → /executor, /exit)
//   - Space after command: complete arguments based on the command type
//   - Partial argument text: filter completions by prefix match
//   - All completions include Devin-style navigation hints via Usage()
func (c *slashCompleter) completeReeflective(line []rune, cursor int) readline.Completions {
	// Only complete when the line starts with "/".
	if len(line) == 0 || line[0] != '/' {
		return readline.Completions{}
	}

	// Get the partial input up to the cursor.
	input := string(line[:cursor])

	// Find the first space — it separates command from arguments.
	spaceIdx := strings.Index(input, " ")

	if spaceIdx == -1 {
		// Phase 1: completing the command name itself (no space typed yet).
		return c.completeCommandReeflective(input)
	}

	// Phase 2: completing arguments after the command.
	cmd := strings.ToLower(input[:spaceIdx])
	argPartial := input[spaceIdx+1:]

	// Check if this command supports argument completion.
	if !commandsWithArgs[cmd] {
		return readline.Completions{}
	}

	return c.completeArgsReeflective(cmd, argPartial)
}

// completionHints is the navigation hint text shown at the bottom of the
// completion menu, matching Devin CLI's style.
const completionHints = "tab next · shift+tab prev · ↵ accept · esc close"

// completeCommandReeflective returns completions for a partial slash command.
func (c *slashCompleter) completeCommandReeflective(partial string) readline.Completions {
	var matches []string
	for _, cmd := range allSlashCommands() {
		if strings.HasPrefix(cmd, partial) {
			matches = append(matches, cmd)
		}
	}
	sort.Strings(matches)
	return readline.CompleteValues(matches...).Usage(completionHints)
}

// completeArgsReeflective returns completions for arguments of a specific slash command.
func (c *slashCompleter) completeArgsReeflective(cmd, argPartial string) readline.Completions {
	var candidates []string

	switch cmd {
	case "/mode":
		candidates = modeValues()

	case "/executor":
		candidates = c.modelNamesForCompletion()

	case "/synthesizer":
		candidates = c.modelNamesForCompletion()
		candidates = append(candidates, "none")

	case "/responders":
		// /responders takes comma-separated model names.
		// We complete the last segment after the last comma.
		lastComma := strings.LastIndex(argPartial, ",")
		segmentStart := lastComma + 1
		segment := argPartial[segmentStart:]
		prefix := argPartial[:segmentStart]

		allModels := c.modelNamesForCompletion()
		var matches []string
		for _, m := range allModels {
			if strings.HasPrefix(m, segment) {
				matches = append(matches, prefix+m)
			}
		}
		sort.Strings(matches)
		return readline.CompleteValues(matches...).NoSpace(',').Usage(completionHints)

	case "/language", "/lang":
		candidates = commonLanguages()

	default:
		return readline.Completions{}
	}

	// Filter candidates by prefix match against argPartial.
	var matches []string
	for _, cand := range candidates {
		if strings.HasPrefix(cand, argPartial) {
			matches = append(matches, cand)
		}
	}
	sort.Strings(matches)
	return readline.CompleteValues(matches...).Usage(completionHints)
}

// prompt returns the REPL prompt string — Devin-style ❭ symbol.
func (s *replSession) prompt() string {
	return "\n❭ "
}

// rightPrompt returns the text shown on the right side of the input line.
// Like Devin CLI showing the current model name (e.g. "GLM-5.2 High").
func (s *replSession) rightPrompt() string {
	execModel := s.executor
	if execModel == "" {
		execModel = "(no executor)"
	}
	return execModel
}

// hintText returns the persistent hint text shown below the input line.
func (s *replSession) hintText() string {
	return "Type a prompt to run it, or /help for commands · /exit to quit"
}

// separatorLine returns a full-width separator line using ─ characters,
// with an optional label appended at the right end (Devin-style).
func separatorLine(label string) string {
	width := terminalWidth()
	labelPart := ""
	if label != "" {
		labelPart = " (" + label + ") ─"
	}
	// Reserve space for the label + padding.
	avail := width - len(labelPart) - 1
	if avail < 10 {
		avail = 10
	}
	return strings.Repeat("─", avail) + labelPart
}

// terminalWidth returns the terminal width, or 80 as fallback.
func terminalWidth() int {
	// Use readline's term package to get the width.
	return 80
}

// printWelcome prints the REPL welcome banner — Devin-style layout.
func printWelcome(s *replSession) {
	modeDisplay := s.currentMode
	if s.currentMode == "research" {
		modeDisplay = "ask"
	}

	langDisplay := s.language
	if langDisplay == "" {
		langDisplay = "(default)"
	}

	responderCount := len(s.responders)

	// ASCII art logo (compact figlet-style) + version/mode info on the right.
	logo := `  ___  ___    ___ ___
 / _ \| _ \  / __| __|
| (_) |  _/  \__ \ _|
 \___/|_|   |___/___|`

	// Print logo with version/mode info aligned to the right of it.
	logoLines := strings.Split(logo, "\n")
	infoLines := []string{
		"OpenConveneCLI",
		fmt.Sprintf("v%s · %s mode", version.Version, modeDisplay),
		fmt.Sprintf("%d responders · %s", responderCount, orDash(s.executor)),
	}

	for i, line := range logoLines {
		info := ""
		if i < len(infoLines) {
			info = "  " + infoLines[i]
		}
		fmt.Printf("  %s%s\n", line, info)
	}

	// Model configuration details.
	fmt.Println()
	fmt.Printf("  Responders:  %s\n", strings.Join(s.responders, ", "))
	fmt.Printf("  Executor:    %s\n", orDash(s.executor))
	fmt.Printf("  Synthesizer: %s\n", orDash(s.synthesizer))
	fmt.Printf("  Language:    %s\n", langDisplay)
	fmt.Println()

	// Separator line with mode indicator (Devin-style).
	fmt.Println(separatorLine(modeDisplay + " mode"))
}

// ---------------------------------------------------------------------------
// Slash command handling
// ---------------------------------------------------------------------------

// handleSlashCommand parses and executes a slash command.
// Returns true if the REPL should exit.
func handleSlashCommand(s *replSession, line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help", "/h", "/?":
		printHelp()

	case "/models", "/m":
		printModelsInREPL(s)

	case "/detect", "/d":
		printDetectInREPL()

	case "/mode":
		handleModeCommand(s, args)

	case "/responders":
		handleRespondersCommand(s, args)

	case "/executor":
		handleExecutorCommand(s, args)

	case "/synthesizer":
		handleSynthesizerCommand(s, args)

	case "/language", "/lang":
		handleLanguageCommand(s, args)

	case "/choose":
		handleChooseCommand(args)

	case "/usage", "/u":
		printUsage(s)

	case "/status":
		printStatus(s)

	case "/config", "/c", "/settings":
		printConfigSummary(s)

	case "/compact":
		fmt.Println("(stub) /compact — session compaction not yet implemented.")
		fmt.Println("  In a future version this will summarize the conversation to free tokens.")

	case "/clear", "/new":
		// /clear and /new both reset the session (like Devin/Codex).
		s.usage = newSessionUsage()
		fmt.Print("\033[2J\033[H") // ANSI clear screen + home
		fmt.Println("(session cleared)")

	case "/resume", "/continue":
		fmt.Println("(stub) /resume — session resume not yet implemented.")
		fmt.Println("  In a future version this will resume a previous session by ID.")
		fmt.Println("  Usage: /resume [session-id]  or  /continue [session-id]")

	case "/update":
		handleUpdateCommand()

	case "/exit", "/quit", "/q":
		return true

	default:
		fmt.Printf("Unknown command: %s (type /help for available commands)\n", cmd)
	}

	return false
}

// printHelp shows all available slash commands.
func printHelp() {
	fmt.Println(`
Available commands:

Navigation & control:
  /help, /h, /?           Show this help
  /status                 Show session status (mode, models, run count)
  /clear, /new            Clear screen and reset session
  /compact                (stub) Summarize conversation to free tokens
  /exit, /quit, /q        Exit REPL

Mode & model:
  /mode [ask|code|agent]  Show or switch current mode
  /models, /m             List all configured models
  /responders [a,b,c]     Show or set responders (comma-separated)
  /executor [name]        Show or set executor
  /synthesizer [name]     Show or set synthesizer ("none" to clear)
  /language [lang]        Show or set output language (e.g. zh-TW, 繁體中文, English)
  /lang [lang]            Alias for /language ("none" to clear)
  /choose <title|opt1|..> Show interactive decision menu (↑↓ navigate, ↵ select)

Session & config:
  /usage, /u              Show session usage statistics (per-CLI calls)
  /config, /c, /settings  Show current configuration
  /detect, /d             Detect installed CLIs
  /resume, /continue      (stub) Resume a previous session
  /update                 Show update instructions

  Or type any text to run it as a prompt in the current mode.`)
}

// handleModeCommand handles /mode [ask|code|agent].
func handleModeCommand(s *replSession, args []string) {
	if len(args) == 0 {
		display := s.currentMode
		if s.currentMode == "research" {
			display = "ask"
		}
		fmt.Printf("Current mode: %s\n", display)
		fmt.Println("Available options:")
		fmt.Println("  ask    — research mode (no execution, analysis only)")
		fmt.Println("  code   — code mode (with execution, writes code)")
		fmt.Println("  agent  — agent mode (with execution, long-running agent tasks)")
		fmt.Println("Usage: /mode <ask|code|agent>  (or press Tab after /mode )")
		return
	}

	switch strings.ToLower(args[0]) {
	case "ask", "research":
		s.currentMode = "research"
		fmt.Println("Mode switched to: ask (research, no execution)")
	case "code":
		s.currentMode = "code"
		fmt.Println("Mode switched to: code (with execution)")
	case "agent":
		s.currentMode = "agent"
		fmt.Println("Mode switched to: agent (with execution)")
	default:
		fmt.Printf("Unknown mode: %s (available: ask, code, agent)\n", args[0])
	}
}

// handleRespondersCommand handles /responders [a,b,c].
func handleRespondersCommand(s *replSession, args []string) {
	if len(args) == 0 {
		fmt.Printf("Current responders: %s\n", strings.Join(s.responders, ", "))
		fmt.Println("Available models (press Tab after /responders to auto-complete):")
		printAvailableModels(s)
		fmt.Println("Usage: /responders <model1,model2,...>  (comma-separated)")
		fmt.Println("  Tip: you can also use dynamic model names like devin:glm-5.2")
		return
	}

	names := strings.Split(args[0], ",")
	var valid []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, exists := s.cfg.Models[n]; !exists {
			// May be a dynamic model name (CLI-模型名) — accepted at runtime.
			fmt.Printf("NOTE: %q not in models section; will be resolved as dynamic model name at runtime\n", n)
		}
		valid = append(valid, n)
	}
	if len(valid) == 0 {
		fmt.Println("No valid responders specified.")
		return
	}
	s.responders = valid
	fmt.Printf("Responders set to: %s\n", strings.Join(s.responders, ", "))
}

// handleExecutorCommand handles /executor [name].
func handleExecutorCommand(s *replSession, args []string) {
	if len(args) == 0 {
		fmt.Printf("Current executor: %s\n", orDash(s.executor))
		fmt.Println("Available models (press Tab after /executor to auto-complete):")
		printAvailableModels(s)
		fmt.Println("Usage: /executor <model-name>")
		fmt.Println("  Tip: you can also use dynamic model names like devin:glm-5.2")
		return
	}
	name := strings.TrimSpace(args[0])
	if name == "none" || name == "" {
		s.executor = ""
		fmt.Println("Executor cleared.")
		return
	}
	if _, exists := s.cfg.Models[name]; !exists {
		fmt.Printf("NOTE: %q not in models section; will be resolved as dynamic model name at runtime\n", name)
	}
	s.executor = name
	fmt.Printf("Executor set to: %s\n", s.executor)
}

// handleSynthesizerCommand handles /synthesizer [name].
func handleSynthesizerCommand(s *replSession, args []string) {
	if len(args) == 0 {
		fmt.Printf("Current synthesizer: %s\n", orDash(s.synthesizer))
		fmt.Println("Available models (press Tab after /synthesizer to auto-complete):")
		printAvailableModels(s)
		fmt.Println("Special: /synthesizer none  (clear → executor doubles as synthesizer)")
		fmt.Println("Usage: /synthesizer <model-name|none>")
		fmt.Println("  Tip: you can also use dynamic model names like devin:glm-5.2")
		return
	}
	name := strings.TrimSpace(args[0])
	if name == "none" || name == "" {
		s.synthesizer = ""
		fmt.Println("Synthesizer cleared (executor will double as synthesizer).")
		return
	}
	if _, exists := s.cfg.Models[name]; !exists {
		fmt.Printf("NOTE: %q not in models section; will be resolved as dynamic model name at runtime\n", name)
	}
	s.synthesizer = name
	fmt.Printf("Synthesizer set to: %s\n", s.synthesizer)
}

// handleLanguageCommand handles /language [lang] — show or set the output language.
// Examples: /language zh-TW, /language 繁體中文, /language en, /language none
func handleLanguageCommand(s *replSession, args []string) {
	if len(args) == 0 {
		if s.language == "" {
			fmt.Println("Current language: (none — model defaults)")
		} else {
			fmt.Printf("Current language: %s\n", s.language)
		}
		fmt.Println("Available options (press Tab after /language to auto-complete):")
		fmt.Println("  zh-TW, zh-CN, 繁體中文, 简体中文")
		fmt.Println("  English, 日本語, 한국어, Français, Deutsch, Español")
		fmt.Println("  none  (clear language preference)")
		fmt.Println("Usage: /language <lang>  (or any custom language string)")
		return
	}

	lang := strings.TrimSpace(strings.Join(args, " "))
	if lang == "none" || lang == "" {
		s.language = ""
		s.cfg.Defaults.Language = ""
		fmt.Println("Language cleared — models will use their default language.")
		// Persist the cleared language to config.
		if s.configPath != "" {
			if err := saveLanguageToConfig(s.configPath, ""); err != nil {
				fmt.Fprintf(os.Stderr, "  (warning: could not save language to config: %v)\n", err)
			}
		}
		return
	}

	s.language = lang
	s.cfg.Defaults.Language = lang
	fmt.Printf("Language set to: %s\n", lang)
	fmt.Println("  (Model responses will be in this language. CLI commands remain in English.)")

	// Persist to config file so it survives across sessions.
	if s.configPath != "" {
		if err := saveLanguageToConfig(s.configPath, lang); err != nil {
			fmt.Fprintf(os.Stderr, "  (warning: could not save language to config: %v)\n", err)
		}
	}
}

// printStatus shows a concise session status (like Codex's /status).
func printStatus(s *replSession) {
	modeDisplay := s.currentMode
	if s.currentMode == "research" {
		modeDisplay = "ask"
	}

	langDisplay := s.language
	if langDisplay == "" {
		langDisplay = "(default)"
	}

	fmt.Println("=== Session Status ===")
	fmt.Printf("  Mode:          %s\n", modeDisplay)
	fmt.Printf("  Model (exec):  %s\n", orDash(s.executor))
	fmt.Printf("  Responders:    %s\n", strings.Join(s.responders, ", "))
	fmt.Printf("  Synthesizer:   %s\n", orDash(s.synthesizer))
	fmt.Printf("  Language:      %s\n", langDisplay)
	fmt.Printf("  Runs:          %d\n", s.usage.totalRuns)
	fmt.Printf("  Session time:  %s\n", time.Since(s.usage.sessionStart).Round(time.Second))
}

// ---------------------------------------------------------------------------
// /models, /detect, /config, /usage — display commands
// ---------------------------------------------------------------------------

// printAvailableModels prints a compact list of model names available for
// /executor, /synthesizer, and /responders commands. Used when those commands
// are called without arguments to show the user what they can pick.
func printAvailableModels(s *replSession) {
	if len(s.cfg.Models) == 0 {
		fmt.Println("  (no models in config — use dynamic names like devin:glm-5.2)")
		return
	}

	names := make([]string, 0, len(s.cfg.Models))
	for name := range s.cfg.Models {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print in a compact multi-column format.
	const cols = 4
	for i := 0; i < len(names); i += cols {
		end := i + cols
		if end > len(names) {
			end = len(names)
		}
		row := names[i:end]
		fmt.Printf("  %s\n", strings.Join(row, ", "))
	}
}

func printModelsInREPL(s *replSession) {
	if len(s.cfg.Models) == 0 {
		fmt.Println("No models configured.")
		return
	}

	names := make([]string, 0, len(s.cfg.Models))
	for name := range s.cfg.Models {
		names = append(names, name)
	}
	sort.Strings(names)

	detected := adapter.DetectAvailableAdapters()
	installedMap := make(map[string]bool, len(detected))
	for _, d := range detected {
		installedMap[d.Name] = d.Found
	}

	fmt.Printf("%-12s %-10s %-16s %-10s\n",
		"MODEL", "READ_ONLY", "EXECUTOR_CAPABLE", "INSTALLED")
	fmt.Printf("%-12s %-10s %-16s %-10s\n",
		"------------", "----------", "----------------", "----------")

	for _, name := range names {
		m := s.cfg.Models[name]
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
		fmt.Printf("%-12s %-10s %-16s %-10s\n",
			name, readOnly, execCapable, installed)
	}
	fmt.Printf("\nTotal: %d model(s)\n", len(s.cfg.Models))
}

func printDetectInREPL() {
	results := adapter.DetectAvailableAdapters()

	fmt.Printf("%-12s %-10s %-40s %-10s %-12s %-12s\n",
		"CLI", "INSTALLED", "PATH", "READ_ONLY", "CAN_RESPOND", "CAN_EXECUTE")
	fmt.Printf("%-12s %-10s %-40s %-10s %-12s %-12s\n",
		"------------", "----------", "----------------------------------------", "----------", "------------", "------------")

	var installedCount int
	for _, r := range results {
		installed := "no"
		if r.Found {
			installed = "yes"
			installedCount++
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
	}
	fmt.Printf("\nInstalled: %d / %d\n", installedCount, len(results))
}

func printConfigSummary(s *replSession) {
	modeDisplay := s.currentMode
	if s.currentMode == "research" {
		modeDisplay = "ask"
	}

	langDisplay := s.language
	if langDisplay == "" {
		langDisplay = "(default)"
	}

	fmt.Println("=== Current Configuration ===")
	fmt.Printf("  Config file:  %s\n", orDash(s.configPath))
	fmt.Printf("  Mode:         %s\n", modeDisplay)
	fmt.Printf("  Responders:   %s\n", strings.Join(s.responders, ", "))
	fmt.Printf("  Executor:     %s\n", orDash(s.executor))
	fmt.Printf("  Synthesizer:  %s\n", orDash(s.synthesizer))
	fmt.Printf("  Language:     %s\n", langDisplay)
	fmt.Printf("  Timeout:      %ds\n", s.defaultTimeout)
	fmt.Printf("  Models count: %d\n", len(s.cfg.Models))
}

func printUsage(s *replSession) {
	u := s.usage

	if u.totalRuns == 0 {
		fmt.Println("No runs yet in this session.")
		fmt.Println("Type a prompt to start using Convene.")
		return
	}

	fmt.Println("=== Session Usage Statistics ===")
	fmt.Printf("  Session duration: %s\n", time.Since(u.sessionStart).Round(time.Second))
	fmt.Printf("  Total runs:       %d\n", u.totalRuns)
	fmt.Println()

	// Collect all CLI names that were called.
	allNames := make(map[string]bool)
	for name := range u.calls {
		allNames[name] = true
	}

	if len(allNames) == 0 {
		fmt.Println("  No CLI calls recorded.")
		return
	}

	names := make([]string, 0, len(allNames))
	for name := range allNames {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("  %-12s %6s %8s %8s %12s\n",
		"CLI", "CALLS", "SUCCESS", "FAILED", "TOTAL TIME")
	fmt.Printf("  %-12s %6s %8s %8s %12s\n",
		"------------", "------", "--------", "--------", "------------")

	for _, name := range names {
		calls := u.calls[name]
		successes := u.successes[name]
		failures := u.failures[name]
		elapsed := u.totalElapsed[name]
		fmt.Printf("  %-12s %6d %8d %8d %12s\n",
			name, calls, successes, failures, elapsed.Round(time.Millisecond))
	}

	totalCalls := 0
	totalSuccesses := 0
	totalFailures := 0
	for _, c := range u.calls {
		totalCalls += c
	}
	for _, s := range u.successes {
		totalSuccesses += s
	}
	for _, f := range u.failures {
		totalFailures += f
	}
	fmt.Printf("  %-12s %6d %8d %8d\n",
		"TOTAL", totalCalls, totalSuccesses, totalFailures)
}

// ---------------------------------------------------------------------------
// Running prompts in the REPL
// ---------------------------------------------------------------------------

// runPromptInREPL runs a single prompt through the Convene pipeline using
// the session's current mode and model settings.
//
// After printing the result, it checks for [[DECISION]] markers in the AI
// output. If found, it displays the interactive decision menu and feeds
// the user's choice back as a new prompt (recursive call).
func runPromptInREPL(s *replSession, task string) {
	// Validate mode + model combination.
	validationErrors, validationWarnings := mode.ValidateModeConfig(
		mode.Mode(s.currentMode), s.responders, s.executor,
		stringToPtr(s.synthesizer), s.cfg.Models)

	if len(validationErrors) > 0 {
		fmt.Fprintln(os.Stderr, "ERROR: mode validation failed:")
		for _, e := range validationErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		fmt.Println("(Use /mode, /responders, /executor, /synthesizer to adjust)")
		return
	}

	for _, w := range validationWarnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

	// Set up synthesizer pointer.
	var synthPtr *string
	if s.synthesizer != "" {
		synthPtr = &s.synthesizer
	}

	// Run Convene.
	fmt.Fprintf(os.Stderr, "Running in %s mode with responders [%s]...\n",
		s.currentMode, strings.Join(s.responders, ", "))

	ctx := context.Background()
	engine := convene.NewConveneEngine(s.cfg)

	result, err := engine.Run(ctx, task, s.currentMode, s.responders, s.executor, synthPtr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}

	// Record usage stats.
	recordUsage(s.usage, &result, s.synthesizer, s.executor)

	// Format and print the output.
	m := mode.Mode(s.currentMode)
	output := mode.FormatOutput(result, m)

	// Check for [[DECISION]] markers in the output.
	decision, cleanedOutput := ParseDecision(output)
	if decision != nil {
		// Print the output without the decision marker block.
		fmt.Print(cleanedOutput)
		fmt.Println()

		// Present the interactive decision menu.
		followUp := presentDecision(decision)
		if followUp != "" {
			fmt.Println()
			// Feed the user's choice back as a new prompt.
			runPromptInREPL(s, followUp)
		}
	} else {
		// No decision marker — print normally.
		fmt.Print(output)
	}
}

// recordUsage updates session usage stats from a ConveneResult.
func recordUsage(u *sessionUsage, result *convene.ConveneResult, synthName, execName string) {
	u.totalRuns++

	if result == nil || result.Metadata == nil {
		return
	}

	// Track responders.
	for name := range result.Responses {
		u.calls[name]++
		u.successes[name]++
		if elapsed, ok := result.Metadata[fmt.Sprintf("%s_elapsed", name)]; ok {
			if d, ok := elapsed.(time.Duration); ok {
				u.totalElapsed[name] += d
			}
		}
	}

	// Track failed responders.
	for k, v := range result.Metadata {
		if strings.HasSuffix(k, "_success") {
			name := strings.TrimSuffix(k, "_success")
			if v == false {
				if _, isResponse := result.Responses[name]; !isResponse {
					u.calls[name]++
					u.failures[name]++
				}
			}
			if elapsed, ok := result.Metadata[fmt.Sprintf("%s_elapsed", name)]; ok {
				if d, ok := elapsed.(time.Duration); ok {
					u.totalElapsed[name] += d
				}
			}
		}
	}

	// Track synthesizer.
	if synthName != "" {
		if v, ok := result.Metadata["synthesizer_success"]; ok {
			u.calls[synthName]++
			if v == true {
				u.successes[synthName]++
			} else {
				u.failures[synthName]++
			}
			if elapsed, ok := result.Metadata["synthesizer_elapsed"]; ok {
				if d, ok := elapsed.(time.Duration); ok {
					u.totalElapsed[synthName] += d
				}
			}
		}
	}

	// Track executor.
	if execName != "" {
		if v, ok := result.Metadata["executor_success"]; ok {
			u.calls[execName]++
			if v == true {
				u.successes[execName]++
			} else {
				u.failures[execName]++
			}
			if elapsed, ok := result.Metadata["executor_elapsed"]; ok {
				if d, ok := elapsed.(time.Duration); ok {
					u.totalElapsed[execName] += d
				}
			}
		}
	}
}

// printSessionSummary prints a summary when the REPL exits.
func printSessionSummary(s *replSession) {
	if s.usage.totalRuns == 0 {
		fmt.Println("No prompts were run in this session.")
		return
	}

	fmt.Printf("=== Session Summary ===\n")
	fmt.Printf("  Total runs: %d\n", s.usage.totalRuns)
	fmt.Printf("  Duration:   %s\n", time.Since(s.usage.sessionStart).Round(time.Second))

	totalCalls := 0
	for _, c := range s.usage.calls {
		totalCalls += c
	}
	fmt.Printf("  Total CLI calls: %d\n", totalCalls)
	fmt.Println()
	fmt.Println("  Per-CLI breakdown:")
	fmt.Printf("  %-12s %6s %8s %8s\n", "CLI", "CALLS", "SUCCESS", "FAILED")
	fmt.Printf("  %-12s %6s %8s %8s\n", "------------", "------", "--------", "--------")

	names := make([]string, 0, len(s.usage.calls))
	for name := range s.usage.calls {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("  %-12s %6d %8d %8d\n",
			name, s.usage.calls[name], s.usage.successes[name], s.usage.failures[name])
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// orDash returns s or "—" if empty.
func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// saveLanguageToConfig updates the language field in the config file on disk.
// It reads the existing YAML, updates the defaults.language field, and writes
// it back. This preserves all other config settings.
//
// If the config file doesn't have a defaults section or language field, it
// adds one. If the file can't be parsed as YAML, it falls back to a simple
// text replacement.
func saveLanguageToConfig(configPath, language string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	content := string(data)

	// Parse as YAML, update, and write back.
	// We use a simple approach: parse into a map, update the language field,
	// and marshal back. This preserves structure better than text replacement.
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Can't parse as YAML — fall back to text replacement.
		return saveLanguageByTextReplace(configPath, content, language)
	}

	defaults, ok := cfg["defaults"].(map[string]interface{})
	if !ok {
		defaults = make(map[string]interface{})
		cfg["defaults"] = defaults
	}

	if language == "" {
		delete(defaults, "language")
	} else {
		defaults["language"] = language
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, out, 0644)
}

// saveLanguageByTextReplace is a fallback when YAML parsing fails.
// It does a simple text replacement of the language line.
func saveLanguageByTextReplace(configPath, content, language string) error {
	lines := strings.Split(content, "\n")
	found := false
	inDefaults := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "defaults:" {
			inDefaults = true
			continue
		}
		if inDefaults && strings.HasPrefix(trimmed, "language:") {
			if language == "" {
				// Remove the line.
				lines = append(lines[:i], lines[i+1:]...)
			} else {
				lines[i] = fmt.Sprintf("  language: %s", language)
			}
			found = true
			break
		}
		// Exit defaults section if we hit a non-indented line.
		if inDefaults && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '#' {
			inDefaults = false
		}
	}

	if !found && language != "" {
		// Add language field after "defaults:" line.
		for i, line := range lines {
			if strings.TrimSpace(line) == "defaults:" {
				// Insert after the defaults: line.
				newLine := fmt.Sprintf("  language: %s", language)
				lines = append(lines[:i+1], append([]string{newLine}, lines[i+1:]...)...)
				break
			}
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
}

// stringToPtr returns a pointer to s, or nil if s is empty.
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// handleUpdateCommand shows the user how to update openconvene.
// It detects the current OS and prints the appropriate install command
// for the user to copy-paste into their shell after exiting the REPL.
func handleUpdateCommand() {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│                       Update OpenConveneCLI                 │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Printf("  Current version: %s\n", version.Version)
	fmt.Println()
	fmt.Println("  Please exit the REPL first (type /exit), then run the")
	fmt.Println("  command below in your shell:")
	fmt.Println()

	switch runtime.GOOS {
	case "windows":
		fmt.Println("  PowerShell:")
		fmt.Println()
		fmt.Println("    irm https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.ps1 | iex")
		fmt.Println()
		fmt.Println("  Or with winget (if published):")
		fmt.Println()
		fmt.Println("    winget install openconvene")
	case "darwin", "linux":
		fmt.Println("  Bash / Zsh:")
		fmt.Println()
		fmt.Println("    curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash")
		fmt.Println()
		fmt.Println("  Or with Go (if installed):")
		fmt.Println()
		fmt.Println("    go install github.com/masteryee-labs/open-convene-cli/cmd/openconvene@latest")
	default:
		fmt.Println("  curl -fsSL https://raw.githubusercontent.com/masteryee-labs/open-convene-cli/main/install.sh | bash")
	}

	fmt.Println()
	fmt.Println("  After installation, restart openconvene to use the new version.")
}
