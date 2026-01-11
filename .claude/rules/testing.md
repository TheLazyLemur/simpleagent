---
paths: "*_test.go"
---
# Testing

Structure: given-when-then (comments optional)
```go
func TestThing(t *testing.T) {
    // given
    input := ...
    // when
    result := doThing(input)
    // then
    if result != expected { t.Errorf(...) }
}
```

- Test success + error paths
- Use `t.TempDir()` for files
- Flaky tests: `t.Skip("reason")`
- No table-driven unless >3 cases with identical logic
