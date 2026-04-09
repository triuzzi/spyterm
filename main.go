// spyterm reads terminal pane contents from iTerm2 via AppleScript.
//
// Designed for use with Claude Code and other AI coding assistants:
// run dev servers in split panes, and the assistant reads errors directly.
//
// Install: go install github.com/triuzzi/spyterm@latest
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		cmdSiblings(80)
		return
	}

	switch os.Args[1] {
	case "siblings", "s":
		lineCount := intArg(2, 80)
		cmdSiblings(lineCount)
	case "list", "ls":
		verbose := len(os.Args) > 2 && (os.Args[2] == "-v" || os.Args[2] == "--verbose")
		cmdList(verbose)
	case "read", "r":
		cmdRead()
	case "send":
		cmdSend()
	case "all", "a":
		lineCount := intArg(2, 30)
		cmdAll(lineCount)
	case "version", "--version", "-v":
		fmt.Println("spyterm", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdSiblings(lines int) {
	tty, err := findTTY()
	if err != nil {
		fatal(err)
	}
	panes, err := siblings(tty)
	if err != nil {
		fatal(err)
	}
	if len(panes) == 0 {
		fmt.Println("No sibling panes found — this tab has only 1 pane.")
		return
	}
	for _, pane := range panes {
		fmt.Printf("=== %s (tty: %s) ===\n", pane.Label(), pane.TTY)
		fmt.Println(tailLines(pane.Contents, lines))
		fmt.Println()
	}
}

func cmdList(verbose bool) {
	panes, err := listPanes()
	if err != nil {
		fatal(err)
	}
	if len(panes) == 0 {
		fmt.Println("No panes found.")
		return
	}

	// Group panes: window -> tab -> panes
	type tabInfo struct {
		tab   int
		panes []Pane
	}
	type windowInfo struct {
		id   int
		name string
		tabs []tabInfo
	}

	windowOrder := []int{}
	windowMap := map[int]*windowInfo{}
	tabSeen := map[string]bool{}

	for _, pane := range panes {
		window, ok := windowMap[pane.WindowID]
		if !ok {
			window = &windowInfo{id: pane.WindowID, name: pane.WindowName}
			windowMap[pane.WindowID] = window
			windowOrder = append(windowOrder, pane.WindowID)
		}
		key := fmt.Sprintf("%d-%d", pane.WindowID, pane.Tab)
		if !tabSeen[key] {
			tabSeen[key] = true
			window.tabs = append(window.tabs, tabInfo{tab: pane.Tab})
		}
		// Append pane to the last tab entry
		tab := &window.tabs[len(window.tabs)-1]
		tab.panes = append(tab.panes, pane)
	}

	totalWindows := len(windowOrder)
	totalTabs := 0
	totalPanes := len(panes)
	for _, windowID := range windowOrder {
		totalTabs += len(windowMap[windowID].tabs)
	}

	if verbose {
		// Verbose: print each pane with content, grouped by window/tab
		for windowIndex, windowID := range windowOrder {
			window := windowMap[windowID]
			if windowIndex > 0 {
				fmt.Println()
			}
			fmt.Printf("\033[1m▌ W%d  %s\033[0m\n", window.id, window.name)
			for _, tab := range window.tabs {
				for _, pane := range tab.panes {
					fmt.Printf("\n  \033[1m── T%d P%d  %s\033[0m\n\n", tab.tab, pane.Index, pane.Name)
					content := tailLines(pane.Contents, 20)
					for _, line := range strings.Split(content, "\n") {
						fmt.Printf("  %s\n", line)
					}
				}
			}
		}
		fmt.Printf("\n%d windows, %d tabs, %d panes\n", totalWindows, totalTabs, totalPanes)
	} else {
		// Compact: tree view
		for windowIndex, windowID := range windowOrder {
			window := windowMap[windowID]
			lastWindow := windowIndex == len(windowOrder)-1
			windowPrefix := "├── "
			windowContinuation := "│   "
			if lastWindow {
				windowPrefix = "└── "
				windowContinuation = "    "
			}
			fmt.Printf("%s\033[1mW%-6d %s\033[0m\n", windowPrefix, window.id, window.name)
			for tabIndex, tab := range window.tabs {
				lastTab := tabIndex == len(window.tabs)-1
				tabPrefix := windowContinuation + "├── "
				tabContinuation := windowContinuation + "│   "
				if lastTab {
					tabPrefix = windowContinuation + "└── "
					tabContinuation = windowContinuation + "    "
				}
				fmt.Printf("%sT%d\n", tabPrefix, tab.tab)
				for paneIndex, pane := range tab.panes {
					lastPane := paneIndex == len(tab.panes)-1
					panePrefix := tabContinuation + "├── "
					if lastPane {
						panePrefix = tabContinuation + "└── "
					}
					fmt.Printf("%sP%-6d %s\n", panePrefix, pane.Index, pane.Name)
				}
			}
		}
		fmt.Printf("\n%d windows, %d tabs, %d panes\n", totalWindows, totalTabs, totalPanes)
	}
}

func cmdRead() {
	// Parse: read [WINDOW_ID] TAB PANE [LINES]
	// Accepts W1234/1234, T4/4, P10/10 prefixed or plain numeric forms.
	args := os.Args[2:]
	var windowID, tab, pane, lines int

	switch {
	case len(args) >= 3:
		firstArg := mustID(args[0], "window/tab")
		// Window IDs are large numbers (>1000); tab indices are small.
		// A W/w prefix also signals a window ID regardless of value.
		hasWindowPrefix := len(args[0]) > 1 && (args[0][0] == 'W' || args[0][0] == 'w')
		if firstArg > 100 || hasWindowPrefix {
			windowID = firstArg
			tab = mustID(args[1], "tab")
			pane = mustID(args[2], "pane")
			lines = intArgFromSlice(args, 3, 50)
		} else {
			tab = firstArg
			pane = mustID(args[1], "pane")
			lines = intArgFromSlice(args, 2, 50)
		}
	case len(args) == 2:
		tab = mustID(args[0], "tab")
		pane = mustID(args[1], "pane")
		lines = 50
	default:
		fmt.Fprintln(os.Stderr, "usage: spyterm read [W<id>] <T>tab <P>pane [lines]")
		os.Exit(1)
	}

	result, err := readPane(windowID, tab, pane)
	if err != nil {
		fatal(err)
	}
	fmt.Println(tailLines(result.Contents, lines))
}

func cmdSend() {
	// Parse: send [--keys] [WINDOW_ID] TAB PANE COMMAND...
	// Accepts prefixed IDs: W1234, T4, P2
	args := os.Args[2:]

	// Check for --keys flag
	keysMode := false
	if len(args) > 0 && args[0] == "--keys" {
		keysMode = true
		args = args[1:]
	}

	var windowID, tab, pane int
	var commandStart int

	switch {
	case len(args) >= 4:
		firstArg := mustID(args[0], "window/tab")
		hasWindowPrefix := len(args[0]) > 1 && (args[0][0] == 'W' || args[0][0] == 'w')
		if firstArg > 100 || hasWindowPrefix {
			windowID = firstArg
			tab = mustID(args[1], "tab")
			pane = mustID(args[2], "pane")
			commandStart = 3
		} else {
			tab = firstArg
			pane = mustID(args[1], "pane")
			commandStart = 2
		}
	case len(args) >= 3:
		tab = mustID(args[0], "tab")
		pane = mustID(args[1], "pane")
		commandStart = 2
	default:
		fmt.Fprintln(os.Stderr, "usage: spyterm send [--keys] [W<id>] <T>tab <P>pane <command/keys...>")
		os.Exit(1)
	}

	remaining := args[commandStart:]
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "error: no command or keys specified")
		os.Exit(1)
	}

	// If no window ID, resolve from first matching tab+pane
	if windowID == 0 {
		target, err := readPane(0, tab, pane)
		if err != nil {
			fatal(err)
		}
		windowID = target.WindowID
	}

	if keysMode {
		if err := sendKeysToPane(windowID, tab, pane, remaining); err != nil {
			fatal(err)
		}
		fmt.Printf("sent keys to W%d T%d P%d: %s\n", windowID, tab, pane, strings.Join(remaining, " "))
	} else {
		command := strings.Join(remaining, " ")
		if err := sendToPane(windowID, tab, pane, command); err != nil {
			fatal(err)
		}
		fmt.Printf("sent to W%d T%d P%d: %s\n", windowID, tab, pane, command)
	}
}

