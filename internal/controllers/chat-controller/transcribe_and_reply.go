package chatController

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/1kovalevskiy/tg_stt_bot/internal/controllers"
	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

const (
	// maxFileSize is the Bot API download limit: getFile refuses bigger files,
	// so they are rejected before any download attempt.
	maxFileSize = 20 << 20 // 20 MB

	msgFileTooLarge     = "Файл слишком большой, я не могу его скачать."
	msgNoSpeech         = "Речь не распознана."
	msgTranscribeFailed = "Не удалось расшифровать сообщение."
)

// transcribeAndReply is the shared scenario behind HandleVoice and
// HandleVideoNote: reject oversized files, download and transcribe the audio
// and reply with the recognized text, split into Telegram-sized chunks.
// Expected outcomes (too large, no speech) are reported to the chat and are
// not errors; provider failures are reported to the chat as well and returned
// wrapped, so the wiring can log them.
func (c *Controller) transcribeAndReply(ctx context.Context, audio models.IncomingAudio, filename string) error {
	if audio.FileSize > maxFileSize {
		return c.reply(ctx, audio, msgFileTooLarge)
	}

	text, err := c.transcribe(ctx, audio, filename)
	if err != nil {
		return errors.Join(err, c.reply(ctx, audio, msgTranscribeFailed))
	}

	if strings.TrimSpace(text) == "" {
		return c.reply(ctx, audio, msgNoSpeech)
	}

	// Every chunk replies to the original message: the provider does not
	// expose the IDs of the messages it sends, so the chunks cannot be
	// chained to each other.
	for _, chunk := range models.SplitText(text, models.TelegramMessageLimit) {
		if err := c.reply(ctx, audio, chunk); err != nil {
			return err
		}
	}

	return nil
}

// transcribe downloads the audio file and sends it to the STT service.
func (c *Controller) transcribe(ctx context.Context, audio models.IncomingAudio, filename string) (string, error) {
	file, err := c.telegram.DownloadFile(ctx, audio.FileID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrDownloadAudio, err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close downloaded audio", "err", closeErr, "chat_id", audio.ChatID)
		}
	}()

	text, err := c.stt.Transcribe(ctx, file, filename)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrTranscribeAudio, err)
	}

	return text, nil
}

// reply sends text as a reply to the original message.
func (c *Controller) reply(ctx context.Context, audio models.IncomingAudio, text string) error {
	if err := c.telegram.SendReply(ctx, audio.ChatID, audio.MessageID, text); err != nil {
		return fmt.Errorf("%w: %w", controllers.ErrSendReply, err)
	}

	return nil
}
