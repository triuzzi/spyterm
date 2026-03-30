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
spyterm all [N]            # read all panes
```

IDs accept both plain numbers and prefixed forms: `W35267`, `T6`, `P2` or `35267`, `6`, `2`.

### How it works

spyterm uses iTerm2's AppleScript API to read terminal session contents. It detects
which tab it's running in by walking the process tree to find its TTY, then reads
the other panes in that tab.

```
┌──────────────────────┬──────────────────────┐
│  Claude Code         │  npm run dev         │
│  (this session)      │  (sibling pane)      │
│                      │                      │
│  > /spyterm          │  Error: Cannot find  │
│  reads ────────────> │  module 'foo'        │
│                      │                      │
├──────────────────────┤                      │
│  npm run proxy       │                      │
│  (sibling pane)      │                      │
└──────────────────────┴──────────────────────┘
```

### Aliases

| Command | Alias |
|---------|-------|
| `siblings` | `s` |
| `list` | `ls` |
| `read` | `r` |
| `all` | `a` |

## License

MIT
