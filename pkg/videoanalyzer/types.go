package videoanalyzer

import (
	"fmt"
	"time"
)

// VideoMeta holds metadata extracted from yt-dlp --dump-json.
type VideoMeta struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Duration    float64 `json:"duration"`
	Uploader    string  `json:"uploader"`
	UploadDate  string  `json:"upload_date"`
	ViewCount   int64   `json:"view_count"`
	LikeCount   int64   `json:"like_count"`
	Channel     string  `json:"channel"`
	ChannelURL  string  `json:"channel_url"`
	WebpageURL  string  `json:"webpage_url"`
	Thumbnail   string  `json:"thumbnail"`
	Categories  []string `json:"categories"`
	Tags        []string `json:"tags"`
}

// DurationFormatted returns the duration as a human-readable string.
func (m *VideoMeta) DurationFormatted() string {
	d := time.Duration(m.Duration * float64(time.Second))
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, min, sec)
	}
	return fmt.Sprintf("%dm%02ds", min, sec)
}

// Frame represents a single extracted video frame.
type Frame struct {
	Index       int     `json:"index"`
	Timestamp   float64 `json:"timestamp"`
	JPEG        []byte  `json:"-"`
	Description string  `json:"description,omitempty"`
	Notable     bool    `json:"notable,omitempty"`
	KeyText     string  `json:"key_text,omitempty"`
}

// TimestampFormatted returns the timestamp as MM:SS or HH:MM:SS.
func (f *Frame) TimestampFormatted() string {
	total := int(f.Timestamp)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// TranscriptLine is a single subtitle entry.
type TranscriptLine struct {
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
	Text     string  `json:"text"`
}

// VideoSummary is the synthesis LLM output.
type VideoSummary struct {
	Title        string   `json:"title"`
	OneLiner     string   `json:"one_liner"`
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	Topics       []string `json:"topics"`
	Audience     string   `json:"audience"`
	Quality      string   `json:"quality"`
	ActionItems  []string `json:"action_items,omitempty"`
	Timestamps   []TimestampNote `json:"timestamps,omitempty"`
}

// TimestampNote links a timestamp to a notable moment.
type TimestampNote struct {
	Time string `json:"time"`
	Note string `json:"note"`
}

// AnalyzeResult bundles everything produced by the pipeline.
type AnalyzeResult struct {
	Meta       *VideoMeta      `json:"meta"`
	Transcript []TranscriptLine `json:"transcript,omitempty"`
	Frames     []Frame         `json:"frames"`
	Summary    *VideoSummary   `json:"summary"`
}
