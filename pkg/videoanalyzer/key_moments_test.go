package videoanalyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCombinedFilter(t *testing.T) {
	// No key timestamps — scene detection + interval safety net
	f := buildCombinedFilter(0.3, nil)
	assert.Contains(t, f, "gt(scene")
	assert.Contains(t, f, "prev_selected_t")
	assert.Contains(t, f, "showinfo")

	// With key timestamps
	f = buildCombinedFilter(0.3, []float64{10.0, 60.5})
	assert.Contains(t, f, "between(t")
	assert.Contains(t, f, "9.5")
	assert.Contains(t, f, "10.5")
	assert.Contains(t, f, "60")
	assert.Contains(t, f, "61")
}

func TestDeduplicateFrames(t *testing.T) {
	frames := []Frame{
		{Timestamp: 0},
		{Timestamp: 0.5},  // too close to 0
		{Timestamp: 1.0},  // too close to 0
		{Timestamp: 3.0},  // kept (>2s gap)
		{Timestamp: 3.5},  // too close to 3.0
		{Timestamp: 10.0}, // kept
	}

	result := deduplicateFrames(frames, 2.0)
	assert.Len(t, result, 3)
	assert.Equal(t, 0.0, result[0].Timestamp)
	assert.Equal(t, 3.0, result[1].Timestamp)
	assert.Equal(t, 10.0, result[2].Timestamp)
}

func TestDeduplicateFrames_Empty(t *testing.T) {
	assert.Empty(t, deduplicateFrames(nil, 2.0))
	assert.Len(t, deduplicateFrames([]Frame{{Timestamp: 1}}, 2.0), 1)
}
