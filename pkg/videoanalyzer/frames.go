package videoanalyzer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"github.com/sipeed/picoclaw/pkg/config"
)

var ptsTimePattern = regexp.MustCompile(`pts_time:(\d+\.?\d*)`)

// jpegSOI is the JPEG Start Of Image marker.
var jpegSOI = []byte{0xFF, 0xD8}

// jpegEOI is the JPEG End Of Image marker.
var jpegEOI = []byte{0xFF, 0xD9}

// ExtractFrames streams video through yt-dlp | ffmpeg and extracts JPEG frames.
// keyTimestamps are LLM-selected important moments from the transcript.
// The filter combines: scene detection + key timestamps + fixed interval fallback.
func ExtractFrames(ctx context.Context, url string, cfg config.FrameConfig, tools config.VideoAnalyzerTools, keyTimestamps []float64) ([]Frame, error) {
	vf := buildCombinedFilter(cfg.SceneThreshold, keyTimestamps)

	frames, err := extractWithFilter(ctx, url, cfg, tools, vf)
	if err != nil {
		return nil, err
	}

	// Fallback to fixed interval if we got too few frames
	if len(frames) < 3 {
		frames, err = extractWithFilter(ctx, url, cfg, tools, "fps=1/10,showinfo")
		if err != nil {
			return nil, err
		}
	}

	// Deduplicate frames that are too close together (within 2 seconds)
	frames = deduplicateFrames(frames, 2.0)

	if len(frames) > cfg.MaxFrames {
		frames = frames[:cfg.MaxFrames]
	}

	return frames, nil
}

// buildCombinedFilter creates an ffmpeg select filter that combines:
// - Scene detection: gt(scene,threshold)
// - Key timestamps: between(t,T-0.5,T+0.5) for each LLM-selected moment
// - Fixed interval: isnan(prev_selected_t)+gte(t-prev_selected_t,30) as safety net
func buildCombinedFilter(sceneThreshold float64, keyTimestamps []float64) string {
	var parts []string

	// Scene detection
	parts = append(parts, fmt.Sprintf("gt(scene\\,%g)", sceneThreshold))

	// Key timestamps from transcript analysis (±0.5s window)
	for _, ts := range keyTimestamps {
		parts = append(parts, fmt.Sprintf("between(t\\,%g\\,%g)", ts-0.5, ts+0.5))
	}

	// Fixed interval safety net: every 30s if nothing else selected
	parts = append(parts, "isnan(prev_selected_t)+gte(t-prev_selected_t\\,30)")

	selectExpr := "select='" + joinFilter(parts) + "'"
	return selectExpr + ",showinfo"
}

func joinFilter(parts []string) string {
	if len(parts) == 0 {
		return "1"
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += "+" + p
	}
	return result
}

// deduplicateFrames removes frames that are within minGap seconds of each other.
func deduplicateFrames(frames []Frame, minGap float64) []Frame {
	if len(frames) <= 1 {
		return frames
	}
	result := []Frame{frames[0]}
	for i := 1; i < len(frames); i++ {
		if frames[i].Timestamp-result[len(result)-1].Timestamp >= minGap {
			result = append(result, frames[i])
		}
	}
	return result
}