func cmdAll(lines int) {
	panes, err := listPanes()
	if err != nil {
		fatal(err)
	}
	for _, pane := range panes {
		fmt.Printf("=== %s (tty: %s) ===\n", pane.Label(), pane.TTY)
		fmt.Println(tailLines(pane.Contents, lines))
		fmt.Println()
	}
}

func printUsage() {
	fmt.Print(`spyterm — read iTerm2 split-pane contents from the terminal

Usage:
  spyterm                    Read sibling panes (default, 80 lines)
  spyterm siblings [N]       Read last N lines from sibling panes (same tab)
  spyterm list [-v]           Show all windows/tabs/panes (-v for content)
  spyterm read [W] T P [N]   Read pane (accepts W1234/1234, T4/4, P2/2)
  spyterm send [W] T P CMD   Send a command to a pane (text + Enter)
  spyterm send --keys T P K  Send raw keys (^C, ^D, ^Z, ^[, etc.)
  spyterm all [N]            Read last N lines from ALL panes
  spyterm version            Show version

Aliases: siblings=s, list=ls, read=r, all=a

Examples:
  spyterm                    # see what's in your split panes
  spyterm s 200              # last 200 lines from siblings
  spyterm read 6 3           # tab 6, pane 3, last 50 lines
  spyterm read 35267 6 3 100 # specific window, tab 6, pane 3, 100 lines
`)
}

// intArg returns os.Args[index] as int, or the default.
func intArg(index, defaultValue int) int {
	if index >= len(os.Args) {
		return defaultValue
	}
	value, err := strconv.Atoi(os.Args[index])
	if err != nil {
		return defaultValue
	}
	return value
}

func intArgFromSlice(args []string, index, defaultValue int) int {
	if index >= len(args) {
		return defaultValue
	}
	value, err := strconv.Atoi(args[index])
	if err != nil {
		return defaultValue
	}
	return value
}

// parseID strips an optional W/T/P prefix (case-insensitive) and returns the numeric value.
// Accepts "W1234", "w1234", "T4", "t4", "P10", "p10", or plain "1234".
func parseID(text string) (int, error) {
	if len(text) > 1 {
		prefix := text[0]
		if prefix == 'W' || prefix == 'w' || prefix == 'T' || prefix == 't' || prefix == 'P' || prefix == 'p' {
			text = text[1:]
		}
	}
	return strconv.Atoi(text)
}

func mustID(text, name string) int {
	value, err := parseID(text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s: %s\n", name, text)
		os.Exit(1)
	}
	return value
}

func fatal(err error) {
	message := err.Error()
	if strings.Contains(message, "not running") || strings.Contains(message, "Connection is invalid") {
		fmt.Fprintln(os.Stderr, "error: iTerm2 is not running")
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", message)
	}
	os.Exit(1)
}
