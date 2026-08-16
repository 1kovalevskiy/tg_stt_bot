package chatController

import (
	"context"

	"github.com/1kovalevskiy/tg_stt_bot/internal/models"
)

// voiceFilename is the file name reported to the STT service for voice
// messages. parakeet detects the container by content, so the extension is
// informational only.
const voiceFilename = "voice.oga"

// HandleVoice transcribes a voice message and replies with the text.
func (c *Controller) HandleVoice(ctx context.Context, audio models.IncomingAudio) error {
	return c.transcribeAndReply(ctx, audio, voiceFilename)
}