func extractWithFilter(ctx context.Context, url string, cfg config.FrameConfig, tools config.VideoAnalyzerTools, vf string) ([]Frame, error) {
	scale := fmt.Sprintf("scale=-1:%d:force_original_aspect_ratio=decrease", cfg.Resolution)
	fullVF := vf + "," + scale

	// Start yt-dlp process
	ytdlp := exec.CommandContext(ctx, tools.YtdlpPath,
		"-f", fmt.Sprintf("bestvideo[height<=%d]/best[height<=%d]", cfg.Resolution, cfg.Resolution),
		"-o", "-",
		url,
	)
	ytdlpOut, err := ytdlp.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp stdout pipe: %w", err)
	}
	if err := ytdlp.Start(); err != nil {
		return nil, fmt.Errorf("starting yt-dlp: %w", err)
	}

	// Start ffmpeg process, reading from yt-dlp stdout
	ffmpeg := exec.CommandContext(ctx, tools.FfmpegPath,
		"-i", "pipe:0",
		"-vf", fullVF,
		"-vsync", "vfr",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", strconv.Itoa(jpegQualityToFFmpeg(cfg.JPEGQuality)),
		"pipe:1",
	)
	ffmpeg.Stdin = ytdlpOut

	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		_ = ytdlp.Process.Kill()
		return nil, fmt.Errorf("ffmpeg stdout pipe: %w", err)
	}

	ffmpegStderr, err := ffmpeg.StderrPipe()
	if err != nil {
		_ = ytdlp.Process.Kill()
		return nil, fmt.Errorf("ffmpeg stderr pipe: %w", err)
	}

	if err := ffmpeg.Start(); err != nil {
		_ = ytdlp.Process.Kill()
		return nil, fmt.Errorf("starting ffmpeg: %w", err)
	}

	// Parse timestamps from ffmpeg stderr in background
	timestamps := make(chan float64, cfg.MaxFrames)
	go func() {
		parseTimestamps(ffmpegStderr, timestamps)
		close(timestamps)
	}()

	// Split JPEG frames from ffmpeg stdout
	frames, err := splitJPEGs(ffmpegOut, cfg.MaxFrames)

	// Clean up processes — this also closes stderr, unblocking parseTimestamps
	_ = ffmpeg.Wait()
	_ = ytdlp.Wait()

	// Now safe to drain — parseTimestamps has exited and closed the channel
	tsSlice := drainTimestamps(timestamps)
	for i := range frames {
		frames[i].Index = i
		if i < len(tsSlice) {
			frames[i].Timestamp = tsSlice[i]
		}
	}

	if err != nil {
		return nil, err
	}

	return frames, nil
}

// parseTimestamps reads ffmpeg stderr for showinfo pts_time values.
func parseTimestamps(r io.Reader, ch chan<- float64) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if m := ptsTimePattern.FindStringSubmatch(line); len(m) == 2 {
			if ts, err := strconv.ParseFloat(m[1], 64); err == nil {
				ch <- ts
			}
		}
	}
}

func drainTimestamps(ch <-chan float64) []float64 {
	var out []float64
	for ts := range ch {
		out = append(out, ts)
	}
	return out
}

// splitJPEGs reads a stream of concatenated JPEGs and splits them by SOI/EOI markers.
func splitJPEGs(r io.Reader, maxFrames int) ([]Frame, error) {
	buf := make([]byte, 0, 256*1024)
	readBuf := make([]byte, 32*1024)
	var frames []Frame

	for len(frames) < maxFrames {
		n, err := r.Read(readBuf)
		if n > 0 {
			buf = append(buf, readBuf[:n]...)

			// Try to extract complete JPEGs
			for len(frames) < maxFrames {
				frame, rest, ok := extractOneJPEG(buf)
				if !ok {
					break
				}
				frames = append(frames, Frame{
					JPEG: frame,
				})
				buf = rest
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return frames, fmt.Errorf("reading frame data: %w", err)
		}
	}

	return frames, nil
}

// extractOneJPEG finds the first complete JPEG in buf (SOI to EOI).
func extractOneJPEG(buf []byte) (jpeg, rest []byte, ok bool) {
	soiIdx := bytes.Index(buf, jpegSOI)
	if soiIdx < 0 {
		return nil, buf, false
	}

	// Search for EOI after SOI
	eoiIdx := bytes.Index(buf[soiIdx+2:], jpegEOI)
	if eoiIdx < 0 {
		return nil, buf, false
	}
	eoiIdx += soiIdx + 2 // absolute position

	end := eoiIdx + 2
	jpeg = make([]byte, end-soiIdx)
	copy(jpeg, buf[soiIdx:end])

	return jpeg, buf[end:], true
}

// jpegQualityToFFmpeg converts quality percentage (85) to ffmpeg q:v scale (2-31, lower=better).
func jpegQualityToFFmpeg(quality int) int {
	if quality <= 0 {
		quality = 85
	}
	if quality > 100 {
		quality = 100
	}
	// Map 100→2, 1→31
	return 2 + (100-quality)*29/99
}

// formatTimestamp formats seconds as "1:23" or "1:02:03".
func formatTimestamp(seconds float64) string {
	total := int(seconds)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
