// menu.go — Interactive decision menu for the openconvene REPL.
//
// This file implements a Devin-style interactive selection menu that appears
// when the AI needs human input for a decision. The menu supports:
//   - Numbered options with titles and descriptions
//   - Up/down arrow key navigation
//   - Enter to select, Esc to cancel
//   - Number keys (1-9) for quick selection
//   - "e" to select and type additional text
//   - Navigation hints at the bottom (Devin-style)
//
// The menu is displayed using raw terminal mode for key-by-key input,
// with ANSI escape sequences for cursor movement and screen clearing.

package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// Menu option types
// ---------------------------------------------------------------------------

// MenuOption represents a single selectable option in the decision menu.
type MenuOption struct {
	// Title is the short label shown for the option (e.g. "生成全部 Prompt 檔案").
	Title string

	// Description is the longer explanation shown below the title.
	Description string
}

// MenuResult contains the result of a menu interaction.
type MenuResult struct {
	// Selected is the index of the chosen option, or -1 if cancelled.
	Selected int

	// CustomText is additional text typed by the user (when using "e" to
	// select+type). Empty if the user just pressed Enter.
	CustomText string

	// Cancelled is true if the user pressed Esc.
	Cancelled bool
}

// ---------------------------------------------------------------------------
// Menu display
// ---------------------------------------------------------------------------

// displayMenu renders the interactive menu and handles keyboard input.
// It returns the user's selection.
//
// The menu looks like:
//
//	── Prompt 檔案缺失 ────────────────────────────────────────────────────────
//	  重構管線的 11 個 Session Prompt 檔案 + 5 個 Shared_State 檔案全部不存在...
//
//	❭ 1 生成全部 Prompt 檔案
//	    由我（指揮官）根據 README.md 的規格描述...
//	  2 我手動提供檔案
//	    你會自己建立這些檔案...
//	  3 暫停，讓我檢查
//	    暫停指揮官流程...
//	  4 只生成 R0 先開始
//	    先生成 R0_Foundation.md...
//	    Other (type your own)
//	────────────────────────────────────────────────────────────────────────────
//	↑↓ navigate · ↵ select · e select+type · ? help me out · esc cancel
//
// Navigation:
//   - Up/Down arrows: move selection
//   - Enter: select current option
//   - 1-9: quick select by number
//   - e: select current option and type additional text
//   - Esc: cancel
func displayMenu(title, summary string, options []MenuOption) MenuResult {
	// If stdin is not a terminal, fall back to simple numbered input.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return displayMenuFallback(title, summary, options)
	}

	// Enable raw terminal mode for key-by-key input.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return displayMenuFallback(title, summary, options)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	selected := 0
	totalOptions := len(options)

	for {
		// Render the menu.
		renderMenu(title, summary, options, selected)

		// Read a single key.
		key := readKey()

		switch key {
		case keyUp:
			if selected > 0 {
				selected--
			}
		case keyDown:
			if selected < totalOptions-1 {
				selected++
			}
		case keyEnter:
			// Clear the menu before returning.
			clearMenu(totalOptions, summary)
			return MenuResult{Selected: selected}
		case keyEsc:
			clearMenu(totalOptions, summary)
			return MenuResult{Selected: -1, Cancelled: true}
		case keyHelp:
			// Show help text temporarily.
			renderHelp()
		default:
			// Check for number keys (1-9).
			if key >= '1' && key <= '9' {
				idx := int(key - '1')
				if idx < totalOptions {
					selected = idx
					clearMenu(totalOptions, summary)
					return MenuResult{Selected: selected}
				}
			}
			// Check for 'e' (select + type).
			if key == 'e' || key == 'E' {
				clearMenu(totalOptions, summary)
				// Restore terminal for text input.
				term.Restore(int(os.Stdin.Fd()), oldState)
				customText := readCustomText(options[selected].Title)
				return MenuResult{Selected: selected, CustomText: customText}
			}
		}
	}
}

