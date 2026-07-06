# spyterm

Spy on your iTerm2 split panes from the terminal.

![Go 1.24+](https://img.shields.io/badge/go-1.24+-00ADD8)
![macOS](https://img.shields.io/badge/platform-macOS-lightgrey)
![License: MIT](https://img.shields.io/badge/license-MIT-green)

Built for AI coding assistants like Claude Code: run dev servers in split panes,
and the assistant reads errors directly — no copy-paste needed.

## Install

```bash
# If $GOBIN or $GOPATH/bin is in your PATH:
go install github.com/triuzzi/spyterm@latest

# Or build and place it explicitly:
go install github.com/triuzzi/spyterm@latest && ln -sf "$(go env GOBIN)/spyterm" ~/.local/bin/spyterm
```

## Usage

```bash
spyterm                    # read sibling panes (default)
spyterm siblings [N]       # last N lines from sibling panes (default 80)
spyterm list [-v]          # show all windows/tabs/panes (-v for content)
spyterm read [W] T P [N]   # read a specific pane
spyterm send [W] T P CMD   # send a command to a pane (text + Enter)
spyterm send --keys T P K  # send raw keys (^C, ^D, ^Z, ^[, etc.)
spyterm split <v|h> [W T P] [CMD]  # split a pane; optionally run CMD in it
spyterm all [N]            # read all panes
```

IDs accept both plain numbers and prefixed forms: `W35267`, `T6`, `P2` or `35267`, `6`, `2`.

### Splitting panes

`spyterm split` creates a new pane by splitting an existing one — the current pane
by default, or a specific `W/T/P` target. Handy for an assistant that wants to spin
up a dev server next to itself and then watch it.

```bash
spyterm split v                  # split current pane side by side (new pane right)
spyterm split h                  # split current pane stacked (new pane below)
spyterm split h npm run dev      # split below, run "npm run dev" in the new pane
spyterm split v W35267 T6 P2     # split a specific pane instead of the current one
```

- **`v` / vertical** = side by side (vertical divider); **`h` / horizontal** = stacked
  (horizontal divider) — matching iTerm2's own Cmd+D / Cmd+Shift+D naming.
- The new pane **inherits the source pane's working directory**, so splitting to run
  project commands lands you in the right place regardless of your profile's
  working-directory setting.
- **Focus stays on the current pane** — the split happens in the background, so an
  assistant can create and drive a pane without stealing your cursor.
- The command prints the new pane's `W/T/P` label so you can `read` or `send` to it
  next.

### How it works

spyterm uses iTerm2's AppleScript API to read terminal session contents. It detects
which tab it's running in by walking the process tree to find its TTY, then reads
the other panes in that tab.

```
┌───────────────────────┬──────────────────────┐
│  Claude Code          │  npm run dev         │
│  (this session)       │  (sibling pane)      │
│                       │                      │
│  > /spyterm           │  Error: Cannot find  │
│  reads ────────────>  │  module 'foo'        │
│                       │                      │
├───────────────────────┤                      │
│  npm run proxy        │                      │
│  (sibling pane)       │                      │
└───────────────────────┴──────────────────────┘
```

### Aliases

| Command | Alias |
|---------|-------|
| `siblings` | `s` |
| `list` | `ls` |
| `read` | `r` |
| `all` | `a` |

## Security

The `send` command lets you (or an AI agent) execute arbitrary commands in other
iTerm2 terminal sessions. This is powerful — an agent can restart a crashed dev
server or send Ctrl+C to a hung process — but it means any tool calling
`spyterm send` can type into your other panes as if it were you at the keyboard.

The permission boundary lives at the agent/skill level, not in spyterm itself:

- **Claude Code** gates bash commands behind user approval by default.
- The [spyterm skill](skill/SKILL.md) instructs agents to always ask for
  confirmation before sending commands, even when running with
  `--dangerously-skip-permissions`.
- Read-only commands (`siblings`, `list`, `read`, `all`) carry no write risk.

If you only need observation, you never have to use `send`. If you do use it,
understand that anything an agent sends will execute with your shell's full
privileges in the target pane.

## License

MIT
