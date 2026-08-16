package chatController

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// HandleVoice transcribes a voice message and replies with the text.
func (c *Controller) HandleVoice(ctx context.Context, audio models.IncomingAudio) error {
	return c.transcribeAndReply(ctx, audio, models.VoiceFilename)
}
