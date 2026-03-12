---
name: video-analyzer
description: Analyze YouTube videos — extract transcript, summarize key points via LLM, deliver structured digest. Use when user sends a YouTube link or asks to analyze/summarize a video.
metadata: {"nanobot":{"emoji":"🎬","requires":{"bins":["yt-dlp"]}}}
---

# Video Analyzer

Analyze YouTube videos using the built-in `analyze_video` tool.

## When to use

Use this skill immediately when:
- User sends a YouTube URL (youtube.com, youtu.be, m.youtube.com)
- User asks to "analyze", "summarize", or "review" a video
- User asks "what's this video about?"

## How to run

Use the `analyze_video` tool (NOT exec, NOT Python scripts):

```json
{"url": "VIDEO_URL"}
```

This extracts the transcript, summarizes via LLM, and sends the digest to Telegram. Takes ~30 seconds.

### CRITICAL RULES

- ALWAYS use the `analyze_video` tool — it is a native built-in tool
- Do NOT use `exec` to run shell commands
- Do NOT write Python scripts
- Do NOT use yt-dlp or ffmpeg directly
- The tool handles everything automatically
- After the tool returns, tell the user the analysis is complete

### Optional parameters

- `with_frames` (bool) — enable visual frame extraction + vision analysis (much slower)
- `no_telegram` (bool) — skip Telegram delivery

### Example

User sends: `https://www.youtube.com/watch?v=dQw4w9WgXcQ`

Call the `analyze_video` tool with:
```json
{"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ"}
```

## Supported URLs

- `https://www.youtube.com/watch?v=...`
- `https://youtu.be/...`
- `https://m.youtube.com/watch?v=...`
- `https://youtube.com/shorts/...`
