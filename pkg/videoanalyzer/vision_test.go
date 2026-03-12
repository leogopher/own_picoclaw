package videoanalyzer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestAnalyzeFrames_BatchGrouping(t *testing.T) {
	// Mock server that returns frame analyses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []struct {
				Content any `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)

		// Count images in the request to determine batch size
		analyses := []frameAnalysis{
			{Description: "test frame", Notable: true, KeyText: "hello"},
			{Description: "another frame", Notable: false, KeyText: ""},
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": mustJSON(analyses),
					},
					"finish_reason": "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	frames := []Frame{
		{Index: 0, JPEG: makeTestJPEG(0x01), Timestamp: 10},
		{Index: 1, JPEG: makeTestJPEG(0x02), Timestamp: 20},
	}

	meta := &VideoMeta{Title: "Test", Uploader: "Tester", Duration: 120}

	cfg := config.VideoAnalyzerProvider{
		Model:            "test-model",
		BaseURL:          server.URL,
		APIKey:           "test-key",
		MaxFramesPerBatch: 2,
		TimeoutSeconds:   10,
	}

	result, err := AnalyzeFrames(context.Background(), frames, meta, cfg)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "test frame", result[0].Description)
	assert.True(t, result[0].Notable)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
