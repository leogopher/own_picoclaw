package videoanalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var youtubeURLPattern = regexp.MustCompile(
	`^https?://(?:www\.|m\.)?(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/shorts/)[\w-]+`,
)

// ValidateURL checks that the URL looks like a YouTube video URL.
func ValidateURL(url string) error {
	if !youtubeURLPattern.MatchString(url) {
		return fmt.Errorf("not a valid YouTube URL: %s", url)
	}
	return nil
}

// GetMetadata calls yt-dlp --dump-json to extract video metadata.
func GetMetadata(ctx context.Context, url, ytdlpPath string) (*VideoMeta, error) {
	cmd := exec.CommandContext(ctx, ytdlpPath, "--dump-json", "--no-download", url)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp metadata failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("running yt-dlp: %w", err)
	}

	var meta VideoMeta
	if err := json.Unmarshal(out, &meta); err != nil {
		return nil, fmt.Errorf("parsing yt-dlp JSON: %w", err)
	}

	return &meta, nil
}
