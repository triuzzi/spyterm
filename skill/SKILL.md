---
name: spyterm
description: Watch and control sibling terminal panes. Use when the user asks to check what's happening in other panes, monitor dev servers, send commands to other panes, or when you need to see if a build/server is failing after making changes.
---

# spyterm — terminal pane watcher & controller

## Commands

### `/spyterm watch`

Read all sibling panes (same tab) and report what's happening.

```bash
spyterm siblings 80
```

Run the command above, then analyze the output from each pane:

1. **Identify each pane** by its label (e.g., `W35267 T5 P2`)
2. **Classify status** for each pane:
   - **Error** — stack traces, uncaught exceptions, build failures, segfaults, panic, FATAL, compilation errors
   - **Warning** — deprecation notices, non-fatal warnings, retry loops
   - **Running** — server listening, watch mode active, process idle
   - **Idle** — just a shell prompt, no active process
3. **Report only actionable items** — if a pane has errors, show the relevant error lines and suggest a fix. If everything is clean, say so briefly.
4. **Do NOT dump raw terminal output** — summarize and extract what matters.

### `/spyterm watch --fix`

Same as `watch`, but after identifying errors, automatically attempt to fix them. Only fix issues that are clearly caused by code changes (build errors, type errors, missing imports). Do not restart servers or run commands in other panes.

### `/spyterm send`

Send a command or raw keys to a specific pane:

```bash
# Send text + Enter:
spyterm send T5 P2 npm run dev          # types "npm run dev" + Enter
spyterm send W35267 T5 P2 npm run build # specific window

# Send raw keys (no Enter appended):
spyterm send --keys T5 P2 ^C            # Ctrl+C (interrupt)
spyterm send --keys T5 P2 ^D            # Ctrl+D (EOF)
spyterm send --keys T5 P2 ^Z            # Ctrl+Z (suspend)
spyterm send --keys T5 P2 ^[            # Escape

# Send text + raw Enter in one call (REQUIRED for TUI targets):
spyterm send --keys T5 P2 "some text" ^M  # types text then ^M (carriage return)
```

Common workflow — restart a dev server:
```bash
spyterm send --keys T5 P2 ^C            # stop the process
spyterm send T5 P2 npm run dev          # start it again
```

Use `spyterm list` first to find the right pane target.

#### Gotcha: `\n` vs `\r` — TUI targets need `^M`, not plain `send`

Plain `spyterm send T P "<text>"` invokes iTerm2's `write text "..."` which appends `\n` (newline / line feed / ASCII 10). **Shells happen to treat `\n` as line submit, so this works for shell commands.** But TUI applications running in raw mode (Claude Code, vim, fzf, less, htop, etc.) distinguish `\n` from `\r` — they only treat `\r` (carriage return / ASCII 13 / `^M`) as Enter. So plain `send` will type the text into the TUI's input box but **never submit it** — the prompt sits in INSERT mode.

This is NOT a multi-byte UTF-8 / em-dash issue. iTerm2's `write text` handles UTF-8 fine. The cause is purely the `\n` vs `\r` distinction at the TUI input-handler level.

**Right pattern for TUI targets:** use `--keys` with the text and `^M` together:

```bash
spyterm send --keys T5 P2 "Read /tmp/brief.md and execute" ^M
```

`send --keys` accepts a mix of literal strings and key tokens (`^M`, `^C`, etc.). Anything that doesn't match `^X`/special-key notation is treated as a literal string. The text is typed, then `^M` (= `character id 13` = carriage return) submits.

**Wrong patterns that look right but silently leave the prompt unsubmitted:**

```bash
spyterm send T5 P2 "Read /tmp/brief.md..."   # types text but TUI never sees Enter
spyterm send T5 P2 "Read /tmp/brief.md..."   # then chase with ^M as a 2nd call —
spyterm send --keys T5 P2 ^M                  #   works, but two calls is fragile (race risk)
```

#### Gotcha: race condition when omitting window ID

