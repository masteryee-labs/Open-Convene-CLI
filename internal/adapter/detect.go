// detect.go — CLI detection for OpenConveneCLI.
//
// DetectAvailableAdapters checks which of the 9 supported coding-agent CLIs
// are installed on the system (via exec.LookPath) and returns detailed
// information about each, including install commands for those not found.
//
// This is used by the `openconvene detect` command (S4) to show the user
// which CLIs are available. It does NOT auto-install any CLI — install
// commands are displayed for reference only.
//
// exec.LookPath is cross-platform: on Windows it searches for .exe/.cmd/.bat
// extensions; on Linux/macOS it searches PATH directories.

package adapter

import (
	"context"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// DetectResult holds the detection result for a single CLI.
//
// S4's `detect` command formats this into a user-facing table.
type DetectResult struct {
	// Name is the CLI detection name (e.g. "devin", "grok", "codex").
	Name string

	// Found is true if the CLI was found in PATH via exec.LookPath.
	Found bool

	// Path is the full path to the CLI executable (empty if not found).
	Path string

	// ReadOnly indicates the CLI's read-only capability:
	// "true"  — enforces read-only mode (safe for responders)
	// "false" — modifies files by default (aider)
	// "maybe" — has non-interactive mode but inherently agentic
	ReadOnly string

	// CanRespond indicates whether the CLI is suitable as a responder.
	// Aider (read_only=false) is false; all others are true.
	CanRespond bool

	// CanExecute indicates whether the CLI is suitable as an executor.
	// All 9 CLIs are executor-capable.
	CanExecute bool

	// InstallCmd is the installation command for this CLI (display only;
	// OpenConveneCLI never auto-installs).
	InstallCmd string

	// HeadlessCmd is an example non-interactive mode command with {prompt}
	// placeholder, shown for reference.
	HeadlessCmd string
}

// knownCLIInfo describes a single known CLI's capabilities.
type knownCLIInfo struct {
	ReadOnly    string
	CanRespond  bool
	CanExecute  bool
	InstallCmd  string
	HeadlessCmd string
}

// knownCLIs maps CLI detection names to their capability info.
//
// This is hardcoded — the set of supported CLIs and their capability matrix
// are fixed and rarely change. Values are based on official documentation
// research (Docs/03-Model-Adapters.md) and live CLI verification where
// available (S2).
var knownCLIs = map[string]knownCLIInfo{
	"devin": {
		ReadOnly:    "maybe",
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `curl -fsSL https://cli.devin.ai/install.sh | bash`,
		HeadlessCmd: `devin -p "{prompt}"`,
	},
	"grok": {
		ReadOnly:    "maybe",
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `curl -fsSL https://x.ai/cli/install.sh | bash`,
		HeadlessCmd: `grok -p "{prompt}"`,
	},
	"codex": {
		ReadOnly:    "true", // --sandbox read-only enforces read-only
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `npm install -g @openai/codex`,
		HeadlessCmd: `codex exec --sandbox read-only "{prompt}"`,
	},
	"agy": {
		ReadOnly:    "maybe",
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `curl -fsSL https://antigravity.google/cli/install.sh | bash`,
		HeadlessCmd: `agy -p "{prompt}"`,
	},
	"cursor": {
		ReadOnly:    "true", // without --force, agent is read-only
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `curl https://cursor.com/install -fsS | bash`,
		HeadlessCmd: `cursor agent -p "{prompt}"`,
	},
	"kimi": {
		ReadOnly:    "true", // read-only ops auto-approved, no file modification
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `curl -fsSL https://code.kimi.com/kimi-code/install.sh | bash`,
		HeadlessCmd: `kimi -p "{prompt}"`,
	},
	"hermes": {
		ReadOnly:    "maybe",
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `hermes setup --portal  # see hermes-agent.nousresearch.com`,
		HeadlessCmd: `hermes chat -q "{prompt}"`,
	},
	"aider": {
		ReadOnly:    "false", // inherently a code editor, modifies files
		CanRespond:  false,   // not suitable as responder
		CanExecute:  true,
		InstallCmd:  `python -m pip install aider-install && aider-install`,
		HeadlessCmd: `aider --yes --model {model} "{prompt}"`,
	},
	"opencode": {
		ReadOnly:    "maybe",
		CanRespond:  true,
		CanExecute:  true,
		InstallCmd:  `# see https://opencode.ai/docs/cli/`,
		HeadlessCmd: `opencode run "{prompt}"`,
	},
}

// DetectAvailableAdapters detects which supported CLIs are installed on the
// system.
//
// Returns a slice of DetectResult for all 9 known CLIs, sorted by Name.
// Each result indicates whether the CLI was found (Found=true, with Path)
// or not found (Found=false, with InstallCmd for user reference).
//
// This function does NOT install any CLI. Install commands are for display
// only. The user must install CLIs manually.
func DetectAvailableAdapters() []DetectResult {
	// Collect and sort CLI names for stable output order.
	names := make([]string, 0, len(knownCLIs))
	for name := range knownCLIs {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]DetectResult, 0, len(names))
	for _, name := range names {
		info := knownCLIs[name]

		// exec.LookPath searches PATH for the executable.
		// On Windows, it also checks .exe/.cmd/.bat extensions.
		path, err := exec.LookPath(name)
		found := err == nil

		results = append(results, DetectResult{
			Name:        name,
			Found:       found,
			Path:        path,
			ReadOnly:    info.ReadOnly,
			CanRespond:  info.CanRespond,
			CanExecute:  info.CanExecute,
			InstallCmd:  info.InstallCmd,
			HeadlessCmd: info.HeadlessCmd,
		})
	}

	// Results are already sorted by Name (names were sorted before iteration).
	return results
}

