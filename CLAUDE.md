# CLAUDE.md

This file provides guidance to coding assistants when working with code in this repository.

## Build & Run

```bash
go run .                     # start new session
go run . -resume <id>        # resume session
go run . -sessions           # list sessions
go run . -delete <id>        # delete session
```

Requires `ANTHROPIC_API_KEY` env var.

## Architecture

Simple CLI agent (`api.minimax.io/anthropic`).

**Files:**
- `main.go` - CLI entrypoint, agentic loop
- `session.go` - Session persistence (save/load/list/delete)
- `reminders.go` - System reminder injection (plan mode, todo nudges)
- `tools/` - Tool package with registry pattern
  - `tools.go` - Registry: `All()`, `ReadOnly()`, `Execute()`
  - `fs.go` - read_file, write_file, ls, mkdir, rm
  - `replace.go` - replace_text
  - `git.go` - git
  - `search.go` - grep (search for patterns in files)
  - `todo.go` - todo_write, `Todo` type, `ExecuteTodo()`, `HasPending()`, `RenderTodos()`
  - `questions.go` - ask_user_question (interactive prompts)
- `CLAUDE.md` - Appended to system prompt if present

**Commands:**
- `/plan` - Toggle plan mode (read-only exploration)

**Todo system:** `todo_write` replaces full list. Persisted in session JSON. Auto-injects reminder when pending todos exist and 3+ turns without update.

**Dependency:** `openagent/claude` - Local Claude API client with streaming support.

**Key types from claude package:**
- `Client` - API client, created via `NewClient(opts...)`, uses `ANTHROPIC_API_KEY` by default
- `MessageStream` - Streaming response handler with event callbacks (`OnText`, `OnMessage`, etc.)
- `Tool`, `InputSchema`, `Property` - Tool definition structs
- `MessageParam`, `ContentBlock`, `ToolResultBlock` - Message types for agentic loop

Sessions stored in `~/.config/agent/sessions/<uuid>.json` using UUID v7 (time-ordered).

Codemaps (CODEMAP_*.yaml) document code flows across files with annotated code blocks.

  Finding Codemaps

  # Find all codemaps
  find . -name "CODEMAP_*.yaml" -o -name "CODEMAP_*.md"

  # Search by topic
  rg -l "authentication|auth" CODEMAP_*.yaml

  Structure

  title: Flow name
  summary: One-line description
  description: Key blocks referenced as [1a], [2b], etc.
  sections:
    - id: 1
      title: Section name
      tree:
        - text: Explanation
          children:
            - block:
                id: "1a"
                title: Block name
                code: "snippet"
                file: path/to/file.go
                line: 25

  Usage

  1. Read the summary - understand what flow is documented
  2. Follow sections - sections are ordered steps in the flow
  3. Use block references - file:line lets you jump to source
  4. Cross-reference - description lists key blocks [1a], [2b] for quick lookup

  When asked about a flow/feature, search codemaps first before grep-diving.
