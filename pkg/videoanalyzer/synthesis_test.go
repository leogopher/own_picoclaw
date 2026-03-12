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

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"clean", `{"a":1}`, `{"a":1}`},
		{"markdown fences", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"surrounding text", "Here is the JSON:\n{\"a\":1}\nDone!", `{"a":1}`},
		{"nested braces", `{"a":{"b":2}}`, `{"a":{"b":2}}`},
		{"array input", `[{"a":1}]`, `[{"a":1}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSON(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRepairJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"trailing comma in object",
			`{"a": 1, "b": 2,}`,
			`{"a": 1, "b": 2}`,
		},
		{
			"trailing comma in array",
			`[1, 2, 3,]`,
			`[1, 2, 3]`,
		},
		{
			"trailing comma with whitespace",
			"{\"a\": 1,\n  }",
			"{\"a\": 1\n  }",
		},
		{
			"no trailing comma",
			`{"a": 1, "b": 2}`,
			`{"a": 1, "b": 2}`,
		},
		{
			"comma inside string preserved",
			`{"a": "hello, world"}`,
			`{"a": "hello, world"}`,
		},
		{
			"nested trailing commas",
			`{"a": [1, 2,], "b": {"c": 3,},}`,
			`{"a": [1, 2], "b": {"c": 3}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repairJSON(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
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
