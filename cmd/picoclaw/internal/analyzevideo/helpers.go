package analyzevideo

import (
	"fmt"

	"github.com/sipeed/picoclaw/pkg/videoanalyzer"
)

func printResult(result *videoanalyzer.AnalyzeResult, opts videoanalyzer.Options) {
	meta := result.Meta

	fmt.Printf("Video: %s\n", meta.Title)
	fmt.Printf("Channel: %s\n", meta.Channel)
	fmt.Printf("Duration: %s\n", meta.DurationFormatted())
	fmt.Printf("Frames extracted: %d\n", len(result.Frames))

	if len(result.Transcript) > 0 {
		fmt.Printf("Transcript lines: %d\n", len(result.Transcript))
	}

	if opts.FramesOnly {
		fmt.Println("\n(frames-only mode — LLM analysis skipped)")
		return
	}

	if result.Summary != nil {
		fmt.Printf("\n--- Summary ---\n")
		fmt.Printf("%s\n\n", result.Summary.OneLiner)
		fmt.Println(result.Summary.Summary)

		if len(result.Summary.KeyPoints) > 0 {
			fmt.Println("\nKey Points:")
			for _, p := range result.Summary.KeyPoints {
				fmt.Printf("  • %s\n", p)
			}
		}
	}

	notableCount := 0
	for _, f := range result.Frames {
		if f.Notable {
			notableCount++
		}
	}
	if notableCount > 0 {
		fmt.Printf("\nNotable frames: %d\n", notableCount)
	}
}
