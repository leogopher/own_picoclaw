package videoanalyzer

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/sipeed/picoclaw/pkg/config"
)

const telegramMaxMessageLen = 4096

// DeliverTelegram sends the analysis result to a Telegram chat.
func DeliverTelegram(ctx context.Context, result *AnalyzeResult, cfg config.VideoAnalyzerTelegram) error {
	if cfg.BotToken == "" {
		return fmt.Errorf("telegram bot_token not configured")
	}
	if cfg.ChatID == "" {
		return fmt.Errorf("telegram chat_id not configured")
	}

	bot, err := telego.NewBot(cfg.BotToken, telego.WithDiscardLogger())
	if err != nil {
		return fmt.Errorf("creating telegram bot: %w", err)
	}

	chatID := tu.ID(0)
	// Parse chat ID — could be numeric or @username
	if id, err := parseChatID(cfg.ChatID); err == nil {
		chatID = tu.ID(id)
	} else {
		chatID = tu.Username(cfg.ChatID)
	}

	// Send summary message
	html := formatTelegramHTML(result)
	parts := splitMessage(html, telegramMaxMessageLen)
	for _, part := range parts {
		msg := tu.Message(chatID, part).WithParseMode("HTML")
		if _, err := bot.SendMessage(ctx, msg); err != nil {
			return fmt.Errorf("sending telegram message: %w", err)
		}
	}

	// Send notable frames as photos
	if cfg.IncludeFrames {
		notableFrames := getNotableFrames(result.Frames, cfg.MaxFramesInDigest)
		for _, frame := range notableFrames {
			caption := fmt.Sprintf("[%s] %s", frame.TimestampFormatted(), frame.Description)
			if len(caption) > 1024 {
				caption = caption[:1021] + "..."
			}
			photo := &telego.SendPhotoParams{
				ChatID:  chatID,
				Photo:   tu.FileFromReader(bytes.NewReader(frame.JPEG), "frame.jpg"),
				Caption: caption,
			}
			if _, err := bot.SendPhoto(ctx, photo); err != nil {
				return fmt.Errorf("sending telegram photo: %w", err)
			}
		}
	}

	return nil
}

func parseChatID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}

// formatTelegramHTML builds the summary message in HTML format.
func formatTelegramHTML(result *AnalyzeResult) string {
	var b strings.Builder
	meta := result.Meta
	summary := result.Summary

	b.WriteString(fmt.Sprintf("<b>%s</b>\n", escapeHTML(summary.Title)))
	b.WriteString(fmt.Sprintf("<i>%s</i>\n\n", escapeHTML(summary.OneLiner)))

	b.WriteString(fmt.Sprintf("Channel: %s\n", escapeHTML(meta.Channel)))
	b.WriteString(fmt.Sprintf("Duration: %s\n", meta.DurationFormatted()))
	if meta.WebpageURL != "" {
		b.WriteString(fmt.Sprintf("Link: %s\n", meta.WebpageURL))
	}
	b.WriteString("\n")

	b.WriteString("<b>Summary</b>\n")
	b.WriteString(escapeHTML(summary.Summary))
	b.WriteString("\n\n")

	if len(summary.KeyPoints) > 0 {
		b.WriteString("<b>Key Points</b>\n")
		for _, point := range summary.KeyPoints {
			b.WriteString(fmt.Sprintf("• %s\n", escapeHTML(point)))
		}
		b.WriteString("\n")
	}

	if len(summary.ActionItems) > 0 {
		b.WriteString("<b>Action Items</b>\n")
		for _, item := range summary.ActionItems {
			b.WriteString(fmt.Sprintf("• %s\n", escapeHTML(item)))
		}
		b.WriteString("\n")
	}

	if len(summary.Topics) > 0 {
		b.WriteString(fmt.Sprintf("Tags: %s\n", escapeHTML(strings.Join(summary.Topics, ", "))))
	}

	return b.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// splitMessage splits a message into chunks that fit within maxLen.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			parts = append(parts, text)
			break
		}
		// Find a good split point (newline)
		splitAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > maxLen/2 {
			splitAt = idx + 1
		}
		parts = append(parts, text[:splitAt])
		text = text[splitAt:]
	}
	return parts
}

func getNotableFrames(frames []Frame, max int) []Frame {
	if max <= 0 {
		max = 5
	}
	var notable []Frame
	for i := range frames {
		if frames[i].Notable && len(frames[i].JPEG) > 0 {
			notable = append(notable, frames[i])
			if len(notable) >= max {
				break
			}
		}
	}
	return notable
}
