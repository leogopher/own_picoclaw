package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/videoanalyzer"
)

// AnalyzeVideoTool wraps the videoanalyzer pipeline as a native agent tool.
// It implements AsyncExecutor so the agent can continue chatting while analysis runs.
type AnalyzeVideoTool struct {
	cfg     config.VideoAnalyzerConfig
	fullCfg *config.Config
}

// Compile-time check: AnalyzeVideoTool implements AsyncExecutor.
var _ AsyncExecutor = (*AnalyzeVideoTool)(nil)

func NewAnalyzeVideoTool(cfg config.VideoAnalyzerConfig, fullCfg *config.Config) *AnalyzeVideoTool {
	return &AnalyzeVideoTool{cfg: cfg, fullCfg: fullCfg}
}

func (t *AnalyzeVideoTool) Name() string {
	return "analyze_video"
}

func (t *AnalyzeVideoTool) Description() string {
	return "Analyze a YouTube video: extract transcript, summarize key points via LLM, and deliver a structured digest to Telegram. Use when a user sends a YouTube URL or asks to analyze/summarize a video. Takes ~30 seconds."
}

func (t *AnalyzeVideoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "YouTube video URL (youtube.com, youtu.be, m.youtube.com, shorts)",
			},
			"with_frames": map[string]any{
				"type":        "boolean",
				"description": "Enable visual frame extraction + vision analysis (much slower, 5-10 min). Default: false",
			},
			"no_telegram": map[string]any{
				"type":        "boolean",
				"description": "Skip sending the digest to Telegram. Default: false",
			},
		},
		"required": []string{"url"},
	}
}

func (t *AnalyzeVideoTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	return t.execute(ctx, args, nil)
}

func (t *AnalyzeVideoTool) ExecuteAsync(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	return t.execute(ctx, args, cb)
}

func (t *AnalyzeVideoTool) execute(ctx context.Context, args map[string]any, cb AsyncCallback) *ToolResult {
	url, ok := args["url"].(string)
	if !ok || strings.TrimSpace(url) == "" {
		return ErrorResult("url is required and must be a non-empty string")
	}

	withFrames, _ := args["with_frames"].(bool)
	noTelegram, _ := args["no_telegram"].(bool)

	opts := videoanalyzer.Options{
		WithFrames: withFrames || t.cfg.WithFrames,
		NoTelegram: noTelegram,
	}

	if cb != nil {
		// Async mode: run in background, call back when done
		go func() {
			result := t.runAnalysis(ctx, url, opts)
			cb(ctx, result)
		}()
		return AsyncResult(fmt.Sprintf("Video analysis started for %s. I'll report back when it's done (~30 seconds).", url))
	}

	// Sync mode
	return t.runAnalysis(ctx, url, opts)
}

func (t *AnalyzeVideoTool) runAnalysis(ctx context.Context, url string, opts videoanalyzer.Options) *ToolResult {
	log.Info().Str("url", url).Bool("with_frames", opts.WithFrames).Msg("analyze_video tool: starting")

	result, err := videoanalyzer.Analyze(ctx, url, t.cfg, t.fullCfg, opts)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("analyze_video tool: failed")
		return ErrorResult(fmt.Sprintf("Video analysis failed: %v", err)).WithError(err)
	}

	// Build a summary for the LLM
	var sb strings.Builder
	sb.WriteString("Video analysis complete!\n\n")

	if result.Summary != nil {
		sb.WriteString(fmt.Sprintf("**%s**\n", result.Summary.Title))
		sb.WriteString(fmt.Sprintf("%s\n\n", result.Summary.OneLiner))

		if result.Summary.Summary != "" {
			sb.WriteString(fmt.Sprintf("Summary: %s\n\n", result.Summary.Summary))
		}

		if len(result.Summary.KeyPoints) > 0 {
			sb.WriteString("Key points:\n")
			for _, kp := range result.Summary.KeyPoints {
				sb.WriteString(fmt.Sprintf("- %s\n", kp))
			}
			sb.WriteString("\n")
		}

		if len(result.Summary.Topics) > 0 {
			sb.WriteString(fmt.Sprintf("Topics: %s\n", strings.Join(result.Summary.Topics, ", ")))
		}
	}

	if !opts.NoTelegram {
		sb.WriteString("\nDigest sent to Telegram.")
	}

	log.Info().Str("url", url).Msg("analyze_video tool: complete")
	return NewToolResult(sb.String())
}
