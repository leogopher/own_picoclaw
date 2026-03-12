package videoanalyzer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"standard watch URL", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"short URL", "https://youtu.be/dQw4w9WgXcQ", false},
		{"mobile URL", "https://m.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"shorts URL", "https://www.youtube.com/shorts/dQw4w9WgXcQ", false},
		{"no www", "https://youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"http", "http://www.youtube.com/watch?v=dQw4w9WgXcQ", false},
		{"not youtube", "https://example.com/watch?v=abc", true},
		{"empty", "", true},
		{"random string", "not a url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVideoMeta_DurationFormatted(t *testing.T) {
	tests := []struct {
		duration float64
		want     string
	}{
		{0, "0m00s"},
		{65, "1m05s"},
		{3661, "1h01m01s"},
		{7200, "2h00m00s"},
	}
	for _, tt := range tests {
		meta := &VideoMeta{Duration: tt.duration}
		assert.Equal(t, tt.want, meta.DurationFormatted())
	}
}

func TestParseMetadataJSON(t *testing.T) {
	// Simulate yt-dlp --dump-json output
	jsonData := `{
		"id": "dQw4w9WgXcQ",
		"title": "Test Video",
		"description": "A test video",
		"duration": 212.5,
		"uploader": "Test Channel",
		"upload_date": "20240101",
		"view_count": 1000000,
		"channel": "TestChannel",
		"webpage_url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"categories": ["Music"],
		"tags": ["test", "video"]
	}`

	var meta VideoMeta
	err := json.Unmarshal([]byte(jsonData), &meta)
	require.NoError(t, err)

	assert.Equal(t, "dQw4w9WgXcQ", meta.ID)
	assert.Equal(t, "Test Video", meta.Title)
	assert.Equal(t, 212.5, meta.Duration)
	assert.Equal(t, "TestChannel", meta.Channel)
	assert.Equal(t, []string{"Music"}, meta.Categories)
	assert.Equal(t, "3m32s", meta.DurationFormatted())
}
