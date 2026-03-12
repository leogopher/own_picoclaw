package videoanalyzer

import (
	"context"
	"fmt"
	"os/exec"

	"golang.org/x/sync/errgroup"

	"github.com/rs/zerolog/log"

	"github.com/sipeed/picoclaw/pkg/config"
)

// Options controls which outputs to produce.
type Options struct {
	NoTelegram bool
	NoObsidian bool
	FramesOnly bool
}

// Analyze is the top-level orchestrator for video analysis.
func Analyze(ctx context.Context, url string, cfg config.VideoAnalyzerConfig, opts Options) (*AnalyzeResult, error) {
	// Validate URL
	if err := ValidateURL(url); err != nil {
		return nil, err
	}

	// Check external tool dependencies
	if err := checkDeps(cfg.Tools); err != nil {
		return nil, err
	}

	// Step 1: Get metadata
	log.Info().Str("url", url).Msg("fetching video metadata")
	meta, err := GetMetadata(ctx, url, cfg.Tools.YtdlpPath)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}
	log.Info().Str("title", meta.Title).Float64("duration", meta.Duration).Msg("metadata loaded")

	// Step 2: Extract transcript and frames in parallel
	var transcript []TranscriptLine
	var frames []Frame

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		log.Info().Msg("extracting transcript")
		t, err := GetTranscript(gCtx, url, cfg.Transcript, cfg.Tools.YtdlpPath)
		if err != nil {
			log.Warn().Err(err).Msg("transcript extraction failed (non-fatal)")
			return nil // transcript failure is non-fatal
		}
		transcript = t
		log.Info().Int("lines", len(t)).Msg("transcript extracted")
		return nil
	})

	g.Go(func() error {
		log.Info().Msg("extracting frames")
		f, err := ExtractFrames(gCtx, url, cfg.Frames, cfg.Tools)
		if err != nil {
			return fmt.Errorf("frame extraction: %w", err)
		}
		frames = f
		log.Info().Int("frames", len(f)).Msg("frames extracted")
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if opts.FramesOnly {
		return &AnalyzeResult{
			Meta:       meta,
			Transcript: transcript,
			Frames:     frames,
		}, nil
	}

	// Step 3: Vision analysis
	log.Info().Int("frames", len(frames)).Msg("running vision analysis")
	frames, err = AnalyzeFrames(ctx, frames, meta, cfg.Providers.Vision)
	if err != nil {
		return nil, fmt.Errorf("vision: %w", err)
	}

	// Step 4: Synthesis
	log.Info().Msg("synthesizing summary")
	summary, err := Synthesize(ctx, meta, transcript, frames, cfg.Providers.Synthesis)
	if err != nil {
		return nil, fmt.Errorf("synthesis: %w", err)
	}

	result := &AnalyzeResult{
		Meta:       meta,
		Transcript: transcript,
		Frames:     frames,
		Summary:    summary,
	}

	// Step 5: Deliver outputs in parallel
	og, oCtx := errgroup.WithContext(ctx)

	if !opts.NoTelegram && cfg.Telegram.BotToken != "" {
		og.Go(func() error {
			log.Info().Msg("delivering to Telegram")
			if err := DeliverTelegram(oCtx, result, cfg.Telegram); err != nil {
				return fmt.Errorf("telegram: %w", err)
			}
			log.Info().Msg("delivered to Telegram")
			return nil
		})
	}

	if !opts.NoObsidian && cfg.Obsidian.VaultPath != "" {
		og.Go(func() error {
			log.Info().Msg("writing Obsidian note")
			if err := WriteObsidian(result, cfg.Obsidian); err != nil {
				return fmt.Errorf("obsidian: %w", err)
			}
			log.Info().Msg("Obsidian note written")
			return nil
		})
	}

	if err := og.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

func checkDeps(tools config.VideoAnalyzerTools) error {
	if _, err := exec.LookPath(tools.YtdlpPath); err != nil {
		return fmt.Errorf("yt-dlp not found at %q: %w", tools.YtdlpPath, err)
	}
	if _, err := exec.LookPath(tools.FfmpegPath); err != nil {
		return fmt.Errorf("ffmpeg not found at %q: %w", tools.FfmpegPath, err)
	}
	return nil
}
