---
name: video-analyzer
description: Analyze YouTube videos — extract transcript, summarize key points via LLM, deliver structured digest. Use when user sends a YouTube link or asks to analyze/summarize a video.
metadata: {"nanobot":{"emoji":"🎬","requires":{"bins":["picoclaw","yt-dlp"]}}}
---

# Video Analyzer

Analyze YouTube videos using the built-in `picoclaw analyze-video` command.

## When to use

Use this skill immediately when:
- User sends a YouTube URL (youtube.com, youtu.be, m.youtube.com)
- User asks to "analyze", "summarize", or "review" a video
- User asks "what's this video about?"

## How to run

Use the `exec` tool to run:

```bash
picoclaw analyze-video "VIDEO_URL"
```

This extracts the transcript, synthesizes a structured summary via LLM, and sends the result to Telegram automatically. The command takes ~30 seconds.

### Important

- Always pass the full URL in quotes
- Do NOT try to use Python, yt-dlp directly, or any other scripts — use `picoclaw analyze-video` only
- The command handles everything: transcript extraction, LLM synthesis, Telegram delivery
- After running, tell the user the analysis is complete and the digest was sent to Telegram

### Optional flags

- `--no-telegram` — skip Telegram delivery, print result to stdout only
- `--no-obsidian` — skip Obsidian note creation
- `--with-frames` — enable visual frame extraction + vision analysis (much slower, ~5-10 min)
- `--debug` — enable debug logging

### Example

User sends: `https://www.youtube.com/watch?v=dQw4w9WgXcQ`

Run:
```bash
picoclaw analyze-video "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```

Then respond: "Video analyzed! A digest with the summary has been sent to Telegram."

## Supported URLs

- `https://www.youtube.com/watch?v=...`
- `https://youtu.be/...`
- `https://m.youtube.com/watch?v=...`
- `https://youtube.com/shorts/...`