// renderMenu draws the complete menu to the terminal.
func renderMenu(title, summary string, options []MenuOption, selected int) {
	// Move cursor to the beginning and clear everything below.
	fmt.Print("\r\x1b[2K\x1b[0J")

	// Title line with separator.
	titleLine := fmt.Sprintf("── %s ", title)
	remaining := 80 - len(titleLine) - 1
	if remaining < 3 {
		remaining = 3
	}
	fmt.Printf("%s%s\n", titleLine, strings.Repeat("─", remaining))

	// Summary (if provided).
	if summary != "" {
		fmt.Printf("  %s\n", summary)
	}
	fmt.Println()

	// Options.
	for i, opt := range options {
		if i == selected {
			fmt.Printf("  ❭ \x1b[1m%d %s\x1b[0m\n", i+1, opt.Title)
		} else {
			fmt.Printf("    %d %s\n", i+1, opt.Title)
		}
		// Description (indented).
		if opt.Description != "" {
			if i == selected {
				fmt.Printf("      \x1b[2m%s\x1b[0m\n", opt.Description)
			} else {
				fmt.Printf("      \x1b[2m%s\x1b[0m\n", opt.Description)
			}
		}
	}

	// "Other" option.
	if selected == len(options) {
		fmt.Printf("  ❭ \x1b[1mOther (type your own)\x1b[0m\n")
	} else {
		fmt.Println("    Other (type your own)")
	}

	// Bottom separator.
	fmt.Println(strings.Repeat("─", 80))

	// Navigation hints.
	fmt.Printf("\x1b[2m↑↓ navigate · ↵ select · e select+type · ? help me out · esc cancel\x1b[0m")
}

// renderHelp shows a temporary help message.
func renderHelp() {
	// Clear the hints line and show help.
	fmt.Printf("\r\x1b[2K")
	fmt.Printf("\x1b[2mPress ↑↓ to navigate, ↵ to select, e to select+type, 1-9 for quick select, esc to cancel\x1b[0m")
	// Wait for any key to continue.
	readKey()
}

// clearMenu erases the menu from the terminal.
func clearMenu(totalOptions int, summary string) {
	// Calculate total lines to clear: title(1) + summary(1) + blank(1) +
	// options (each option = 2 lines: title + description) + other(1) +
	// separator(1) + hints(1).
	linesToClear := 1 // title
	if summary != "" {
		linesToClear++ // summary
	}
	linesToClear++ // blank line
	linesToClear += totalOptions * 2
	linesToClear++ // "Other" line
	linesToClear++ // separator
	linesToClear++ // hints

	// Move up and clear each line.
	for i := 0; i < linesToClear; i++ {
		fmt.Print("\r\x1b[1A\x1b[2K")
	}
	fmt.Print("\r")
}

// readCustomText reads a line of text input after the user selects an option
// with 'e'. The terminal is in cooked mode at this point.
func readCustomText(optionTitle string) string {
	fmt.Printf("  Selected: %s\n", optionTitle)
	fmt.Printf("  Additional context (press Enter to skip): ")
	var text string
	fmt.Scanln(&text)
	return text
}

// ---------------------------------------------------------------------------
// Fallback (non-terminal)
// ---------------------------------------------------------------------------

// displayMenuFallback is used when stdin is not a terminal (e.g. piped input).
// It prints the menu and reads a number from stdin.
func displayMenuFallback(title, summary string, options []MenuOption) MenuResult {
	fmt.Printf("── %s ──\n", title)
	if summary != "" {
		fmt.Printf("  %s\n", summary)
	}
	fmt.Println()

	for i, opt := range options {
		fmt.Printf("  %d. %s\n", i+1, opt.Title)
		if opt.Description != "" {
			fmt.Printf("     %s\n", opt.Description)
		}
	}
	fmt.Println("    Other (type your own)")
	fmt.Println()

	fmt.Print("Enter choice number (or type your own): ")
	var input string
	fmt.Scanln(&input)

	// Try to parse as a number.
	if len(input) > 0 && input[0] >= '1' && input[0] <= '9' {
		idx := int(input[0] - '1')
		if idx < len(options) {
			return MenuResult{Selected: idx, CustomText: input[1:]}
		}
	}

	// Not a number — treat as custom text.
	return MenuResult{Selected: -1, CustomText: input, Cancelled: input == ""}
}

// ---------------------------------------------------------------------------
// Key reading
// ---------------------------------------------------------------------------

// Key constants for arrow keys and special keys.
const (
	keyUp    = 'A' + 1000 // Sent as ESC [ A
	keyDown  = 'B' + 1000
	keyEnter = '\r'
	keyEsc   = 27
	keyHelp  = '?'
)

// readKey reads a single keypress from stdin in raw mode.
// It handles escape sequences for arrow keys.
func readKey() rune {
	buf := make([]byte, 3)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return 0
	}

	// Single key.
	if n == 1 {
		return rune(buf[0])
	}

	// Escape sequence (ESC [ X).
	if n >= 3 && buf[0] == 27 && buf[1] == 91 {
		switch buf[2] {
		case 'A':
			return keyUp
		case 'B':
			return keyDown
		}
	}

	// Two-byte escape sequence.
	if n == 2 && buf[0] == 27 {
		return keyEsc
	}

	return rune(buf[0])
}
