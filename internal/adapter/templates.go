// templates.go — Built-in CLI command templates for dynamic model resolution.
//
// When a user specifies a model name in the format "CLI-模型名" (e.g.
// "agy-Gemini 3.5 Flash (High)"), the system auto-generates a ModelConfig
// from these built-in templates. This eliminates the need to manually define
// every model in models.yaml — the user can directly use model names from
// `openconvene models-info` output.
//
// Template placeholders:
//   {model}       — replaced with the model name (auto-quoted)
//   {prompt}      — replaced with the shell-escaped prompt at runtime
//   {prompt_file} — replaced with a temp file path containing the prompt
//
// Templates use {prompt_file} by default (more reliable for multi-line
// prompts). CLIs that don't support --prompt-file use {prompt} instead.

package adapter

import "github.com/masteryee-labs/open-convene-cli/internal/config"

// CLITemplate defines the command templates and capabilities for a CLI.
type CLITemplate struct {
	// RespondCmd is the command template for read-only (respond) mode.
	// Empty if the CLI doesn't support respond (e.g. aider).
	RespondCmd string

	// ExecuteCmd is the command template for agentic (execute) mode.
	ExecuteCmd string

	// ReadOnly is the CLI's read-only capability: "true", "false", or "maybe".
	ReadOnly string

	// ExecutorCapable indicates whether the CLI can act as an executor.
	ExecutorCapable bool

	// DefaultModel is the CLI's default model name (empty if unknown).
	DefaultModel string
}

