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
spyterm all [N]            # read all panes
```

IDs accept both plain numbers and prefixed forms: `W35267`, `T6`, `P2` or `35267`, `6`, `2`.

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
