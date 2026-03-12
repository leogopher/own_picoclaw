package videoanalyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
)

var unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// WriteObsidian creates a markdown note with embedded frame images in the vault.
func WriteObsidian(result *AnalyzeResult, cfg config.ObsidianConfig) error {
	if cfg.VaultPath == "" {
		return fmt.Errorf("obsidian vault_path not configured")
	}

	noteDir := filepath.Join(cfg.VaultPath, cfg.VideoNoteDir)
	assetsDir := filepath.Join(cfg.VaultPath, cfg.AssetsDir, sanitizeFilename(result.Meta.ID))

	if err := os.MkdirAll(noteDir, 0o755); err != nil {
		return fmt.Errorf("creating note dir: %w", err)
	}
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		return fmt.Errorf("creating assets dir: %w", err)
	}

	// Save notable frames as JPEGs
	var savedFrames []savedFrame
	for _, frame := range result.Frames {
		if !frame.Notable || len(frame.JPEG) == 0 {
			continue
		}
		filename := fmt.Sprintf("frame_%03d_%s.jpg", frame.Index, strings.ReplaceAll(frame.TimestampFormatted(), ":", "-"))
		path := filepath.Join(assetsDir, filename)
		if err := os.WriteFile(path, frame.JPEG, 0o644); err != nil {
			return fmt.Errorf("saving frame %d: %w", frame.Index, err)
		}
		// Relative path from vault root for wikilinks
		relPath, _ := filepath.Rel(cfg.VaultPath, path)
		savedFrames = append(savedFrames, savedFrame{
			frame:   frame,
			relPath: relPath,
		})
	}

	// Render markdown
	md := renderMarkdown(result, savedFrames)
	noteFilename := sanitizeFilename(result.Summary.Title) + ".md"
	notePath := filepath.Join(noteDir, noteFilename)

	if err := os.WriteFile(notePath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("writing note: %w", err)
	}

	return nil
}

type savedFrame struct {
	frame   Frame
	relPath string
}

func renderMarkdown(result *AnalyzeResult, savedFrames []savedFrame) string {
	var b strings.Builder
	meta := result.Meta
	summary := result.Summary

	// Frontmatter
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("title: %q\n", summary.Title))
	b.WriteString(fmt.Sprintf("source: %s\n", meta.WebpageURL))
	b.WriteString(fmt.Sprintf("channel: %s\n", meta.Channel))
	b.WriteString(fmt.Sprintf("duration: %s\n", meta.DurationFormatted()))
	b.WriteString(fmt.Sprintf("date: %s\n", time.Now().Format("2006-01-02")))
	if len(summary.Topics) > 0 {
		b.WriteString("tags:\n")
		for _, tag := range summary.Topics {
			b.WriteString(fmt.Sprintf("  - %s\n", strings.ReplaceAll(tag, " ", "-")))
		}
	}
	b.WriteString("---\n\n")

	// One-liner
	b.WriteString(fmt.Sprintf("> %s\n\n", summary.OneLiner))

	// Summary
	b.WriteString("## Summary\n\n")
	b.WriteString(summary.Summary)
	b.WriteString("\n\n")

	// Key Points
	if len(summary.KeyPoints) > 0 {
		b.WriteString("## Key Points\n\n")
		for _, point := range summary.KeyPoints {
			b.WriteString(fmt.Sprintf("- %s\n", point))
		}
		b.WriteString("\n")
	}

	// Action Items
	if len(summary.ActionItems) > 0 {
		b.WriteString("## Action Items\n\n")
		for _, item := range summary.ActionItems {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		b.WriteString("\n")
	}

	// Notable Frames
	if len(savedFrames) > 0 {
		b.WriteString("## Notable Frames\n\n")
		for _, sf := range savedFrames {
			b.WriteString(fmt.Sprintf("### %s\n\n", sf.frame.TimestampFormatted()))
			b.WriteString(fmt.Sprintf("![[%s]]\n\n", sf.relPath))
			b.WriteString(sf.frame.Description)
			if sf.frame.KeyText != "" {
				b.WriteString(fmt.Sprintf("\n\n> Text: %s", sf.frame.KeyText))
			}
			b.WriteString("\n\n")
		}
	}

	// Timestamps
	if len(summary.Timestamps) > 0 {
		b.WriteString("## Timestamps\n\n")
		for _, ts := range summary.Timestamps {
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", ts.Time, ts.Note))
		}
		b.WriteString("\n")
	}

	// Metadata footer
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("*Analyzed by PicoClaw on %s*\n", time.Now().Format("2006-01-02 15:04")))

	return b.String()
}

func sanitizeFilename(name string) string {
	s := unsafeFilenameChars.ReplaceAllString(name, "_")
	if len(s) > 200 {
		s = s[:200]
	}
	return strings.TrimSpace(s)
}
