---
name: go
description: "PicoClaw Go development conventions and patterns. Use this skill whenever writing, modifying, or reviewing Go code in the picoclaw project — including adding new features, fixing bugs, creating new packages, adding CLI subcommands, implementing channels/tools/providers, or working with the config system. Also trigger when the user asks about project structure, build system, or testing patterns."
---

# Go Development — PicoClaw Project

## Project Identity

- **Module**: `github.com/sipeed/picoclaw`
- **Go version**: 1.25.7+
- **Config format**: JSON (never YAML)
- **Binary**: `picoclaw` (single static binary, CGO_ENABLED=0)
- **CLI framework**: spf13/cobra
- **Logging**: rs/zerolog (structured, leveled)
- **Testing**: testify/assert + table-driven tests

## Package Layout

```
cmd/picoclaw/              # Main binary entry point
cmd/picoclaw/internal/     # Subcommand implementations (one package per command)
pkg/                       # Shared library packages
pkg/agent/                 # Agent loop, instances, context, memory
pkg/bus/                   # Async message bus (InboundMessage, OutboundMessage)
pkg/channels/              # Channel interface + 15 implementations
pkg/config/                # Config structs, loading, env var overrides
pkg/providers/             # LLM provider interface + implementations
pkg/tools/                 # Tool interface + 24 built-in tools
pkg/skills/                # Skill loader and registry
pkg/mcp/                   # Model Context Protocol support
web/backend/               # Go HTTP API server
web/frontend/              # React/TypeScript frontend
```

## Core Interfaces

When adding new functionality, implement the relevant interface:

### Adding a new Tool

```go
// pkg/tools/base.go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]any                           // JSON Schema
    Execute(ctx context.Context, args map[string]any) *ToolResult
}
```

Register in `pkg/agent/instance.go` conditionally:
```go
if cfg.Tools.IsToolEnabled("my_tool") {
    toolsRegistry.Register(tools.NewMyTool(cfg, mediaStore))
}
```

Add config field to `ToolsConfig` in `pkg/config/config.go`:
```go
MyTool GenericToolConfig `json:"my_tool"`
```

### Adding a new Channel

1. Create `pkg/channels/myplatform/` with `init.go` + `myplatform.go`
2. Register via init():
```go
func init() {
    channels.RegisterFactory("myplatform", func(cfg *config.Config, b *bus.MessageBus) (channels.Channel, error) {
        return NewMyPlatformChannel(cfg, b)
    })
}
```
3. Implement `channels.Channel` interface (Name, Start, Stop, Send, IsRunning, IsAllowed, IsAllowedSender, ReasoningChannelID)
4. Optionally implement capability interfaces: TypingCapable, MessageEditor, PlaceholderCapable, ReactionCapable, MediaSender, CommandRegistrarCapable, WebhookHandler
5. Add config struct to `pkg/config/config.go` under `ChannelsConfig`

### Adding a new LLM Provider

```go
// pkg/providers/types.go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, options map[string]any) (*LLMResponse, error)
    GetDefaultModel() string
}
```

For OpenAI-compatible APIs, reuse `pkg/providers/openai_compat/provider.go` — just add a new case in `pkg/providers/factory.go`.

### Adding a new CLI Subcommand

1. Create `cmd/picoclaw/internal/mycommand/command.go`
2. Follow cobra pattern:
```go
func NewMyCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "What it does",
        RunE: func(cmd *cobra.Command, args []string) error {
            // implementation
        },
    }
    return cmd
}
```
3. Register in `cmd/picoclaw/main.go`: `cmd.AddCommand(mycommand.NewMyCommand())`

## Configuration Pattern

Config is JSON, loaded from `~/.picoclaw/config.json` with env var overrides (`PICOCLAW_*`).

Add new config fields to `pkg/config/config.go`:
```go
type Config struct {
    // ... existing fields ...
    MyFeature MyFeatureConfig `json:"my_feature,omitempty"`
}

type MyFeatureConfig struct {
    Enabled bool   `json:"enabled"`
    APIKey  string `json:"api_key" env:"PICOCLAW_MY_FEATURE_API_KEY"`
}
```

Update `config/config.example.json` with the new section.

## Coding Conventions

### Error Handling
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Return errors up, handle at boundaries
- Use `FailoverError` for provider errors that need classification

### Logging
```go
import "github.com/rs/zerolog/log"

log.Info().Str("channel", name).Msg("channel started")
log.Error().Err(err).Str("tool", t.Name()).Msg("execution failed")
log.Debug().Int("count", n).Msg("frames extracted")
```

### Concurrency
- Use `context.Context` for cancellation (pass as first param)
- Use `sync.RWMutex` for concurrent map access
- Use `atomic.Bool` / `atomic.Uint64` for flags and counters
- Use buffered channels for async dispatch
- Use `sync.WaitGroup` or `errgroup.Group` for parallel work

### Context Values
- Use unexported pointer-typed keys (collision-free):
```go
type ctxKey struct{ name string }
var myKey = &ctxKey{"myValue"}
```

### Naming
- Interfaces: capability-named (TypingCapable, MediaSender)
- Factories: `NewXxx(cfg, deps...) (*Xxx, error)`
- Test files: `xxx_test.go` alongside implementation
- Package names: lowercase, single word when possible

## Build System

```bash
make build        # Build for current platform
make build-all    # Build for all platforms
make test         # Run tests
make lint         # Run linters
make fmt          # Format code
make install      # Build and install to ~/.local/bin
```

Build flags: CGO_ENABLED=0, version/commit/time injected via ldflags.

## Testing

Table-driven tests with testify:
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty input", "", ""},
        {"normal case", "hello", "HELLO"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

Run tests: `go test ./pkg/... ./cmd/...` or `make test`

## Message Flow (for reference)

1. Channel receives platform update -> creates `bus.InboundMessage`
2. Published to `bus.MessageBus` (buffered channel)
3. Agent loop picks up -> builds context (history, tools, system prompt)
4. Calls LLM provider (with fallback chain)
5. Parses tool calls -> executes tools (parallel via WaitGroup)
6. Loops until no tool calls or max iterations
7. Publishes `bus.OutboundMessage` to bus
8. Manager routes to channel worker -> rate limit -> send with retry

## Key Dependencies

| Purpose | Package |
|---------|---------|
| CLI | `github.com/spf13/cobra` |
| Logging | `github.com/rs/zerolog` |
| Config env | `github.com/caarlos0/env/v11` |
| Testing | `github.com/stretchr/testify` |
| Telegram | `github.com/mymmrac/telego` |
| Discord | `github.com/bwmarrin/discordgo` |
| Claude SDK | `github.com/anthropics/anthropic-sdk-go` |
| OpenAI SDK | `github.com/openai/openai-go/v3` |
| MCP | `github.com/modelcontextprotocol/go-sdk` |
| WebSocket | `github.com/gorilla/websocket` |
| UUID | `github.com/google/uuid` |
