package videoanalyzer

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestJPEG creates a minimal valid JPEG byte sequence.
func makeTestJPEG(payload byte) []byte {
	return []byte{0xFF, 0xD8, payload, payload, payload, 0xFF, 0xD9}
}

func TestSplitJPEGs(t *testing.T) {
	// Build stream of 3 concatenated JPEGs
	j1 := makeTestJPEG(0x01)
	j2 := makeTestJPEG(0x02)
	j3 := makeTestJPEG(0x03)

	stream := append(append(j1, j2...), j3...)

	frames, err := splitJPEGs(bytes.NewReader(stream), 10)
	require.NoError(t, err)
	assert.Len(t, frames, 3)
	assert.Equal(t, j1, frames[0].JPEG)
	assert.Equal(t, j2, frames[1].JPEG)
	assert.Equal(t, j3, frames[2].JPEG)
}

func TestSplitJPEGs_MaxFrames(t *testing.T) {
	j1 := makeTestJPEG(0x01)
	j2 := makeTestJPEG(0x02)
	j3 := makeTestJPEG(0x03)

	stream := append(append(j1, j2...), j3...)

	frames, err := splitJPEGs(bytes.NewReader(stream), 2)
	require.NoError(t, err)
	assert.Len(t, frames, 2)
}

func TestSplitJPEGs_Empty(t *testing.T) {
	frames, err := splitJPEGs(bytes.NewReader(nil), 10)
	require.NoError(t, err)
	assert.Empty(t, frames)
}

func TestSplitJPEGs_Garbage(t *testing.T) {
	// Garbage data with no JPEG markers
	garbage := []byte{0x00, 0x01, 0x02, 0x03}
	frames, err := splitJPEGs(bytes.NewReader(garbage), 10)
	require.NoError(t, err)
	assert.Empty(t, frames)
}

func TestSplitJPEGs_PartialJPEG(t *testing.T) {
	// SOI but no EOI — incomplete frame
	partial := []byte{0xFF, 0xD8, 0x01, 0x02}
	frames, err := splitJPEGs(bytes.NewReader(partial), 10)
	require.NoError(t, err)
	assert.Empty(t, frames)
}

func TestJpegQualityToFFmpeg(t *testing.T) {
	assert.Equal(t, 2, jpegQualityToFFmpeg(100))
	assert.Equal(t, 31, jpegQualityToFFmpeg(1))
	assert.Equal(t, 6, jpegQualityToFFmpeg(85)) // Default quality
	assert.Equal(t, 6, jpegQualityToFFmpeg(0))  // 0 maps to default 85
}

func TestExtractOneJPEG(t *testing.T) {
	j := makeTestJPEG(0xAA)
	extra := []byte{0x00, 0x01}

	buf := append(j, extra...)
	jpeg, rest, ok := extractOneJPEG(buf)
	assert.True(t, ok)
	assert.Equal(t, j, jpeg)
	assert.Equal(t, extra, rest)
}

func TestExtractOneJPEG_NoSOI(t *testing.T) {
	_, _, ok := extractOneJPEG([]byte{0x00, 0x01})
	assert.False(t, ok)
}

func TestFormatTimestamp(t *testing.T) {
	assert.Equal(t, "0:00", formatTimestamp(0))
	assert.Equal(t, "1:05", formatTimestamp(65))
	assert.Equal(t, "1:01:01", formatTimestamp(3661))
}
