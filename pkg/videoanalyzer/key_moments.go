package videoanalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

const keyMomentsPrompt = `You are analyzing a video transcript to find the most important visual moments.

Given the transcript with timestamps, pick 5-15 timestamps where a screenshot would be most valuable — moments where:
- A new topic or concept is introduced
- Something is being demonstrated or shown
- A key conclusion or insight is stated
- The speaker references something visual ("as you can see", "look at this", "this diagram")

Respond ONLY with a JSON array of objects, no markdown fences:
[{"time": 42.5, "reason": "introduces the main concept"}]

Keep reasons short (under 10 words).`

type keyMoment struct {
	Time   float64 `json:"time"`
	Reason string  `json:"reason"`
}

// GetKeyMoments uses an LLM to identify important timestamps from the transcript.
func GetKeyMoments(ctx context.Context, transcript []TranscriptLine, meta *VideoMeta, cfg config.VideoAnalyzerProvider) ([]float64, error) {
	if len(transcript) == 0 {
		return nil, nil
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	provider := openai_compat.NewProvider(cfg.APIKey, cfg.BaseURL, "",
		openai_compat.WithRequestTimeout(timeout),
	)

	// Build transcript text for the prompt
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Video: %q (%s)\n\nTranscript:\n", meta.Title, meta.DurationFormatted()))
	for _, line := range transcript {
		ts := formatTimestamp(line.Start)
		b.WriteString(fmt.Sprintf("[%s] %s\n", ts, line.Text))
	}

	// Truncate if too long
	text := b.String()
	if len(text) > 12000 {
		text = text[:12000] + "\n[...truncated]"
	}

	messages := []protocoltypes.Message{
		{Role: "system", Content: keyMomentsPrompt},
		{Role: "user", Content: text},
	}

	resp, err := provider.Chat(ctx, messages, nil, cfg.Model, nil)
	if err != nil {
		return nil, fmt.Errorf("key moments API call: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var moments []keyMoment
	if err := json.Unmarshal([]byte(content), &moments); err != nil {
		return nil, fmt.Errorf("parsing key moments: %w (content: %.200s)", err, content)
	}

	timestamps := make([]float64, 0, len(moments))
	for _, m := range moments {
		if m.Time >= 0 && m.Time <= meta.Duration {
			timestamps = append(timestamps, m.Time)
		}
	}

	return timestamps, nil
}
