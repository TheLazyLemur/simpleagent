---
paths: tools/*.go
---
# Tool Implementation

Register in `init()` with `claude.Tool` struct:
```go
func init() {
    register(claude.Tool{
        Name: "tool_name",
        Description: "...",
        InputSchema: claude.InputSchema{...},
    }, handlerFunc)
}
```

Handler pattern:
- Parse `json.RawMessage` into typed struct
- Return `Result` interface (implements `String()` + `Render()`)
- Error: `newResult("toolname", "error: ...")` - never panic

Optional fields use pointers: `*int`, `*bool` for JSON null handling.
