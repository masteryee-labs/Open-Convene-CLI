// decision.go — AI output marker parsing and decision menu integration.
//
// This file implements:
//   - Parsing of [[DECISION]]...[[/DECISION]] markers from AI responses
//   - The /choose slash command for manual decision menu invocation
//   - Integration with the interactive menu (menu.go) to present choices
//     to the user and feed their selection back into the Convene pipeline.
//
// Marker format (parsed from AI output):
//
//	[[DECISION]]
//	title: Prompt 檔案缺失
//	summary: 重構管線的 11 個 Session Prompt 檔案全部不存在...
//	option: 生成全部 Prompt 檔案 | 由我（指揮官）根據 README.md 的規格描述...
//	option: 我手動提供檔案 | 你會自己建立這些檔案...
//	[[/DECISION]]
//
// When the CLI detects this marker in an AI response, it:
//  1. Strips the marker block from the displayed output
//  2. Shows the interactive decision menu (menu.go)
//  3. Feeds the user's choice back as a new prompt: "使用者選擇了: 選項 N — <title>"
//  4. If the user typed additional context, appends it to the follow-up prompt.

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// ParsedDecision represents a decision block extracted from AI output.
type ParsedDecision struct {
	// Title is the decision prompt title.
	Title string

	// Summary is the optional context/description shown above the options.
	Summary string

	// Options is the list of selectable options.
	Options []MenuOption
}

// ---------------------------------------------------------------------------
// Marker parsing
// ---------------------------------------------------------------------------

// decisionMarkerRe matches the [[DECISION]]...[[/DECISION]] block.
var decisionMarkerRe = regexp.MustCompile(`(?s)\[\[DECISION\]\]\s*\n(.*?)\n\s*\[\[/DECISION\]\]`)

// ParseDecision scans text for a [[DECISION]]...[[/DECISION]] marker block
// and returns the parsed decision plus the text with the block removed.
// If no marker is found, returns nil and the original text.
func ParseDecision(text string) (*ParsedDecision, string) {
	match := decisionMarkerRe.FindStringSubmatch(text)
	if match == nil {
		return nil, text
	}

	body := match[1]
	decision := parseDecisionBody(body)
	if decision == nil || len(decision.Options) == 0 {
		return nil, text
	}

	// Remove the marker block from the text.
	cleanedText := decisionMarkerRe.ReplaceAllString(text, "")
	// Clean up any leftover blank lines.
	cleanedText = strings.TrimSpace(cleanedText)

	return decision, cleanedText
}

// parseDecisionBody parses the inner content of a [[DECISION]] block.
// Expected format:
//
//	title: <title text>
//	summary: <summary text>
//	option: <title> | <description>
//	option: <title> | <description>
func parseDecisionBody(body string) *ParsedDecision {
	decision := &ParsedDecision{}

	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "key: value" format.
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "title":
			decision.Title = value
		case "summary":
			decision.Summary = value
		case "option":
			opt := parseOptionLine(value)
			decision.Options = append(decision.Options, opt)
		}
	}

	if decision.Title == "" {
		decision.Title = "需要決策"
	}

	return decision
}

// parseOptionLine parses an option line in "title | description" format.
// If no pipe is found, the entire line is the title with no description.
func parseOptionLine(line string) MenuOption {
	pipeIdx := strings.Index(line, "|")
	if pipeIdx == -1 {
		return MenuOption{Title: strings.TrimSpace(line)}
	}
	return MenuOption{
		Title:       strings.TrimSpace(line[:pipeIdx]),
		Description: strings.TrimSpace(line[pipeIdx+1:]),
	}
}

// ---------------------------------------------------------------------------
// Decision menu invocation
// ---------------------------------------------------------------------------

// presentDecision displays the interactive decision menu and returns the
// user's choice as a follow-up prompt string.
//
// If the user cancels (Esc), returns an empty string.
// If the user selects an option, returns a formatted follow-up prompt like:
//
//	使用者選擇了: 選項 1 — 生成全部 Prompt 檔案
//	(使用者補充: <custom text>)  // only if custom text was provided
func presentDecision(decision *ParsedDecision) string {
	result := displayMenu(decision.Title, decision.Summary, decision.Options)

	if result.Cancelled {
		fmt.Println("\n  (決策已取消)")
		return ""
	}

	if result.Selected < 0 || result.Selected >= len(decision.Options) {
		// "Other" was selected or invalid — use custom text.
		if result.CustomText != "" {
			return fmt.Sprintf("使用者選擇了: 自訂輸入 — %s", result.CustomText)
		}
		fmt.Println("\n  (決策已取消)")
		return ""
	}

	opt := decision.Options[result.Selected]
	followUp := fmt.Sprintf("使用者選擇了: 選項 %d — %s", result.Selected+1, opt.Title)
	if result.CustomText != "" {
		followUp += fmt.Sprintf("\n(使用者補充: %s)", result.CustomText)
	}
	return followUp
}

// ---------------------------------------------------------------------------
// /choose slash command
// ---------------------------------------------------------------------------

// handleChooseCommand handles the /choose slash command.
// It allows the user to manually invoke the decision menu.
//
// Usage:
//
//	/choose <title> | <option1> | <option2> | ...
//	/choose "Which approach?" "Do A" "Do B" "Do C"
//
// The simplest form treats each pipe-separated value as an option:
//
//	/choose Do A | Do B | Do C
func handleChooseCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: /choose <title> | <option1> | <option2> | ...")
		fmt.Println("  Or:   /choose \"Question?\" \"Option 1\" \"Option 2\"")
		fmt.Println()
		fmt.Println("Shows an interactive menu for decision-making.")
		fmt.Println("Use ↑↓ to navigate, ↵ to select, e to select+type, esc to cancel.")
		return
	}

	// Join all args back into a single string (they were split by spaces).
	raw := strings.Join(args, " ")

	// Try pipe-separated format first: "title | opt1 | opt2 | ..."
	if strings.Contains(raw, "|") {
		parts := strings.Split(raw, "|")
		if len(parts) >= 2 {
			title := strings.TrimSpace(parts[0])
			var options []MenuOption
			for _, p := range parts[1:] {
				p = strings.TrimSpace(p)
				if p != "" {
					options = append(options, MenuOption{Title: p})
				}
			}
			if len(options) > 0 {
				decision := &ParsedDecision{Title: title, Options: options}
				followUp := presentDecision(decision)
				if followUp != "" {
					fmt.Printf("\n  → %s\n", followUp)
				}
				return
			}
		}
	}

	// Fall back: treat each arg as an option, no title.
	var options []MenuOption
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a != "" {
			options = append(options, MenuOption{Title: a})
		}
	}
	if len(options) == 0 {
		fmt.Println("No options provided.")
		return
	}

	decision := &ParsedDecision{Title: "選擇一個選項", Options: options}
	followUp := presentDecision(decision)
	if followUp != "" {
		fmt.Printf("\n  → %s\n", followUp)
	}
}
