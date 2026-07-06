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
	SessionID  string // iTerm2's stable session identifier (immune to index reshuffling)
	Contents   string
}

func (pane Pane) Label() string {
	return fmt.Sprintf("W%d T%d P%d/%d", pane.WindowID, pane.Tab, pane.Index, pane.Total)
}

const recordSep = "<<ITERM_PANE_RECORD>>"

// listScript enumerates every pane in every tab in every window using
// reference-based iteration (`repeat with x in collection`). Reference
// iteration keeps a stable handle to each element evaluated at the start of
// the loop, avoiding the -1719 ("Invalid index") TOCTOU race that the
// previous index-based loop (`repeat with t from 1 to count`) was prone to.
// No `try` blocks: if a tab/session vanishes mid-iteration the script fails
// loudly so the caller knows the listing is incomplete instead of receiving
// a silently truncated result.
const listScript = `
set sep to "<<ITERM_PANE_RECORD>>"
set delim to character id 9
tell application "iTerm2"
	set output to ""
	repeat with w in windows
		set wid to id of w
		set wname to name of w
		set tabIndex to 0
		repeat with theTab in tabs of w
			set tabIndex to tabIndex + 1
			set sessIndex to 0
			set sessList to sessions of theTab
			set sc to count of sessList
			repeat with s in sessList
				set sessIndex to sessIndex + 1
				set theTTY to tty of s as text
				set sname to name of s
				set suid to id of s
				set output to output & sep & wid & delim & wname & delim & tabIndex & delim & sessIndex & delim & sc & delim & theTTY & delim & sname & delim & suid & linefeed & (contents of s)
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
		parts := strings.SplitN(meta, "\t", 8)
		if len(parts) < 8 {
			continue
		}
		windowID, _ := strconv.Atoi(parts[0])
		windowName := parts[1]
		tab, _ := strconv.Atoi(parts[2])
		index, _ := strconv.Atoi(parts[3])
		total, _ := strconv.Atoi(parts[4])
		tty := parts[5]
		name := parts[6]
		sessionID := parts[7]
		panes = append(panes, Pane{
			WindowID:   windowID,
			WindowName: windowName,
			Tab:        tab,
			Index:      index,
			Total:      total,
			TTY:        tty,
			Name:       name,
			SessionID:  sessionID,
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
	me, err := findByTTY(all, tty)
	if err != nil {
		return nil, err
	}
	// Collect siblings (same window+tab, different TTY)
	var result []Pane
	for i := range all {
		if all[i].WindowID == me.WindowID && all[i].Tab == me.Tab && all[i].TTY != tty {
			result = append(result, all[i])
		}
	}
	return result, nil
}

// paneByTTY returns the pane whose TTY matches, or an error if none is found.
func paneByTTY(tty string) (*Pane, error) {
	all, err := listPanes()
	if err != nil {
		return nil, err
	}
	return findByTTY(all, tty)
}

// findByTTY returns the pane in panes whose TTY matches, or an error if none does.
func findByTTY(panes []Pane, tty string) (*Pane, error) {
	for i := range panes {
		if panes[i].TTY == tty {
			return &panes[i], nil
		}
	}
	return nil, fmt.Errorf("no iTerm2 session found with TTY %s", tty)
}

// paneBySessionID returns the pane with the given stable session ID. It errors
// if the listing fails or no session matches: for a freshly-split session a miss
// can only mean the AppleScript listing failed or the topology changed under us,
// both of which the caller should surface rather than mask behind a placeholder.
func paneBySessionID(sessionID string) (*Pane, error) {
	all, err := listPanes()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].SessionID == sessionID {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("session %s not found", sessionID)
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

// escapeForAppleScript escapes a Go string for safe interpolation into a
// double-quoted AppleScript string literal: backslashes first, then quotes.
func escapeForAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// sessionWalk builds an AppleScript handler that walks every window→tab→session
// and, at the session whose id matches the handler's targetID argument, runs
// `action` with `s` bound to that session (the action must `return`). If no
// session matches, it errors loudly. The session ID is immune to tab/pane
// reordering, so addressing never raises -1719 ("Invalid index") from the walk
// itself. No `try` blocks: a vanishing tab/session surfaces instead of being
// silently skipped, and the final `error` fires only when the walk completes
// without a match.
func sessionWalk(handler, action string) string {
	return fmt.Sprintf(`
