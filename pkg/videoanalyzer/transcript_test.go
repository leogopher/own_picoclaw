package videoanalyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseJSON3(t *testing.T) {
	data := []byte(`{
		"events": [
			{
				"tStartMs": 1000,
				"dDurationMs": 2500,
				"segs": [
					{"utf8": "Hello "},
					{"utf8": "world"}
				]
			},
			{
				"tStartMs": 5000,
				"dDurationMs": 3000,
				"segs": [
					{"utf8": "Second line"}
				]
			},
			{
				"tStartMs": 8000,
				"dDurationMs": 1000,
				"segs": [
					{"utf8": "  "}
				]
			}
		]
	}`)

	lines, err := ParseJSON3(data)
	require.NoError(t, err)

	// Empty segments should be filtered out
	assert.Len(t, lines, 2)

	assert.Equal(t, 1.0, lines[0].Start)
	assert.Equal(t, 2.5, lines[0].Duration)
	assert.Equal(t, "Hello world", lines[0].Text)

	assert.Equal(t, 5.0, lines[1].Start)
	assert.Equal(t, 3.0, lines[1].Duration)
	assert.Equal(t, "Second line", lines[1].Text)
}

func TestParseJSON3_Empty(t *testing.T) {
	data := []byte(`{"events": []}`)
	lines, err := ParseJSON3(data)
	require.NoError(t, err)
	assert.Empty(t, lines)
}

func TestParseJSON3_Invalid(t *testing.T) {
	_, err := ParseJSON3([]byte(`not json`))
	assert.Error(t, err)
}
