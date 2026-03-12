# OpenClaw — YouTube Video Analyzer Spec

> Adapted from original CLAUDE.md plan after auditing the actual picoclaw codebase.
> Corrections marked with **[CORRECTED]** where the original plan assumed wrong.

## Overview

A new `picoclaw openclaw` subcommand that:
1. Takes a YouTube URL
2. Extracts frames via stream-pipe (yt-dlp → ffmpeg, no file on disk)
3. Sends frames to Qwen Vision LLM for visual analysis
4. Gets transcript from captions
5. Synthesizes everything via Qwen or Kimi into a structured summary
6. Delivers to Telegram (primary) and Obsidian (secondary)

## Codebase Audit Results

### What exists and can be reused

| Component | Location | How to reuse |
|-----------|----------|--------------|
| **Telegram bot** | `pkg/channels/telegram/telegram.go` | Uses `github.com/mymmrac/telego`. Has SendMessage (HTML), SendPhoto, message splitting. However, it's a bidirectional **channel** (receives + sends), not a one-way sender. For openclaw, import `telego` directly for lightweight one-way delivery. |
| **OpenAI-compat LLM client** | `pkg/providers/openai_compat/provider.go` | Full OpenAI-compat HTTP client. Accepts `base_url`, `api_key`, `model`. Can be used for Qwen/Kimi vision and synthesis calls. |
| **Config system** | `pkg/config/config.go` | **[CORRECTED]** JSON format (NOT YAML). Struct-based with `json` tags and `env` tag overrides via `caarlos0/env`. |
| **CLI framework** | `cmd/picoclaw/main.go` | Cobra-based. Subcommands in `cmd/picoclaw/internal/`. Each command is a package with `NewXxxCommand() *cobra.Command`. |
| **Message bus** | `pkg/bus/` | InboundMessage/OutboundMessage types. Could publish to existing Telegram channel, but adds complexity vs direct telego calls. |
| **Provider factory** | `pkg/providers/factory.go` | Creates providers from ModelConfig. Already supports DashScope (`qwen`) and Moonshot (`moonshot`) in the factory. |

### What does NOT exist

| Component | Status | Action needed |
|-----------|--------|---------------|
| **Obsidian writer** | **[CORRECTED]** Does NOT exist. Original plan said "reuse picoclaw's Obsidian writer" — there is none. | Build from scratch: write markdown files + save JPEG assets to vault directory. Simple file I/O, ~100 lines. |
| **Frame extraction** | Does not exist | Core new functionality. yt-dlp → ffmpeg pipe + JPEG splitter. |
| **Vision analysis** | Does not exist | New code, but reuses openai_compat provider pattern. |
| **Synthesis** | Does not exist | New code, same openai_compat pattern. |

## Architecture Decisions

### Package location

**[CORRECTED]** Original plan said `internal/openclaw/`. Picoclaw convention is:
- Shared packages → `pkg/`
- CLI-specific code → `cmd/picoclaw/internal/`

