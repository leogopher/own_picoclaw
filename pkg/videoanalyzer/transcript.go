package videoanalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// json3Event is a single event in yt-dlp's json3 subtitle format.
type json3Event struct {
	TStartMs json.Number `json:"tStartMs"`
	DDurationMs json.Number `json:"dDurationMs"`
	Segs     []json3Seg  `json:"segs"`
}

// json3Seg is a text segment within a json3 event.
type json3Seg struct {
	UTF8 string `json:"utf8"`
}

// json3File is the top-level json3 subtitle structure.
type json3File struct {
	Events []json3Event `json:"events"`
}

// GetTranscript downloads subtitles via yt-dlp and parses them.
// Returns empty slice (not error) if no captions are available.
func GetTranscript(ctx context.Context, url string, cfg config.TranscriptConfig, ytdlpPath string) ([]TranscriptLine, error) {
	tmpDir, err := os.MkdirTemp("", "videoanalyzer-subs-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	langs := strings.Join(cfg.Languages, ",")
	outTmpl := filepath.Join(tmpDir, "subs.%(ext)s")

	args := []string{
		"--skip-download",
		"--write-subs",
		"--sub-langs", langs,
		"--sub-format", "json3",
		"-o", outTmpl,
		url,
	}
	if !cfg.FallbackASR {
		// Only manual subs, no auto-generated
	} else {
		args = append([]string{"--write-auto-subs"}, args...)
	}

	cmd := exec.CommandContext(ctx, ytdlpPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		// No captions is not an error
		if strings.Contains(string(out), "no subtitles") ||
			strings.Contains(string(out), "There are no subtitles") {
			return nil, nil
		}
		return nil, fmt.Errorf("yt-dlp subtitles: %s", strings.TrimSpace(string(out)))
	}

	// Find the downloaded json3 file
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("reading temp dir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json3") {
			data, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading subtitle file: %w", err)
			}
			return ParseJSON3(data)
		}
	}

	// No subtitle file found — no captions available
	return nil, nil
}

// ParseJSON3 parses yt-dlp's json3 subtitle format into TranscriptLines.
func ParseJSON3(data []byte) ([]TranscriptLine, error) {
	var file json3File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing json3 subtitles: %w", err)
	}

	var lines []TranscriptLine
	for _, event := range file.Events {
		var text strings.Builder
		for _, seg := range event.Segs {
			text.WriteString(seg.UTF8)
		}
		t := strings.TrimSpace(text.String())
		if t == "" {
			continue
		}

		startMs, _ := event.TStartMs.Int64()
		durMs, _ := event.DDurationMs.Int64()

		lines = append(lines, TranscriptLine{
			Start:    float64(startMs) / 1000.0,
			Duration: float64(durMs) / 1000.0,
			Text:     t,
		})
	}
	return lines, nil
}
