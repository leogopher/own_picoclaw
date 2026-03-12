package analyzevideo

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/videoanalyzer"
)

// NewAnalyzeVideoCommand creates the analyze-video cobra command.
func NewAnalyzeVideoCommand() *cobra.Command {
	var (
		noTelegram     bool
		noObsidian     bool
		withFrames     bool
		framesOnly     bool
		sceneThreshold float64
		maxFrames      int
		debug          bool
	)

	cmd := &cobra.Command{
		Use:   "analyze-video URL",
		Short: "Analyze a YouTube video via transcript and LLM synthesis",
		Long: `Analyze a YouTube video by extracting the transcript,
synthesizing a structured summary via LLM, and delivering
results to Telegram and/or Obsidian.

By default uses transcript only (fast, ~30s).
Add --with-frames for visual frame analysis (slower, uses vision LLM).

Requires yt-dlp (and ffmpeg if using --with-frames).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if debug {
				logger.SetLevel(logger.DEBUG)
			}

			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// Apply flag overrides
			vaCfg := cfg.VideoAnalyzer
			if sceneThreshold > 0 {
				vaCfg.Frames.SceneThreshold = sceneThreshold
			}
			if maxFrames > 0 {
				vaCfg.Frames.MaxFrames = maxFrames
			}

			opts := videoanalyzer.Options{
				NoTelegram: noTelegram,
				NoObsidian: noObsidian,
				WithFrames: withFrames || vaCfg.WithFrames, // CLI flag or config
				FramesOnly: framesOnly,
			}

			result, err := videoanalyzer.Analyze(cmd.Context(), args[0], vaCfg, cfg, opts)
			if err != nil {
				return err
			}

			printResult(result, opts)
			return nil
		},
	}

	cmd.Flags().BoolVar(&noTelegram, "no-telegram", false, "Skip Telegram delivery")
	cmd.Flags().BoolVar(&noObsidian, "no-obsidian", false, "Skip Obsidian note creation")
	cmd.Flags().BoolVar(&withFrames, "with-frames", false, "Enable frame extraction + vision analysis (slower)")
	cmd.Flags().BoolVar(&framesOnly, "frames-only", false, "Extract frames only, no LLM (implies --with-frames)")
	cmd.Flags().Float64Var(&sceneThreshold, "scene-threshold", 0, "Override scene detection threshold (0.0-1.0)")
	cmd.Flags().IntVar(&maxFrames, "max-frames", 0, "Override maximum number of frames to extract")
	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug logging")

	return cmd
}
