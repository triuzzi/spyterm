package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// runAppleScript executes an AppleScript via osascript and returns its stdout.
func runAppleScript(script string, args ...string) (string, error) {
	commandArgs := []string{"-"}
	commandArgs = append(commandArgs, args...)
	command := exec.Command("osascript", commandArgs...)
	command.Stdin = strings.NewReader(script)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AppleScript error: %s\n%s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

// Pane represents one iTerm2 session (split pane).
type Pane struct {
	WindowID   int
	WindowName string
	Tab        int
	Index      int // 1-based pane index within the tab
	Total      int // total panes in the tab
	TTY        string
	Name       string // session name (current process/directory)
	Contents   string
}

func (pane Pane) Label() string {
	return fmt.Sprintf("W%d T%d P%d/%d", pane.WindowID, pane.Tab, pane.Index, pane.Total)
}

const recordSep = "<<ITERM_PANE_RECORD>>"

const listScript = `
set sep to "<<ITERM_PANE_RECORD>>"
set delim to character id 9
tell application "iTerm2"
	set output to ""
	repeat with w in windows
		set wid to id of w
		set wname to name of w
		repeat with t from 1 to (count of tabs of w)
			set theTab to tab t of w
			set sc to count of sessions of theTab
			repeat with i from 1 to sc
				set s to session i of theTab
				set theTTY to tty of s as text
				set sname to name of s
				set output to output & sep & wid & delim & wname & delim & t & delim & i & delim & sc & delim & theTTY & delim & sname & linefeed & (contents of s)
			end repeat
		end repeat
	end repeat
	return output
end tell
`

func listPanes() ([]Pane, error) {
	output, err := runAppleScript(listScript)
	if err != nil {
		return nil, err
	}
	records := strings.Split(output, recordSep)
	var panes []Pane
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		// First line is metadata (tab-separated), rest is content
		newline := strings.IndexByte(record, '\n')
		if newline == -1 {
			continue
		}
		meta := record[:newline]
		content := record[newline+1:]
		parts := strings.SplitN(meta, "\t", 7)
		if len(parts) < 7 {
			continue
		}
		windowID, _ := strconv.Atoi(parts[0])
		windowName := parts[1]
		tab, _ := strconv.Atoi(parts[2])
		index, _ := strconv.Atoi(parts[3])
		total, _ := strconv.Atoi(parts[4])
		tty := parts[5]
		name := parts[6]
		panes = append(panes, Pane{
			WindowID:   windowID,
			WindowName: windowName,
			Tab:        tab,
			Index:      index,
			Total:      total,
			TTY:        tty,
			Name:       name,
			Contents:   content,
		})
	}
	return panes, nil
}

// siblings returns panes that share the same tab as the pane matching the given TTY.
func siblings(tty string) ([]Pane, error) {
	all, err := listPanes()
	if err != nil {
		return nil, err
	}
	// Find which window+tab our TTY belongs to
	var myWindowID, myTab int
	found := false
	for _, pane := range all {
		if pane.TTY == tty {
			myWindowID = pane.WindowID
			myTab = pane.Tab
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("no iTerm2 session found with TTY %s", tty)
	}
	// Collect siblings (same window+tab, different TTY)
	var result []Pane
	for _, pane := range all {
		if pane.WindowID == myWindowID && pane.Tab == myTab && pane.TTY != tty {
			result = append(result, pane)
		}
	}
	return result, nil
}

// readPane returns the contents of a specific pane by window ID, tab, and pane index.
func readPane(windowID, tab, paneIndex int) (*Pane, error) {
	all, err := listPanes()
	if err != nil {
		return nil, err
	}
	for _, pane := range all {
		if pane.WindowID == windowID && pane.Tab == tab && pane.Index == paneIndex {
			return &pane, nil
		}
	}
	// If windowID is 0, match by tab+pane in first window found
	if windowID == 0 {
		for _, pane := range all {
			if pane.Tab == tab && pane.Index == paneIndex {
				return &pane, nil
			}
		}
	}
	return nil, fmt.Errorf("pane W%d T%d P%d not found", windowID, tab, paneIndex)
}

// tailLines returns the last n lines of text. If n <= 0, returns all lines.
func tailLines(text string, lineCount int) string {
	if lineCount <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= lineCount {
		return text
	}
	return strings.Join(lines[len(lines)-lineCount:], "\n")
}
