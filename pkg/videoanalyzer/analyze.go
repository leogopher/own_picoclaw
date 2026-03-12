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
	NoTelegram     bool
	NoObsidian     bool
	FramesOnly     bool
	WithFrames     bool // enable frame extraction + vision (slow, off by default)
}

// Analyze is the top-level orchestrator for video analysis.
// fullCfg is the full picoclaw config, used to resolve provider credentials
// from model_list and Telegram token from channels.telegram.
func Analyze(ctx context.Context, url string, cfg config.VideoAnalyzerConfig, fullCfg *config.Config, opts Options) (*AnalyzeResult, error) {
	if fullCfg != nil {
		// Resolve vision provider from model_list if api_key/base_url not set
		resolveProvider(&cfg.Providers.Vision, fullCfg.ModelList)
		resolveProvider(&cfg.Providers.Synthesis, fullCfg.ModelList)

		// Fall back to channels.telegram for bot token and chat_id
		tg := fullCfg.Channels.Telegram
		if cfg.Telegram.BotToken == "" {
			cfg.Telegram.BotToken = tg.Token
		}
		if cfg.Telegram.ChatID == "" && len(tg.AllowFrom) > 0 {
			cfg.Telegram.ChatID = tg.AllowFrom[0]
		}
	}
	// Validate URL
	if err := ValidateURL(url); err != nil {
		return nil, err
	}

	// Check external tool dependencies
	if err := checkDeps(cfg.Tools, opts.WithFrames || opts.FramesOnly); err != nil {
		return nil, err
	}

	// Step 1: Get metadata
	log.Info().Str("url", url).Msg("fetching video metadata")
	meta, err := GetMetadata(ctx, url, cfg.Tools.YtdlpPath)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}
	log.Info().Str("title", meta.Title).Float64("duration", meta.Duration).Msg("metadata loaded")

	// Step 2: Extract transcript
	log.Info().Msg("extracting transcript")
	transcript, err := GetTranscript(ctx, url, cfg.Transcript, cfg.Tools.YtdlpPath)
	if err != nil {
		log.Warn().Err(err).Msg("transcript extraction failed (non-fatal)")
		transcript = nil
	} else {
		log.Info().Int("lines", len(transcript)).Msg("transcript extracted")
	}

	// Step 3: Frame extraction (only with --with-frames flag)
	var frames []Frame
	if opts.WithFrames || opts.FramesOnly {
		var keyTimestamps []float64
		if len(transcript) > 0 {
			log.Info().Msg("identifying key moments from transcript")
			keyTimestamps, err = GetKeyMoments(ctx, transcript, meta, cfg.Providers.Synthesis)
			if err != nil {
				log.Warn().Err(err).Msg("key moments extraction failed (non-fatal)")
			} else {
				log.Info().Int("moments", len(keyTimestamps)).Msg("key moments identified")
			}
		}

		log.Info().Int("key_timestamps", len(keyTimestamps)).Msg("extracting frames")
		frames, err = ExtractFrames(ctx, url, cfg.Frames, cfg.Tools, keyTimestamps)
		if err != nil {
			return nil, fmt.Errorf("frame extraction: %w", err)
		}
		log.Info().Int("frames", len(frames)).Msg("frames extracted")

		if opts.FramesOnly {
			return &AnalyzeResult{
				Meta:       meta,
				Transcript: transcript,
				Frames:     frames,
			}, nil
		}

		// Vision analysis on frames
		log.Info().Int("frames", len(frames)).Msg("running vision analysis")
		frames, err = AnalyzeFrames(ctx, frames, meta, cfg.Providers.Vision)
		if err != nil {
			return nil, fmt.Errorf("vision: %w", err)
		}
	}

	// Step 4: Synthesis (works with transcript only, or transcript + frames)
	log.Info().Bool("has_transcript", len(transcript) > 0).Int("frames", len(frames)).Msg("synthesizing summary")
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

	if !opts.NoTelegram && cfg.Telegram.Enabled {
		if cfg.Telegram.ChatID == "" {
			return nil, fmt.Errorf("telegram is enabled but chat_id is not set")
		}
		if cfg.Telegram.BotToken == "" {
			return nil, fmt.Errorf("telegram is enabled but no bot token found (set video_analyzer.telegram.bot_token or channels.telegram.token)")
		}
		og.Go(func() error {
			log.Info().Msg("delivering to Telegram")
			if err := DeliverTelegram(oCtx, result, cfg.Telegram); err != nil {
				return fmt.Errorf("telegram: %w", err)
			}
			log.Info().Msg("delivered to Telegram")
			return nil
		})
	}

	if !opts.NoObsidian && cfg.Obsidian.Enabled && cfg.Obsidian.VaultPath != "" {
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

// resolveProvider fills in missing APIKey/BaseURL from model_list by matching
// the provider's Model against model_name entries.
func resolveProvider(p *config.VideoAnalyzerProvider, modelList []config.ModelConfig) {
	if p.APIKey != "" && p.BaseURL != "" {
		return // already fully configured
	}
	for _, m := range modelList {
		if m.ModelName == p.Model {
			if p.APIKey == "" {
				p.APIKey = m.APIKey
			}
			if p.BaseURL == "" {
				p.BaseURL = m.APIBase
			}
			return
		}
	}
}

func checkDeps(tools config.VideoAnalyzerTools, needFFmpeg bool) error {
	if _, err := exec.LookPath(tools.YtdlpPath); err != nil {
		return fmt.Errorf("yt-dlp not found at %q: %w", tools.YtdlpPath, err)
	}
	if needFFmpeg {
		if _, err := exec.LookPath(tools.FfmpegPath); err != nil {
			return fmt.Errorf("ffmpeg not found at %q: %w", tools.FfmpegPath, err)
		}
	}
	return nil
}