// builtinCLITemplates maps CLI names to their built-in command templates.
//
// These templates are based on official CLI documentation and live testing.
// They use {model} for the model name, and {prompt_file} or {prompt} for the
// prompt placeholder.
//
// When a user specifies "devin-glm-5.2", the system:
//  1. Parses CLI name = "devin", model = "glm-5.2"
//  2. Looks up builtinCLITemplates["devin"]
//  3. Replaces {model} with "glm-5.2" in the template
//  4. Generates a ModelConfig with the final command string
var builtinCLITemplates = map[string]CLITemplate{
	"devin": {
		RespondCmd:      `devin --model "{model}" --print --prompt-file {prompt_file}`,
		ExecuteCmd:      `devin --model "{model}" --permission-mode dangerous --prompt-file {prompt_file}`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"grok": {
		RespondCmd:      `grok --model "{model}" --prompt-file {prompt_file}`,
		ExecuteCmd:      `grok --model "{model}" --always-approve --prompt-file {prompt_file}`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
		DefaultModel:    "grok-4.5",
	},
	"codex": {
		// codex exec reads from stdin; use type to pipe the prompt file.
		// -m flag sets the model. No quotes around {model} because:
		// 1) codex model names don't contain spaces, and
		// 2) this command uses cmd /c (due to pipe), and Go exec escapes "
		//    as \" which cmd.exe doesn't understand.
		RespondCmd:      `type {prompt_file} | codex exec -m {model} --sandbox read-only --skip-git-repo-check`,
		ExecuteCmd:      `type {prompt_file} | codex exec -m {model} --sandbox workspace-write --skip-git-repo-check`,
		ReadOnly:        "true",
		ExecutorCapable: true,
		DefaultModel:    "gpt-5.5",
	},
	"agy": {
		// agy doesn't support --prompt-file or stdin; use inline {prompt}.
		// parseArgs handles the quoting correctly for multi-word model names.
		RespondCmd:      `agy --model "{model}" --print "{prompt}"`,
		ExecuteCmd:      `agy --model "{model}" --dangerously-skip-permissions --print "{prompt}"`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"cursor": {
		RespondCmd:      `cursor agent --model "{model}" -p "{prompt}"`,
		ExecuteCmd:      `cursor agent --model "{model}" --force "{prompt}"`,
		ReadOnly:        "true",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"kimi": {
		RespondCmd:      `kimi --model "{model}" -p "{prompt}"`,
		ExecuteCmd:      `kimi --model "{model}" "{prompt}"`,
		ReadOnly:        "true",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"hermes": {
		// hermes may not support --model flag; model selection is via config.
		RespondCmd:      `hermes chat -q "{prompt}"`,
		ExecuteCmd:      `hermes agent "{prompt}"`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"aider": {
		// aider doesn't support respond mode (it modifies files by default).
		RespondCmd:      ``,
		ExecuteCmd:      `aider --yes --model "{model}" "{prompt}"`,
		ReadOnly:        "false",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
	"opencode": {
		RespondCmd:      `opencode run "{prompt}"`,
		ExecuteCmd:      `opencode run "{prompt}"`,
		ReadOnly:        "maybe",
		ExecutorCapable: true,
		DefaultModel:    "",
	},
}

// knownCLINames is the list of 9 supported CLI names, used for parsing
// dynamic model names (CLI:模型名).
var knownCLINames = []string{
	"devin", "grok", "codex", "agy", "cursor",
	"kimi", "hermes", "aider", "opencode",
}

// dynamicModelSeparator is the character that separates the CLI name from the
// model name in a dynamic model name string.
//
// We use ":" instead of "-" because model names frequently contain hyphens
// (e.g. "glm-5.2", "grok-4.5"), which would make parsing ambiguous. ":" never
// appears in CLI names or model names, so the split is always unambiguous.
const dynamicModelSeparator = ":"

// ResolveDynamicModel parses a dynamic model name in the format "CLI:模型名"
// and generates a ModelConfig from the built-in CLI template.
//
// Format: "CLI名稱:模型名稱"
//   - CLI名稱 must be one of the 9 known CLIs (devin, grok, codex, agy, etc.)
//   - 模型名稱 is everything after the separator (may contain hyphens,
//     spaces, parentheses — e.g. "Gemini 3.5 Flash (High)", "glm-5.2")
//
// Examples:
//   "agy:Gemini 3.5 Flash (High)" → CLI=agy, model="Gemini 3.5 Flash (High)"
//   "devin:glm-5.2"               → CLI=devin, model="glm-5.2"
//   "grok:grok-4.5"               → CLI=grok, model="grok-4.5"
//   "codex:gpt-5.5"               → CLI=codex, model="gpt-5.5"
//
// Returns:
//   - (ModelConfig, cliName, true) if the name matches a known CLI template
//   - (zero, "", false) if the name doesn't match any known CLI prefix
//
// The returned ModelConfig has Command and ExecuteCommand already substituted
// with the model name. The {prompt} / {prompt_file} placeholders remain for
// the adapter to fill at runtime.
func ResolveDynamicModel(name string) (config.ModelConfig, string, bool) {
	for _, cliName := range knownCLINames {
		// Check if name starts with "cliName:"
		prefix := cliName + dynamicModelSeparator
		if len(name) > len(prefix) && name[:len(prefix)] == prefix {
			modelName := name[len(prefix):]
			if modelName == "" {
				continue
			}

			tmpl, ok := builtinCLITemplates[cliName]
			if !ok {
				continue
			}

			// Replace {model} in the templates with the actual model name.
			// The model name is already quoted in the template ("{model}"),
			// so no additional escaping is needed.
			respondCmd := replaceAll(tmpl.RespondCmd, "{model}", modelName)
			executeCmd := replaceAll(tmpl.ExecuteCmd, "{model}", modelName)

			cfg := config.ModelConfig{
				Command:         respondCmd,
				ExecuteCommand:  executeCmd,
				ReadOnly:        tmpl.ReadOnly,
				Timeout:         0, // will inherit defaults
				ExecutorCapable: tmpl.ExecutorCapable,
			}

			return cfg, cliName, true
		}
	}

	return config.ModelConfig{}, "", false
}

// replaceAll is a simple string replacement helper (avoids importing strings
// in this file for a single call site).
func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			return result + s
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
}

// indexOf returns the index of the first occurrence of substr in s, or -1.
func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// IsDynamicModelName returns true if the name looks like a dynamic model name
// (i.e., starts with a known CLI name followed by a hyphen).
//
// This is used by config validation to accept dynamic model names in
// defaults.responders/executor/synthesizer without requiring them to be
// defined in the models section.
func IsDynamicModelName(name string) bool {
	_, _, ok := ResolveDynamicModel(name)
	return ok
}
