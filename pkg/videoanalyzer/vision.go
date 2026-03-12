package videoanalyzer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers/openai_compat"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

const visionPrompt = `You are analyzing frames from a YouTube video. For each frame, provide a JSON array with one object per frame in order.

Each object must have:
- "description": 1-2 sentence description of what is shown
- "notable": true if this frame shows something especially important (key diagram, title slide, demo, code, important visual)
- "key_text": any text visible in the frame (empty string if none)

Respond ONLY with a JSON array, no markdown fences.`

type frameAnalysis struct {
	Description string `json:"description"`
	Notable     bool   `json:"notable"`
	KeyText     string `json:"key_text"`
}

// AnalyzeFrames sends frame batches to the vision LLM and fills descriptions.
func AnalyzeFrames(ctx context.Context, frames []Frame, meta *VideoMeta, cfg config.VideoAnalyzerProvider) ([]Frame, error) {
	if len(frames) == 0 {
		return frames, nil
	}

	batchSize := cfg.MaxFramesPerBatch
	if batchSize <= 0 {
		batchSize = 4
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	provider := openai_compat.NewProvider(cfg.APIKey, cfg.BaseURL, "",
		openai_compat.WithRequestTimeout(timeout),
	)

	// Build batches
	var batches [][]int
	for i := 0; i < len(frames); i += batchSize {
		end := i + batchSize
		if end > len(frames) {
			end = len(frames)
		}
		batch := make([]int, end-i)
		for j := range batch {
			batch[j] = i + j
		}
		batches = append(batches, batch)
	}

	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(2) // Max 2 concurrent API calls

	for _, batch := range batches {
		batch := batch
		g.Go(func() error {
			analyses, err := analyzeBatch(gCtx, frames, batch, meta, provider, cfg.Model)
			if err != nil {
				return err
			}
			mu.Lock()
			defer mu.Unlock()
			for i, idx := range batch {
				if i < len(analyses) {
					frames[idx].Description = analyses[i].Description
					frames[idx].Notable = analyses[i].Notable
					frames[idx].KeyText = analyses[i].KeyText
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("vision analysis: %w", err)
	}

	return frames, nil
}

func analyzeBatch(ctx context.Context, frames []Frame, indices []int, meta *VideoMeta, provider *openai_compat.Provider, model string) ([]frameAnalysis, error) {
	// Build media list with base64-encoded JPEGs
	var media []string
	for _, idx := range indices {
		encoded := base64.StdEncoding.EncodeToString(frames[idx].JPEG)
		dataURI := "data:image/jpeg;base64," + encoded
		media = append(media, dataURI)
	}

	// Build prompt with context
	var prompt strings.Builder
	prompt.WriteString(visionPrompt)
	prompt.WriteString(fmt.Sprintf("\n\nVideo: %q by %s (%s)", meta.Title, meta.Uploader, meta.DurationFormatted()))
	prompt.WriteString(fmt.Sprintf("\n\nAnalyzing %d frames:", len(indices)))
	for i, idx := range indices {
		prompt.WriteString(fmt.Sprintf("\n- Frame %d at %s", i+1, frames[idx].TimestampFormatted()))
	}

	messages := []protocoltypes.Message{
		{Role: "user", Content: prompt.String(), Media: media},
	}

	resp, err := provider.Chat(ctx, messages, nil, model, nil)
	if err != nil {
		return nil, fmt.Errorf("vision API call: %w", err)
	}

	// Parse JSON response
	content := strings.TrimSpace(resp.Content)
	// Strip markdown fences if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var analyses []frameAnalysis
	if err := json.Unmarshal([]byte(content), &analyses); err != nil {
		return nil, fmt.Errorf("parsing vision response: %w (content: %.200s)", err, content)
	}

	return analyses, nil
}
