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

// transcribeAndReply is the shared scenario behind HandleVoice and
// HandleVideoNote: reject oversized files, download and transcribe the audio
// and reply with the recognized text, split into Telegram-sized chunks.
// Expected outcomes (too large, no speech) are reported to the chat and are
// not errors; provider failures are reported to the chat as well and returned
// wrapped, so the wiring can log them.
func (c *Controller) transcribeAndReply(ctx context.Context, audio models.IncomingAudio, filename string) error {
	if audio.FileSize > models.TelegramMaxFileSize {
		return c.sendReply(ctx, audio, models.MsgFileTooLarge)
	}

	text, err := c.transcribeAudio(ctx, audio, filename)
	if err != nil {
		return errors.Join(err, c.sendFailureReply(ctx, audio))
	}

	if strings.TrimSpace(text) == "" {
		return c.sendReply(ctx, audio, models.MsgNoSpeech)
	}

	// Every chunk replies to the original message: the provider does not
	// expose the IDs of the messages it sends, so the chunks cannot be
	// chained to each other.
	for _, chunk := range models.SplitText(text, models.TelegramMessageLimit) {
		if err := c.sendReply(ctx, audio, chunk); err != nil {
			// The chunks before this one are already in the chat: without a
			// notice the user reads a transcript that stops mid-sentence.
			return errors.Join(err, c.sendFailureReply(ctx, audio))
		}
	}

	return nil
}

// sendFailureReply tells the user the transcription did not go through. The
// notice detaches from the caller's cancellation: during shutdown that context
// is already canceled and the user would be left without any answer at all.
func (c *Controller) sendFailureReply(ctx context.Context, audio models.IncomingAudio) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), models.FailureReplyTimeout)
	defer cancel()

	return c.sendReply(ctx, audio, models.MsgTranscribeFailed)
}

// transcribeAudio downloads the audio file and sends it to the STT service.
func (c *Controller) transcribeAudio(ctx context.Context, audio models.IncomingAudio, filename string) (string, error) {
	file, err := c.telegram.DownloadFile(ctx, audio.FileID)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrDownloadAudio, err)
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			slog.Warn("failed to close downloaded audio", "err", closeErr, "chat_id", audio.ChatID)
		}
	}()

	text, err := c.stt.TranscribeAudio(ctx, file, filename)
	if err != nil {
		return "", fmt.Errorf("%w: %w", controllers.ErrTranscribeAudio, err)
	}

	return text, nil
}

// sendReply sends text as a reply to the original message.
func (c *Controller) sendReply(ctx context.Context, audio models.IncomingAudio, text string) error {
	if err := c.telegram.SendReply(ctx, audio.ChatID, audio.MessageID, text); err != nil {
		return fmt.Errorf("%w: %w", controllers.ErrSendReply, err)
	}

	return nil
}