on %s(targetID)
	tell application "iTerm2"
		repeat with w in windows
			repeat with t in tabs of w
				repeat with s in sessions of t
					if (id of s) is targetID then
%s
					end if
				end repeat
			end repeat
		end repeat
	end tell
	error "session with id " & targetID & " no longer exists"
end %s
`, handler, action, handler)
}

// sendToPane sends a text command to a specific pane addressed by SessionID.
// The text is sent followed by a carriage return (^M / ASCII 13), which both
// shells (via the TTY's ICRNL flag) and raw-mode TUIs (Claude Code, vim, fzf,
// less, htop, etc.) accept as Enter. iTerm2's default newline is \n (ASCII 10),
// which shells cook as Enter but raw-mode TUIs ignore — so plain default would
// leave TUI prompts typed-but-unsubmitted.
func sendToPane(sessionID, text string) error {
	action := fmt.Sprintf(`						tell s to write text ("%s" & (character id 13)) newline no
						return`, escapeForAppleScript(text))
	script := sessionWalk("findSession", action) +
		fmt.Sprintf("\nfindSession(\"%s\")\n", escapeForAppleScript(sessionID))
	_, err := runAppleScript(script)
	return err
}

// sendKeysToPane sends raw key sequences to a pane addressed by SessionID.
// Keys are specified as a sequence of key names separated by spaces.
// Uses Unix caret notation: ^C (Ctrl+C), ^D (Ctrl+D), ^Z (Ctrl+Z), etc.
func sendKeysToPane(sessionID string, keys []string) error {
	var parts []string
	for _, key := range keys {
		charID := resolveKey(key)
		if charID >= 0 {
			parts = append(parts, fmt.Sprintf("character id %d", charID))
		} else {
			parts = append(parts, fmt.Sprintf("\"%s\"", escapeForAppleScript(key)))
		}
	}
	action := fmt.Sprintf(`						tell s to write text (%s) newline no
						return`, strings.Join(parts, " & "))
	script := sessionWalk("findSession", action) +
		fmt.Sprintf("\nfindSession(\"%s\")\n", escapeForAppleScript(sessionID))
	_, err := runAppleScript(script)
	return err
}

// splitPane splits the session addressed by SessionID and returns the new
// session's stable id and tty. direction must be exactly directionVertical (new
// pane beside the original) or directionHorizontal (new pane below it) — the CLI
// maps the v/h shorthands to these, so the verb is never user-controlled and is
// safe to interpolate into the script.
//
// The new pane inherits the source pane's working directory (via iTerm2's
// session.path variable) so a split reproduces the pane you split — the useful
// behavior when splitting to run project commands, regardless of the profile's
// working-directory setting. After splitting, iTerm2 focuses the new pane; the
// script re-selects the split source so that pane keeps the keyboard cursor.
func splitPane(sessionID, direction string) (newSessionID, newTTY string, err error) {
	action := fmt.Sprintf(`						set delim to character id 9
						tell s to set srcPath to (variable named "session.path")
						tell s
							set newSession to (split %s with same profile)
						end tell
						if srcPath is not "" then
							tell newSession to write text ("cd " & quoted form of srcPath)
						end if
						tell s to select
						return ((id of newSession) & delim & (tty of newSession)) as text`, direction)
	script := sessionWalk("findSession", action) +
		fmt.Sprintf("\nreturn findSession(\"%s\")\n", escapeForAppleScript(sessionID))
	output, err := runAppleScript(script)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(strings.TrimSpace(output), "\t", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected split result: %q", output)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

// resolveKey maps a key name to an ASCII character code, or -1 if not a known key.
// Uses Unix caret notation: ^A through ^Z, ^[, ^\, ^], ^^, ^_
func resolveKey(key string) int {
	// Caret notation: ^C, ^D, ^Z, etc.
	if len(key) == 2 && key[0] == '^' {
		char := key[1]
		if char >= 'A' && char <= 'Z' {
			return int(char - 'A' + 1)
		}
		if char >= 'a' && char <= 'z' {
			return int(char - 'a' + 1)
		}
		switch char {
		case '[':
			return 27 // Escape
		case '\\':
			return 28 // SIGQUIT
		case ']':
			return 29
		case '^':
			return 30
		case '_':
			return 31
		case '?':
			return 127 // Delete/Backspace
		}
	}
	return -1
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
