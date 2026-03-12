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

const synthesisSystemPrompt = `You are a video content analyst. Given video metadata, transcript, and frame descriptions, produce a structured summary.

Respond ONLY with a JSON object (no markdown fences) with these fields:
- "title": concise title (may differ from original if clearer)
- "one_liner": single sentence summary
- "summary": 3-5 paragraph detailed summary
- "key_points": array of 3-7 key takeaways
- "topics": array of topic tags
- "audience": who this video is for
- "quality": brief assessment of production/content quality
- "action_items": array of actionable items (empty if none)
- "timestamps": array of {"time": "M:SS", "note": "description"} for notable moments`

const maxTranscriptChars = 15000

// Synthesize combines all extracted data into a structured summary via LLM.
func Synthesize(ctx context.Context, meta *VideoMeta, transcript []TranscriptLine, frames []Frame, cfg config.VideoAnalyzerProvider) (*VideoSummary, error) {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 120 * time.Second
	}

	provider := openai_compat.NewProvider(cfg.APIKey, cfg.BaseURL, "",
		openai_compat.WithRequestTimeout(timeout),
	)

	prompt := buildSynthesisPrompt(meta, transcript, frames)

	messages := []protocoltypes.Message{
		{Role: "system", Content: synthesisSystemPrompt},
		{Role: "user", Content: prompt},
	}

	resp, err := provider.Chat(ctx, messages, nil, cfg.Model, nil)
	if err != nil {
		return nil, fmt.Errorf("synthesis API call: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var summary VideoSummary
	if err := json.Unmarshal([]byte(content), &summary); err != nil {
		return nil, fmt.Errorf("parsing synthesis response: %w (content: %.200s)", err, content)
	}

	return &summary, nil
}

func buildSynthesisPrompt(meta *VideoMeta, transcript []TranscriptLine, frames []Frame) string {
	var b strings.Builder

	// Metadata section
	b.WriteString("## Video Metadata\n")
	b.WriteString(fmt.Sprintf("- Title: %s\n", meta.Title))
	b.WriteString(fmt.Sprintf("- Channel: %s\n", meta.Channel))
	b.WriteString(fmt.Sprintf("- Duration: %s\n", meta.DurationFormatted()))
	if meta.ViewCount > 0 {
		b.WriteString(fmt.Sprintf("- Views: %d\n", meta.ViewCount))
	}
	if len(meta.Categories) > 0 {
		b.WriteString(fmt.Sprintf("- Categories: %s\n", strings.Join(meta.Categories, ", ")))
	}
	if meta.Description != "" {
		desc := meta.Description
		if len(desc) > 500 {
			desc = desc[:500] + "..."
		}
		b.WriteString(fmt.Sprintf("- Description: %s\n", desc))
	}

	// Transcript section
	if len(transcript) > 0 {
		b.WriteString("\n## Transcript\n")
		transcriptText := formatTranscript(transcript)
		if len(transcriptText) > maxTranscriptChars {
			// Keep first 70% and last 30%
			firstPart := maxTranscriptChars * 70 / 100
			lastPart := maxTranscriptChars * 30 / 100
			transcriptText = transcriptText[:firstPart] + "\n\n[... transcript truncated ...]\n\n" + transcriptText[len(transcriptText)-lastPart:]
		}
		b.WriteString(transcriptText)
	}

	// Frame descriptions section
	b.WriteString("\n\n## Visual Frame Analysis\n")
	for _, frame := range frames {
		if frame.Description == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- [%s] %s", frame.TimestampFormatted(), frame.Description))
		if frame.Notable {
			b.WriteString(" [NOTABLE]")
		}
		if frame.KeyText != "" {
			b.WriteString(fmt.Sprintf(" Text: %q", frame.KeyText))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func formatTranscript(lines []TranscriptLine) string {
	var b strings.Builder
	for _, line := range lines {
		ts := formatTimestamp(line.Start)
		b.WriteString(fmt.Sprintf("[%s] %s\n", ts, line.Text))
	}
	return b.String()
}
