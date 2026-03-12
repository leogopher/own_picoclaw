# CLAUDE.md — PicoClaw

## What is this project

PicoClaw is an ultra-lightweight personal AI assistant in Go (<10MB RAM, <1s startup).
It bridges LLM providers (Claude, GPT, Qwen, DeepSeek, etc.) to messaging channels
(Telegram, Discord, Slack, Matrix, WhatsApp, and 10+ more). Runs on $10 hardware
(Raspberry Pi, RISC-V boards, old Android phones).

**Module:** `github.com/sipeed/picoclaw`
**Go version:** 1.25.7+
**Config format:** JSON (`~/.picoclaw/config.json`), NOT YAML

## Quick reference

```bash
make build          # Build for current platform (CGO_ENABLED=0, static)
make test           # Run all tests
make lint           # Run golangci-lint
make fmt            # Format code
make install        # Build and install to ~/.local/bin
make build-all      # Cross-compile for all platforms
```

Single test: `go test -run TestName -v ./pkg/session/`

## Project structure

```
cmd/picoclaw/                  Main CLI (cobra). Subcommands in internal/.
cmd/picoclaw-launcher-tui/     Web console TUI.
pkg/agent/                     Agent loop, instances, context, memory, thinking.
pkg/bus/                       Async message bus (InboundMessage ↔ OutboundMessage).
pkg/channels/                  Channel interface + 15 implementations.
pkg/config/                    Config structs, JSON loading, env var overrides.
pkg/providers/                 LLMProvider interface + factory + fallback chain.
pkg/tools/                     Tool interface + 24 built-in tools + registry.
pkg/skills/                    Skill loader and ClawHub registry.
pkg/mcp/                       Model Context Protocol support.
pkg/memory/                    JSONL-based session storage.
pkg/routing/                   Complexity-based model routing.
web/backend/                   Go HTTP API server.
web/frontend/                  React/TypeScript frontend.
config/config.example.json     Example configuration.
```

## Key conventions

### Adding code

- **Shared library code** goes in `pkg/`. CLI-specific code goes in `cmd/picoclaw/internal/`.
- **New subcommand**: create `cmd/picoclaw/internal/mycommand/command.go` with
  `NewMyCommand() *cobra.Command`, register in `cmd/picoclaw/main.go`.
- **New channel**: create `pkg/channels/myplatform/` with `init.go` that calls
  `channels.RegisterFactory("myplatform", ...)`. Implement `channels.Channel` interface.
- **New tool**: implement `tools.Tool` interface (Name, Description, Parameters, Execute).
  Register in `pkg/agent/instance.go`. Add config toggle to `ToolsConfig`.
- **New provider**: implement `providers.LLMProvider` interface (Chat, GetDefaultModel).
  For OpenAI-compat APIs, add a case in `pkg/providers/factory.go`.

### Coding style

- **Errors**: wrap with context: `fmt.Errorf("loading config: %w", err)`
- **Logging**: zerolog only. Structured fields, not string interpolation:
  `log.Info().Str("channel", name).Msg("started")`
- **Concurrency**: context.Context for cancellation, sync.RWMutex for maps,
  atomic.Bool for flags, buffered channels for async dispatch.
- **Tests**: table-driven with testify assertions. `_test.go` alongside implementation.
- **Config**: JSON tags + `env` tags for env var overrides. Update `config.example.json`.

### What NOT to do

- Don't use YAML for config — it's JSON everywhere.
- Don't use fmt.Println or log.Printf — use zerolog.
- Don't buffer entire streams in memory — this runs on Pi with <10MB target.
- Don't duplicate channel/provider code — use the existing interfaces and factories.
- Don't create separate config files — extend the existing `Config` struct.

## Active work: OpenClaw (YouTube Video Analyzer)

See `youtube-video-analyzer-spec.md` for the full spec. Summary:

- New `picoclaw openclaw "URL"` subcommand
- Package: `pkg/openclaw/` (library) + `cmd/picoclaw/internal/openclaw/` (CLI)
- Streams video frames via yt-dlp → ffmpeg pipe (no files on disk)
- Vision analysis via Qwen VL (OpenAI-compat API, reuses `pkg/providers/openai_compat/`)
- Synthesis via Qwen-plus or Kimi K2.5
- Delivers to Telegram (using `telego` directly, standalone — no gateway needed)
- Obsidian note writer (new code — no existing Obsidian integration)
- Target: Raspberry Pi 4, <512MB peak memory, ~$0.02/video

## Skills available

- `/go` — PicoClaw Go development conventions, interfaces, patterns
- `/review` — Code review checklist for picoclaw conventions
