package videoanalyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestWriteObsidian(t *testing.T) {
	tmpDir := t.TempDir()

	result := &AnalyzeResult{
		Meta: &VideoMeta{
			ID:         "abc123",
			Title:      "Test Video",
			Channel:    "Test Channel",
			Duration:   300,
			WebpageURL: "https://youtube.com/watch?v=abc123",
		},
		Frames: []Frame{
			{Index: 0, Timestamp: 10, JPEG: []byte{0xFF, 0xD8, 0x01, 0xFF, 0xD9}, Description: "Title slide", Notable: true, KeyText: "Hello"},
			{Index: 1, Timestamp: 60, JPEG: []byte{0xFF, 0xD8, 0x02, 0xFF, 0xD9}, Description: "Code example", Notable: false},
		},
		Summary: &VideoSummary{
			Title:     "Test Video Summary",
			OneLiner:  "A test video about testing",
			Summary:   "This is a detailed summary.",
			KeyPoints: []string{"Point 1", "Point 2"},
			Topics:    []string{"testing", "go"},
		},
	}

	cfg := config.ObsidianConfig{
		VaultPath:    tmpDir,
		VideoNoteDir: "Videos",
		AssetsDir:    "assets/va",
	}

	err := WriteObsidian(result, cfg)
	require.NoError(t, err)

	// Check note file exists
	notePath := filepath.Join(tmpDir, "Videos", "Test Video Summary.md")
	_, err = os.Stat(notePath)
	require.NoError(t, err)

	// Check note content
	content, err := os.ReadFile(notePath)
	require.NoError(t, err)

	md := string(content)
	assert.Contains(t, md, "title: \"Test Video Summary\"")
	assert.Contains(t, md, "source: https://youtube.com/watch?v=abc123")
	assert.Contains(t, md, "A test video about testing")
	assert.Contains(t, md, "Point 1")
	assert.Contains(t, md, "![[")

	// Check notable frame was saved (only index 0 is notable)
	assetsDir := filepath.Join(tmpDir, "assets/va/abc123")
	entries, err := os.ReadDir(assetsDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "only notable frames should be saved")
}

func TestWriteObsidian_NoVaultPath(t *testing.T) {
	result := &AnalyzeResult{
		Meta: &VideoMeta{ID: "x"},
		Summary: &VideoSummary{Title: "T"},
	}
	err := WriteObsidian(result, config.ObsidianConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vault_path not configured")
}

func TestSanitizeFilename(t *testing.T) {
	assert.Equal(t, "hello_world", sanitizeFilename("hello/world"))
	assert.Equal(t, "a_b_c", sanitizeFilename("a:b?c"))
	assert.Equal(t, "normal-name", sanitizeFilename("normal-name"))
}

func TestRenderMarkdown(t *testing.T) {
	result := &AnalyzeResult{
		Meta: &VideoMeta{
			Channel:    "TestChan",
			Duration:   120,
			WebpageURL: "https://youtube.com/watch?v=x",
		},
		Summary: &VideoSummary{
			Title:     "My Summary",
			OneLiner:  "Short line",
			Summary:   "Detailed.",
			KeyPoints: []string{"Key 1"},
			Topics:    []string{"topic one"},
			ActionItems: []string{"Do this"},
			Timestamps: []TimestampNote{{Time: "1:23", Note: "important"}},
		},
	}

	md := renderMarkdown(result, nil)
	assert.Contains(t, md, "---")
	assert.Contains(t, md, "title: \"My Summary\"")
	assert.Contains(t, md, "- topic-one")
	assert.Contains(t, md, "- [ ] Do this")
	assert.Contains(t, md, "**1:23**")
}
