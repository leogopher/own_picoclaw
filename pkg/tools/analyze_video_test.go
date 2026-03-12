package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestAnalyzeVideoTool_Name(t *testing.T) {
	tool := NewAnalyzeVideoTool(config.VideoAnalyzerConfig{}, nil)
	assert.Equal(t, "analyze_video", tool.Name())
}

func TestAnalyzeVideoTool_Parameters(t *testing.T) {
	tool := NewAnalyzeVideoTool(config.VideoAnalyzerConfig{}, nil)
	params := tool.Parameters()

	props, ok := params["properties"].(map[string]any)
	assert.True(t, ok)
	assert.Contains(t, props, "url")
	assert.Contains(t, props, "with_frames")
	assert.Contains(t, props, "no_telegram")

	required, ok := params["required"].([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"url"}, required)
}

func TestAnalyzeVideoTool_MissingURL(t *testing.T) {
	tool := NewAnalyzeVideoTool(config.VideoAnalyzerConfig{}, nil)

	result := tool.Execute(context.Background(), map[string]any{})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "url is required")
}

func TestAnalyzeVideoTool_EmptyURL(t *testing.T) {
	tool := NewAnalyzeVideoTool(config.VideoAnalyzerConfig{}, nil)

	result := tool.Execute(context.Background(), map[string]any{"url": ""})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "url is required")
}

func TestAnalyzeVideoTool_InvalidURL(t *testing.T) {
	tool := NewAnalyzeVideoTool(config.VideoAnalyzerConfig{
		Tools: config.VideoAnalyzerTools{
			YtdlpPath:  "yt-dlp",
			FfmpegPath: "ffmpeg",
		},
	}, nil)

	result := tool.Execute(context.Background(), map[string]any{"url": "https://example.com/not-youtube"})
	assert.True(t, result.IsError)
	assert.Contains(t, result.ForLLM, "failed")
}

func TestAnalyzeVideoTool_ImplementsAsyncExecutor(t *testing.T) {
	var _ AsyncExecutor = (*AnalyzeVideoTool)(nil)
}
