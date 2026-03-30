---
name: spyterm
description: Watch sibling terminal panes for errors, build failures, and server crashes. Use when the user asks to check what's happening in other panes, monitor dev servers, or when you need to see if a build/server is failing after making changes.
---

# spyterm — terminal pane watcher

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

- When reporting errors, quote the exact error message from the pane output — don't paraphrase.
- If multiple panes have errors, prioritize: build errors > runtime errors > warnings.
- For `/spyterm watch --fix`, only fix code issues. Never run commands that affect other panes (no `kill`, no `npm start`, etc.).
- The user's most common setup: Claude Code in one pane, dev server(s) in sibling pane(s). Focus on catching what broke after code changes.
