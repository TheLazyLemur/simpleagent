---
# No pattern - always loaded
---
# General

- Commits: concise, descriptive (no marketing)
- Tests: required for new functionality
- Comments: only for non-obvious logic
- Functions: ~20 lines max, single purpose

Architecture:
- `main.go` - CLI entrypoint, agentic loop
- `tools/` - Registry-based tool system
- `session.go` - JSON persistence
- `config.go` - Rule loading, system prompt
- `reminders.go` - Reminder injection