Decision: `pkg/openclaw/` for all logic (it's a library), `cmd/picoclaw/internal/openclaw/` for the cobra command.

```
pkg/openclaw/
├── types.go          # VideoMeta, Frame, TranscriptLine, VideoSummary
├── metadata.go       # yt-dlp --dump-json → VideoMeta
├── transcript.go     # yt-dlp subtitle extraction → []TranscriptLine
├── frames.go         # yt-dlp | ffmpeg pipe + JPEG splitter → []Frame
├── vision.go         # Qwen VL frame analysis
├── synthesis.go      # Qwen/Kimi summary generation
├── telegram.go       # One-way Telegram delivery (uses telego directly)
├── obsidian.go       # Markdown note + JPEG asset writer (NEW, from scratch)
├── analyze.go        # Top-level orchestrator: Analyze(ctx, url, cfg) error
└── *_test.go         # Tests for each

cmd/picoclaw/internal/openclaw/
├── command.go        # NewOpenClawCommand() *cobra.Command
└── helpers.go        # Flag parsing, config loading
```

### Config format

**[CORRECTED]** JSON, not YAML. Add `OpenClaw` field to existing `Config` struct.

```go
// In pkg/config/config.go — add to Config struct:
type Config struct {
    // ... existing fields ...
    OpenClaw OpenClawConfig `json:"openclaw,omitempty"`
}

type OpenClawConfig struct {
    Providers OpenClawProviders `json:"providers"`
    Frames    FrameConfig       `json:"frames"`
    Transcript TranscriptConfig `json:"transcript"`
    Telegram  OpenClawTelegram  `json:"telegram"`
    Obsidian  ObsidianConfig    `json:"obsidian"`
    Tools     OpenClawTools     `json:"tools"`
}

type OpenClawProviders struct {
    Vision    OpenClawProvider `json:"vision"`
    Synthesis OpenClawProvider `json:"synthesis"`
}

type OpenClawProvider struct {
    Model            string `json:"model"`
    BaseURL          string `json:"base_url"`
    APIKey           string `json:"api_key"          env:"PICOCLAW_OPENCLAW_VISION_API_KEY"`
    MaxFramesPerBatch int   `json:"max_frames_per_batch"`
    TimeoutSeconds   int    `json:"timeout_seconds"`
}

type FrameConfig struct {
    Resolution     int     `json:"resolution"`       // default: 720
    SceneThreshold float64 `json:"scene_threshold"`  // default: 0.3
    MaxFrames      int     `json:"max_frames"`       // default: 50
    JPEGQuality    int     `json:"jpeg_quality"`     // default: 85
}

type TranscriptConfig struct {
    Languages   []string `json:"languages"`    // default: ["en", "ru"]
    FallbackASR bool     `json:"fallback_asr"` // default: false
}

type OpenClawTelegram struct {
    // Reuses the existing channels.telegram.token for the bot token
    // Only openclaw-specific settings here
    IncludeFrames     bool `json:"include_frames"`       // default: true
    MaxFramesInDigest int  `json:"max_frames_in_digest"` // default: 5
    ChatID            string `json:"chat_id"`            // where to send results
}

type ObsidianConfig struct {
    VaultPath   string `json:"vault_path"    env:"PICOCLAW_OPENCLAW_OBSIDIAN_VAULT"`
    VideoNoteDir string `json:"video_note_dir"` // default: "Videos"
    AssetsDir   string `json:"assets_dir"`      // default: "assets/openclaw"
}

type OpenClawTools struct {
    YtdlpPath  string `json:"ytdlp_path"`   // default: "yt-dlp"
    FfmpegPath string `json:"ffmpeg_path"`   // default: "ffmpeg"
}
```

Example `config.json` addition:
```json
{
  "openclaw": {
    "providers": {
      "vision": {
        "model": "qwen3-vl-flash",
        "base_url": "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
        "api_key": "",
        "max_frames_per_batch": 4,
        "timeout_seconds": 60
      },
      "synthesis": {
        "model": "qwen-plus",
        "base_url": "https://dashscope-intl.aliyuncs.com/compatible-mode/v1",
        "api_key": "",
        "timeout_seconds": 120
      }
    },
    "frames": {
      "resolution": 720,
      "scene_threshold": 0.3,
      "max_frames": 50,
      "jpeg_quality": 85
    },
    "transcript": {
      "languages": ["en", "ru"],
      "fallback_asr": false
    },
    "telegram": {
      "include_frames": true,
      "max_frames_in_digest": 5,
      "chat_id": ""
    },
    "obsidian": {
      "vault_path": "",
      "video_note_dir": "Videos",
      "assets_dir": "assets/openclaw"
    },
    "tools": {
      "ytdlp_path": "yt-dlp",
      "ffmpeg_path": "ffmpeg"
    }
  }
}
```

### Telegram delivery strategy

**Option A (recommended):** Import `telego` directly in `pkg/openclaw/telegram.go`. Create a lightweight bot instance using the existing `channels.telegram.token` from config. Call `bot.SendMessage()` and `bot.SendPhoto()` directly. This avoids coupling to the channel lifecycle/bus system.

**Option B:** Publish OutboundMessage to the existing bus and let the Telegram channel handle it. Requires the gateway to be running, which may not be the case for a CLI-only invocation.

Going with **Option A** — openclaw should work standalone via `picoclaw openclaw "URL"` without needing the full gateway.

### LLM client strategy

The existing `pkg/providers/openai_compat/provider.go` implements `LLMProvider.Chat()` which sends text messages. However, openclaw needs **vision API support** (image_url content parts), which the current openai_compat provider may not handle.

**Decision:** Create a lightweight OpenAI-compat HTTP client in `pkg/openclaw/llm.go` that:
- Supports multimodal content (text + image_url parts)
- Uses the same config pattern (base_url, api_key, model)
- Is simpler than the full provider (no tool calls, no streaming, no fallback chain)
- Returns parsed JSON response content

If the existing openai_compat already supports vision content, extend it instead.

### Obsidian writer (new code)

Simple implementation since no existing code:
```go
// pkg/openclaw/obsidian.go

func WriteObsidianNote(cfg ObsidianConfig, summary VideoSummary, frames []Frame) error {
    // 1. Create note directory: {vault_path}/{video_note_dir}/
    // 2. Create assets directory: {vault_path}/{assets_dir}/
    // 3. Save notable frame JPEGs to assets dir
    // 4. Render markdown template with wikilinks to frames
    // 5. Write .md file to note directory
    // 6. Optionally append link to daily note
}
```

## Implementation Steps (Corrected)

### Step 1: Config (extend existing)
- Add `OpenClawConfig` and sub-structs to `pkg/config/config.go`
- Add `OpenClaw OpenClawConfig` field to `Config` struct
- Update `config/config.example.json`
- Add env var tags for API keys

### Step 2: Types
- Create `pkg/openclaw/types.go` with VideoMeta, Frame, TranscriptLine, VideoSummary, etc.

### Step 3: Metadata extraction
- `pkg/openclaw/metadata.go` — `GetMetadata(ctx, url, ytdlpPath) (VideoMeta, error)`
- Single `exec.CommandContext` call: `yt-dlp --dump-json --no-download "URL"`
- Parse JSON stdout into VideoMeta

### Step 4: Transcript extraction
- `pkg/openclaw/transcript.go` — `GetTranscript(ctx, url, cfg) ([]TranscriptLine, error)`
- Download subtitle file via yt-dlp or direct HTTP from metadata URLs
- Parse json3 subtitle format into []TranscriptLine
- Return empty slice (not error) if no captions available

### Step 5: Frame extraction (core)
- `pkg/openclaw/frames.go` — `ExtractFrames(ctx, url, cfg) (<-chan Frame, error)`
- Returns a channel (not slice) for streaming/memory efficiency
- yt-dlp → ffmpeg pipe, JPEG SOI/EOI splitting
- Scene detection with configurable threshold
- Fallback to fixed interval if scene detection produces too few frames

### Step 6: LLM client
- `pkg/openclaw/llm.go` — lightweight OpenAI-compat client with vision support
- Or extend existing `pkg/providers/openai_compat/` if it can handle multimodal
- `ChatWithVision(ctx, images []string, prompt string, cfg OpenClawProvider) (string, error)`
- `ChatText(ctx, prompt string, cfg OpenClawProvider) (string, error)`

### Step 7: Vision analysis
- `pkg/openclaw/vision.go` — `AnalyzeFrames(ctx, frames []Frame, cfg) ([]Frame, error)`
- Batch frames (max_frames_per_batch per request)
- Limit concurrency to 2 (Pi network bottleneck)
- Fill Frame.Description, Frame.Type, Frame.KeyText, Frame.Notable

### Step 8: Synthesis
- `pkg/openclaw/synthesis.go` — `Synthesize(ctx, meta, transcript, frames, cfg) (VideoSummary, error)`
- Combine all data, send to synthesis provider
- Handle transcript truncation for long videos (first 30% + last 20%)
- Parse JSON response into VideoSummary

### Step 9: Telegram delivery
- `pkg/openclaw/telegram.go` — `DeliverTelegram(ctx, summary, frames, cfg) error`
- Create telego bot with token from `channels.telegram.token`
- Send HTML-formatted summary message
- Send notable frame photos with captions
- Respect max_frames_in_digest limit

### Step 10: Obsidian output
- `pkg/openclaw/obsidian.go` — `WriteObsidian(summary, frames, cfg) error`
- New code (no existing Obsidian integration to reuse)
- Write markdown note with frontmatter
- Save notable JPEG frames to assets directory
- Wikilink embedded images

### Step 11: Orchestrator
- `pkg/openclaw/analyze.go` — `Analyze(ctx, url string, cfg Config) error`
- Coordinates all steps
- Uses errgroup for parallel transcript + frame extraction
- Sequential: metadata → (transcript || frames) → vision → synthesis → (telegram || obsidian)

### Step 12: CLI command
- `cmd/picoclaw/internal/openclaw/command.go`
- Register in `cmd/picoclaw/main.go`
- Flags: `--no-telegram`, `--no-obsidian`, `--scene-threshold`, `--synthesis-provider`, `--frames-only`

## Provider API Reference

### DashScope (Alibaba Cloud) — OpenAI-compatible

```
POST https://dashscope-intl.aliyuncs.com/compatible-mode/v1/chat/completions
Authorization: Bearer $DASHSCOPE_API_KEY
Content-Type: application/json

Vision:    model = "qwen3-vl-flash"
Synthesis: model = "qwen-plus"

Images: {"type":"image_url","image_url":{"url":"data:image/jpeg;base64,..."}}
```

### Moonshot (Kimi) — OpenAI-compatible

```
POST https://api.moonshot.ai/v1/chat/completions
Authorization: Bearer $MOONSHOT_API_KEY
Content-Type: application/json

Synthesis: model = "kimi-k2.5"
Temperature: 0.6 (recommended by Moonshot for instant mode)
```

## FFmpeg Command Reference

```bash
# Scene-change detection (default)
ffmpeg -i pipe:0 \
  -vf "select=gt(scene\,0.3),scale=1280:720:force_original_aspect_ratio=decrease" \
  -vsync vfr -f image2pipe -vcodec mjpeg -q:v 5 pipe:1

# Fixed interval fallback (every 10 seconds)
ffmpeg -i pipe:0 \
  -vf "fps=1/10,scale=1280:720:force_original_aspect_ratio=decrease" \
  -f image2pipe -vcodec mjpeg -q:v 5 pipe:1

# With showinfo for timestamps (parse pts_time from stderr)
ffmpeg -i pipe:0 \
  -vf "select=gt(scene\,0.3),showinfo,scale=1280:720:force_original_aspect_ratio=decrease" \
  -vsync vfr -f image2pipe -vcodec mjpeg -q:v 5 pipe:1
# stderr: [Parsed_showinfo...] n:0 pts:12345 pts_time:41.234
```

## Raspberry Pi Constraints

- **Memory:** Stream frames via channel (buffer 2-3). Don't accumulate all in memory.
- **CPU:** Scene detection on Pi 4: ~30-60s for 720p 10-min video. Acceptable.
- **Concurrency:** Transcript + frames in parallel (separate yt-dlp processes). Vision API: max 2 concurrent.
- **Disk:** Zero video files. Only final notable frames saved to Obsidian vault.
- **Peak memory target:** < 512MB

## Cost per Video

20-minute tech talk (~20 frames, ~3000 word transcript):
- Vision (qwen3-vl-flash): ~$0.01
- Synthesis (qwen-plus): ~$0.005
- **Total: ~$0.02**

## Definition of Done

- [ ] `picoclaw openclaw "URL"` produces a Telegram message with summary + key frames
- [ ] Obsidian note created with embedded frame images and wikilinks
- [ ] Works on Raspberry Pi 4 with < 512MB peak memory
- [ ] No video files written to disk at any point
- [ ] Config in existing `config.json`, env vars for API keys
- [ ] Telegram delivery works standalone (no gateway needed)
- [ ] Obsidian writer is new standalone code (no phantom dependency)
- [ ] Handles errors gracefully: no captions → skip transcript, ffmpeg fails → fixed interval fallback
- [ ] Tests for JPEG splitter, metadata parsing, transcript parsing
