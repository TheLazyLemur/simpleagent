---
paths: tools/git.go
---
# Git Safety

Plan mode: read-only commands only
- `readOnlyGitCommands` allowlist: status, diff, log, show, branch, remote
- `blockedSubcommands` map: rebase, merge, cherry-pick, reset --hard
- Block force push (`-f`, `--force`) always

When adding git features, update both allowlist + blocklist as needed.
