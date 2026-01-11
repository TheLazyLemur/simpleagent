---
paths: "{session,todo,config}.go, tools/todo.go"
---
# State Management

Sessions: JSON in `~/.config/agent/sessions/<uuid>.json`
- UUID v7 (time-ordered)
- Always update `.Meta.UpdatedAt` on save
- Preserve `CreatedAt` from existing sessions

Todos: in-memory slice, persisted via session
- Statuses: `pending`, `in_progress`, `completed`
- `todo_write` replaces full list (no partial updates)