`spyterm send` (and `send --keys`) without an explicit window ID calls `listPanes()` to resolve which window the target pane is in. The list AppleScript does:

```applescript
repeat with t from 1 to (count of tabs of w)
  set theTab to tab t of w
  ...
```

`(count of tabs of w)` is snapshotted at loop entry. If a tab is closed during iteration (the user closes a tab manually, an iTerm session ends, etc.), the loop tries to access a now-invalid index and errors with:

```
Can't get tab N of item 1 of every window. Invalid index. (-1719)
```

(`item 1 of every window` is how AppleScript serializes the loop variable in the error.)

**Mitigation:** when you know the window ID (`spyterm list` shows it as `W80135`), pass it explicitly to skip the listPanes resolution:

```bash
spyterm send --keys W80135 T9 P2 ^M    # explicit W → no race
```

This race is most likely to fire when sending multiple commands in quick succession (the first command's UI update can overlap with the second's listPanes call).

### `/spyterm split`

Create a new pane by splitting an existing one — the current pane by default, or a
`W/T/P` target. Useful for spinning up a dev server next to yourself, then watching it.

```bash
spyterm split v                 # split current pane side by side (new pane on the right)
spyterm split h                 # split current pane stacked  (new pane below)
spyterm split h npm run dev     # split below, run "npm run dev" in the new pane
spyterm split v W35267 T6 P2    # split a specific pane instead of the current one
```

- **`v` / vertical** = side by side; **`h` / horizontal** = stacked — matching iTerm2's
  Cmd+D / Cmd+Shift+D naming (vertical divider vs horizontal divider).
- The new pane **inherits the source pane's working directory**, so `spyterm split h npm run dev`
  runs in the right project directory without a manual `cd`.
- **Focus stays on the pane that was split** (the current pane by default) — the split
  is created in the background, so you can keep working where you are. (A `W/T/P` target
  focuses that target pane instead.)
- The command prints the new pane's `W/T/P` label (e.g. `new pane W83 T9 P2/2`). Note the
  new pane's index can shift existing panes; use the printed label (or `spyterm list`) to
  target it with `read`/`send` afterward.
- **Prefix IDs (`W35267 T6 P2`) when combining a target with a command** — plain leading
  numbers are interpreted as the pane target, so prefixes disambiguate the two.

Splitting **adds** a pane (it never runs a command in an existing one), so it does not
carry the same risk as `send`. But if you pass a command to run in the new pane, that
command executes with your shell's privileges — treat the command part with the same
"ALWAYS ASK" caution as `send` below.

### `/spyterm list`

Show the pane layout:

```bash
spyterm list
```

### `/spyterm read`

Read a specific pane. Pass arguments through:

```bash
# Examples:
spyterm read T5 P2        # tab 5, pane 2, last 50 lines
spyterm read W35267 T5 P2 100  # specific window, 100 lines
```

## Behavior guidelines

- **Always use `siblings` for targeting panes** — when asked to send commands, read output, or interact with other panes, use sibling panes (same tab) unless the user explicitly specifies a different tab/window. Tab/pane IDs get renumbered, siblings are always stable.
- When reporting errors, quote the exact error message from the pane output — don't paraphrase.
- If multiple panes have errors, prioritize: build errors > runtime errors > warnings.
- For `/spyterm watch --fix`, only fix code issues. Never run commands that affect other panes (no `kill`, no `npm start`, etc.).
- The user's most common setup: Claude Code in one pane, dev server(s) in sibling pane(s). Focus on catching what broke after code changes.
- Execute `spyterm send`, `send --keys`, and `spyterm split` when necessary to complete the user's request. Inspect the target pane first; never send commands to a root or elevated/sudo shell. For destructive or irreversible commands, surface the exact command and require confirmation. Read-only commands (`siblings`, `list`, `read`, `all`) are always permitted.
- IDs accept both plain numbers and prefixed forms: `W35267`, `T6`, `P2` or `35267`, `6`, `2`.
