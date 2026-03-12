---
name: review
description: "Review Go code changes in the picoclaw project for correctness, style, and adherence to project conventions. Use this skill when the user asks to review code, check a PR, audit changes, or verify that new code follows project patterns. Also trigger when the user says 'review this', 'check my code', 'does this look right', or asks about code quality."
---

# Code Review — PicoClaw Project

Review code changes against picoclaw conventions. Focus on real problems, not style nitpicks.

## Review Checklist

### 1. Architecture Fit
- Does the code belong in the right package? (`pkg/` for shared, `cmd/.../internal/` for CLI-specific)
- Does it implement the correct interface? (Tool, Channel, LLMProvider)
- Does it follow the registration pattern? (init() + factory for channels, conditional registration for tools)
- Does it extend config properly? (add to Config struct in `pkg/config/config.go`, update `config.example.json`)

### 2. Concurrency Safety
- Shared maps protected by `sync.RWMutex`?
- Flags using `atomic.Bool` / `atomic.Uint64`?
- Context passed as first parameter and respected for cancellation?
- No goroutine leaks? (goroutines should exit when context is cancelled)
- Channel operations non-blocking or select-based with cancellation?

### 3. Error Handling
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`?
- No swallowed errors (errors logged but not returned, or silently ignored)?
- Provider errors classified as `FailoverError` when appropriate?
- Graceful degradation where possible?

### 4. Resource Management
- `defer` for cleanup (Close, Unlock, cancel)?
- No resource leaks (HTTP clients, file handles, goroutines)?
- Memory-conscious? (picoclaw targets <10MB RAM on embedded devices)
- Streaming over buffering where possible?

### 5. Logging
- Uses zerolog, not fmt.Println or log.Printf?
- Structured fields: `.Str()`, `.Int()`, `.Err()` — not string interpolation in `.Msg()`
- Appropriate log levels: Debug for internal state, Info for lifecycle events, Warn for recoverable issues, Error for failures

### 6. Testing
- Table-driven tests with descriptive names?
- Uses testify assertions (assert.Equal, require.NoError)?
- Tests alongside implementation (_test.go in same package)?
- Edge cases covered? (empty input, nil, context cancellation)

### 7. Config Integration
- New config fields have `json` tags?
- Environment variable overrides via `env` tags?
- Defaults are sensible?
- Sensitive values (API keys, tokens) support env vars?
- `config.example.json` updated?

### 8. API Compatibility
- Public interfaces not broken without good reason?
- New optional capabilities added as separate interfaces (like TypingCapable, MediaSender)?
- Wire formats (JSON, bus messages) backward-compatible?

## Review Output Format

For each issue found:
```
**[severity] file:line — summary**
explanation and suggested fix
```

Severities:
- **bug** — will cause incorrect behavior or crash
- **issue** — works but violates conventions or has a subtle problem
- **nit** — minor style or readability suggestion (keep these few)
- **question** — unclear intent, needs clarification

End with a summary: what the change does, overall assessment, and whether it's ready to merge.
