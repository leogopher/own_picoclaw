package videoanalyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTelegramHTML(t *testing.T) {
	result := &AnalyzeResult{
		Meta: &VideoMeta{
			Title:      "Test Video",
			Channel:    "Test Channel",
			Duration:   600,
			WebpageURL: "https://youtube.com/watch?v=abc",
		},
		Summary: &VideoSummary{
			Title:    "Test Summary",
			OneLiner: "A short summary",
			Summary:  "Detailed summary here.",
			KeyPoints: []string{
				"Point one",
				"Point two",
			},
			Topics: []string{"go", "testing"},
		},
	}

	html := formatTelegramHTML(result)

	assert.Contains(t, html, "<b>Test Summary</b>")
	assert.Contains(t, html, "<i>A short summary</i>")
	assert.Contains(t, html, "Test Channel")
	assert.Contains(t, html, "10m00s")
	assert.Contains(t, html, "Point one")
	assert.Contains(t, html, "go, testing")
}

func TestEscapeHTML(t *testing.T) {
	assert.Equal(t, "a &amp; b", escapeHTML("a & b"))
	assert.Equal(t, "&lt;script&gt;", escapeHTML("<script>"))
}

func TestSplitMessage(t *testing.T) {
	// Short message — no split needed
	parts := splitMessage("hello", 100)
	assert.Len(t, parts, 1)
	assert.Equal(t, "hello", parts[0])

	// Long message — splits at newline
	msg := "line1\nline2\nline3\nline4"
	parts = splitMessage(msg, 12)
	assert.True(t, len(parts) > 1)
	for _, part := range parts {
		assert.LessOrEqual(t, len(part), 12)
	}
}

func TestGetNotableFrames(t *testing.T) {
	frames := []Frame{
		{Index: 0, Notable: false, JPEG: []byte{1}},
		{Index: 1, Notable: true, JPEG: []byte{2}},
		{Index: 2, Notable: true, JPEG: []byte{3}},
		{Index: 3, Notable: true, JPEG: []byte{4}},
	}

	notable := getNotableFrames(frames, 2)
	assert.Len(t, notable, 2)
	assert.Equal(t, 1, notable[0].Index)
	assert.Equal(t, 2, notable[1].Index)
}

func TestGetNotableFrames_SkipsEmptyJPEG(t *testing.T) {
	frames := []Frame{
		{Index: 0, Notable: true, JPEG: nil},
		{Index: 1, Notable: true, JPEG: []byte{1}},
	}

	notable := getNotableFrames(frames, 5)
	assert.Len(t, notable, 1)
	assert.Equal(t, 1, notable[0].Index)
}

func TestParseChatID(t *testing.T) {
	id, err := parseChatID("12345")
	assert.NoError(t, err)
	assert.Equal(t, int64(12345), id)

	id, err = parseChatID("-100123456")
	assert.NoError(t, err)
	assert.Equal(t, int64(-100123456), id)

	_, err = parseChatID("@username")
	assert.Error(t, err)
}