// CLIModelInfo holds the result of querying a CLI for its available models.
type CLIModelInfo struct {
	// Name is the CLI name (e.g. "devin", "grok", "agy", "codex").
	Name string

	// Installed is true if the CLI was found in PATH.
	Installed bool

	// Models is the list of available model names (empty if query failed).
	Models []string

	// DefaultModel is the CLI's default model (empty if unknown).
	DefaultModel string

	// RawOutput is the raw output from the CLI's models command (for debugging).
	RawOutput string

	// Error is the error message if the query failed (empty on success).
	Error string
}

// knownModelHints provides known model names for CLIs that don't have a
// `models` subcommand. These are based on CLI documentation and may not
// be exhaustive.
var knownModelHints = map[string]struct {
	Models       []string
	DefaultModel string
}{
	"devin": {
		Models:       []string{"glm-5.2", "swe-1.7", "kimi-k2.7", "claude-sonnet-4", "claude-opus-4.6", "opus", "codex"},
		DefaultModel: "(account-dependent)",
	},
	"codex": {
		Models:       []string{"gpt-5.5 (default, ChatGPT account)", "o3 (requires API key)", "codex-5.6 (requires API key)"},
		DefaultModel: "gpt-5.5",
	},
}

// QueryCLIModels queries each installed CLI for its available models.
//
// For CLIs with a `models` subcommand (agy, grok), it executes the command
// and parses the output. For CLIs without one (devin, codex), it returns
// known model hints from documentation.
//
// Only installed CLIs are queried. The function has a 10-second timeout per
// CLI to prevent hanging on unresponsive commands.
func QueryCLIModels() []CLIModelInfo {
	detectResults := DetectAvailableAdapters()
	var infos []CLIModelInfo

	for _, dr := range detectResults {
		info := CLIModelInfo{
			Name:      dr.Name,
			Installed: dr.Found,
		}

		if !dr.Found {
			info.Error = "not installed"
			infos = append(infos, info)
			continue
		}

		// Try to query models via subcommand.
		switch dr.Name {
		case "agy", "grok":
			output, err := runCLIModelsCommand(dr.Name, "models")
			if err != nil {
				info.Error = err.Error()
				// Fall back to hints if available.
				if hints, ok := knownModelHints[dr.Name]; ok {
					info.Models = hints.Models
					info.DefaultModel = hints.DefaultModel
				}
			} else {
				info.RawOutput = output
				info.Models, info.DefaultModel = parseModelsOutput(dr.Name, output)
			}
		default:
			// Use known hints for CLIs without a models subcommand.
			if hints, ok := knownModelHints[dr.Name]; ok {
				info.Models = hints.Models
				info.DefaultModel = hints.DefaultModel
				info.RawOutput = "(no models subcommand — using known hints)"
			} else {
				info.Error = "no models subcommand available"
			}
		}

		infos = append(infos, info)
	}

	return infos
}

// runCLIModelsCommand executes `<cli> models` with a 10-second timeout.
func runCLIModelsCommand(cliName, subcommand string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cliName, subcommand)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

// parseModelsOutput parses the output of a CLI's `models` subcommand.
// It handles different output formats:
//   - agy: one model per line (e.g. "Gemini 3.5 Flash (High)")
//   - grok: "Default model: X" + "Available models:" + "  * X (default)"
func parseModelsOutput(cliName, output string) (models []string, defaultModel string) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	switch cliName {
	case "grok":
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Default model:") {
				defaultModel = strings.TrimSpace(strings.TrimPrefix(line, "Default model:"))
			}
			if strings.HasPrefix(line, "* ") {
				model := strings.TrimSpace(strings.TrimPrefix(line, "* "))
				model = strings.TrimSuffix(model, " (default)")
				models = append(models, model)
			} else if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "Available") {
				model := strings.TrimSpace(line)
				if model != "" && !strings.Contains(model, ":") {
					models = append(models, model)
				}
			}
		}
	default:
		// agy and others: one model per line.
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				models = append(models, line)
			}
		}
	}

	return models, defaultModel
}
