package videoanalyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSynthesisPrompt(t *testing.T) {
	meta := &VideoMeta{
		Title:       "Go Concurrency Patterns",
		Channel:     "GopherCon",
		Duration:    1800,
		ViewCount:   50000,
		Categories:  []string{"Education"},
		Description: "A talk about Go concurrency",
	}

	transcript := []TranscriptLine{
		{Start: 0, Duration: 5, Text: "Hello everyone"},
		{Start: 5, Duration: 5, Text: "Today we'll talk about concurrency"},
	}

	frames := []Frame{
		{Index: 0, Timestamp: 10, Description: "Title slide", Notable: true, KeyText: "Go Concurrency"},
		{Index: 1, Timestamp: 60, Description: "Code example", Notable: false},
	}

	prompt := buildSynthesisPrompt(meta, transcript, frames)

	assert.Contains(t, prompt, "Go Concurrency Patterns")
	assert.Contains(t, prompt, "GopherCon")
	assert.Contains(t, prompt, "30m00s")
	assert.Contains(t, prompt, "Hello everyone")
	assert.Contains(t, prompt, "Title slide")
	assert.Contains(t, prompt, "[NOTABLE]")
	assert.Contains(t, prompt, `"Go Concurrency"`)
}

func TestBuildSynthesisPrompt_TranscriptTruncation(t *testing.T) {
	meta := &VideoMeta{Title: "Test", Duration: 100}

	// Create a very long transcript
	var lines []TranscriptLine
	for i := 0; i < 1000; i++ {
		lines = append(lines, TranscriptLine{
			Start:    float64(i),
			Duration: 1,
			Text:     strings.Repeat("word ", 20),
		})
	}

	prompt := buildSynthesisPrompt(meta, lines, nil)
	assert.Contains(t, prompt, "[... transcript truncated ...]")
}

func TestFormatTranscript(t *testing.T) {
	lines := []TranscriptLine{
		{Start: 0, Duration: 5, Text: "Hello"},
		{Start: 65, Duration: 5, Text: "World"},
	}

	result := formatTranscript(lines)
	assert.Contains(t, result, "[0:00] Hello")
	assert.Contains(t, result, "[1:05] World")
}
